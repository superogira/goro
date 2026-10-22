package login

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

type Screen struct {
	bindings            Bindings
	selectedLoginServer int
	phase               loginPhase
	status              string
	packets             []string
	autoAttempted       bool
	autoCharAttempted   bool
	loginPending        bool
	charSelectPending   bool
	fade                loginFadeState
	username            string
	password            string
	keepID              bool
	background          *render.Image
	bgTiles             []*render.Image
	bgSource            string
	bgLoaded            bool
	bgmStarted          bool
	selectedSlot        int
	maxSlots            int
	charPreviewImages   map[uint32]image.Image
	charWindow          *render.Image
	charBox             *render.Image
	accountStep         loginAccountStep
	serviceWindow       *gameui.ServiceWindow
	loginWindow         *gameui.LoginWindow
	charSelectWindow    *gameui.CharacterSelectWindow
	charCreateWindow    *gameui.CharacterCreateWindow
	charDeleteConfirm   gameui.ConfirmModal
	charDeletePrompt    gameui.TextPromptWindow
	deleteCharID        uint32
	loginPingActive     bool
	nextLoginPing       time.Time
	charPingActive      bool
	nextCharPing        time.Time
	create              charCreateState
	quitConfirm         gameui.ConfirmModal
	disconnectDialog    gameui.ConfirmModal
	mapError            string

	// Web (DOM) character create/select layer state; only touched on
	// js/wasm builds, but kept unconditionally so the fields stay with
	// the Screen they belong to.
	createWebSyncKey    string
	createWebBG         string
	createPreviewWebKey charCreatePreviewKey
	createPreviewWebURL string
	charWebSyncKey      string
	charWebBG           string
	charPreviewWebURLs  map[uint32]string
	charDeleteWebStep   int
	charDeleteWebName   string
}

type loginPhase int

const (
	loginPhaseAccount loginPhase = iota
	loginPhaseCharacter
	loginPhaseCreate
)

type loginAccountStep int

const (
	loginAccountConnection loginAccountStep = iota
	loginAccountCredentials
	loginAccountCharacterService
	loginAccountCharacterConnecting
)

type loginFadePhase int

const (
	loginFadeNone loginFadePhase = iota
	loginFadeOut
	loginFadeHold
	loginFadeIn
)

type loginFadeState struct {
	phase         loginFadePhase
	started       time.Time
	coveredFrames int
	target        loginPhase
	hasTarget     bool
	enterWorld    bool
}

const loginTransitionDuration = 500 * time.Millisecond

const loginWorldHandoffFrames = 1

const loginConfirmSFX = "버튼소리.wav"

const loginServerPingInterval = 10 * time.Second

const charServerPingInterval = 10 * time.Second

func New(bindings Bindings) *Screen {
	if bindings.Console == nil {
		bindings.Console = &gameui.ChatConsole{}
	}
	return &Screen{bindings: bindings, status: "select a server", maxSlots: 9}
}

func NewCharacterSelect(ctx client.Context, bindings Bindings) *Screen {
	mode := New(bindings)
	mode.phase = loginPhaseCharacter
	mode.fade = loginFadeState{phase: loginFadeIn}
	mode.status = "select a character"
	mode.autoAttempted = true
	mode.prepareCharacterSelectFromSession(ctx)
	return mode
}

func (m *Screen) Name() string {
	return "login"
}

func (m *Screen) Enter(ctx client.Context) {
	m.clearLoginWindows(ctx)
	m.keepID = ctx.Config.Login.KeepID
	savedUsername := ctx.Config.Login.SavedUsername
	if ctx.Session != nil {
		ctx.Session.BattleMode = false
		m.keepID = ctx.Session.KeepLoginID
		savedUsername = ctx.Session.SavedUsername
	}
	if m.username == "" {
		m.username = ctx.Config.Login.Username
		if m.username == "" && m.keepID {
			m.username = savedUsername
		}
	}
	if m.password == "" {
		m.password = ctx.Config.Login.Password
	}
	if !ctx.Config.Headless {
		m.loadBackground(ctx)
		m.loadCharacterSelectSkin(ctx)
	}
	m.playLoginBGM(ctx)
	if m.phase == loginPhaseCharacter {
		m.prepareCharacterSelectFromSession(ctx)
		m.showCharacterSelectWindow(ctx)
		m.reconnectCharacterServer(ctx)
	}
	if m.phase == loginPhaseAccount && len(loginConnections(ctx)) == 0 {
		m.status = "no login servers discovered"
	}
	if m.fade.phase == loginFadeIn && m.fade.started.IsZero() {
		// Returning from the world can reconnect and rebuild the character UI
		// here. Begin fading in only once the destination is ready to draw.
		m.fade.started = time.Now()
	}
	if m.mapError != "" {
		m.disconnectDialog.OpenAlert(ctx, "Map unavailable", m.mapError, func() { m.updateAccountWindow(ctx) })
		m.mapError = ""
	}
}

// Update reports when the covered login screen is ready to hand off to the world.
func (m *Screen) Update(ctx client.Context) (bool, error) {
	now := time.Now()
	if m.updateFade(ctx, now) {
		return true, nil
	}

	conns := loginConnections(ctx)
	fading := m.fade.phase != loginFadeNone
	if !fading {
		if m.updateQuitConfirm(ctx) {
			// The confirmation modal is modal: no keyboard or mouse input should
			// leak into the login form, character list, or creation controls.
		} else if m.updateDisconnectDialog(ctx) {
		} else if m.updateCharacterDelete(ctx) {
		} else if m.updatePhaseEscape(ctx, now) {
		} else if m.phase == loginPhaseCreate {
			m.updateCharacterCreateInput(ctx)
		} else if m.phase == loginPhaseCharacter {
			m.updateCharacterSelectInput(ctx)
		} else {
			m.updateAccountInput(ctx)
		}

	}
	if fading && m.phase == loginPhaseCharacter {
		m.showCharacterSelectWindow(ctx)
	}

	m.maybeSendLoginServerPing(ctx, now)
	m.maybeSendCharServerPing(ctx, now)

	if ctx.Config.Login.AutoLogin && !m.autoAttempted {
		m.autoAttempted = true
		index := ctx.Config.Login.ServerSlot
		if index < 0 || index >= len(conns) {
			return false, fmt.Errorf("--server-slot %d is unavailable: clientinfo.xml contains %d login servers (slots start at 0)", index, len(conns))
		}
		m.selectedLoginServer = index
		if conn, ok := m.selectedLoginConnection(ctx); ok {
			m.connectAndMaybeLogin(ctx, conn, false)
		}
	}
	if len(conns) == 0 {
		return false, nil
	}

	loginRefused := false
	for _, pkt := range ctx.Network.DrainPackets() {
		glog.Debugf("recv packet 0x%04X len=%d", pkt.ID, len(pkt.Data))
		if HandleDisconnectPacket(ctx, &m.disconnectDialog, pkt) {
			m.loginPending = false
			continue
		}
		m.packets = append(m.packets, pkt.String())
		if pkt.ID == 0x006C {
			// HC_REFUSE_ENTER can also reject a selection without disconnecting.
			m.charSelectPending = false
			continue
		}
		if pkt.ID == 0x006A {
			refusal, err := network.ParseAccountRefuseLogin(pkt)
			if err != nil {
				m.loginPending = false
				m.packets = append(m.packets, "parse AC_REFUSE_LOGIN: "+err.Error())
			} else {
				m.applyAccountRefuseLogin(ctx, refusal)
				loginRefused = true
			}
			continue
		}
		if m.bindings.ApplyWorldPacket != nil {
			if handled, err := m.bindings.ApplyWorldPacket(ctx, pkt); handled {
				if err != nil {
					m.packets = append(m.packets, err.Error())
				}
				continue
			}
		}
		if change, ok, err := network.ParseMapChange(pkt); err != nil {
			m.packets = append(m.packets, "parse ZC_NPCACK_MAPMOVE: "+err.Error())
		} else if ok {
			m.applyLoginMapChange(ctx, change)
			continue
		}
		if pkt.ID == 0x0069 {
			login, err := network.ParseAccountAcceptLogin(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse AC_ACCEPT_LOGIN: "+err.Error())
			} else {
				if err := m.applyAccountAcceptLogin(ctx, login); err != nil {
					return false, err
				}
			}
		}
		if pkt.ID == 0x006B {
			list, err := network.ParseCharList(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_ENTER: "+err.Error())
			} else {
				ctx.Session.Characters = convertCharacters(list.Characters)
				m.enableCharServerPing(now)
				m.status = fmt.Sprintf("char list: %d characters (%s)", len(list.Characters), list.Layout)
				glog.Debugf("char list characters=%d layout=%s", len(list.Characters), list.Layout)
				for _, character := range list.Characters {
					m.packets = append(m.packets, fmt.Sprintf("char slot=%d gid=%d name=%s lv=%d job=%d", character.Slot, character.ID, character.Name, character.Level, character.Job))
				}
				m.maxSlots = 9
				m.selectedSlot = 0
				if len(list.Characters) > 0 {
					m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
					m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
					if configured := ctx.Config.Login.CharSlot; configured >= 0 {
						m.selectedSlot = clampCharacterSlot(configured, m.maxSlots)
					}
					m.status = "select a character"
				} else {
					m.status = "no characters"
				}
				if m.autoSelectCharacter(ctx) {
					continue
				}
				m.startPhaseFade(loginPhaseCharacter, time.Now())
				if m.phase == loginPhaseCharacter && m.charSelectWindow != nil {
					// A returning character-select mode already has a window built
					// from the saved session. Apply the refreshed server list during
					// Update so gogpu/ui can lay out the replacement before Draw.
					m.showCharacterSelectWindow(ctx)
				}
			}
		}
		if pkt.ID == 0x006D {
			character, err := network.ParseMakeCharacterAccept(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_MAKECHAR: "+err.Error())
			} else {
				created := convertCharacter(character)
				ctx.Session.Characters = upsertCharacter(ctx.Session.Characters, created)
				m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
				m.selectedSlot = int(created.Slot)
				m.create = charCreateState{}
				m.charPreviewImages = nil
				m.status = fmt.Sprintf("created character %s", created.Name)
				glog.Infof("character created slot=%d id=%d name=%s", created.Slot, created.ID, created.Name)
				m.startPhaseFade(loginPhaseCharacter, time.Now())
			}
			continue
		}
		if pkt.ID == 0x006E {
			code, err := network.ParseMakeCharacterRefuse(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_REFUSE_MAKECHAR: "+err.Error())
			} else {
				m.status = describeMakeCharacterRefuse(code)
				glog.Debugf("character creation refused code=%d", code)
			}
			continue
		}
		if pkt.ID == 0x006F {
			if err := network.ParseDeleteCharacterAccept(pkt); err != nil {
				m.packets = append(m.packets, "parse HC_ACCEPT_DELETECHAR: "+err.Error())
			} else {
				m.applyDeleteCharacterAccept(ctx)
			}
			continue
		}
		if pkt.ID == 0x0070 {
			code, err := network.ParseDeleteCharacterRefuse(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_REFUSE_DELETECHAR: "+err.Error())
			} else {
				m.applyDeleteCharacterRefuse(ctx, code)
			}
			continue
		}
		if pkt.ID == 0x0071 {
			zone, err := network.ParseZoneServerNotify(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse HC_NOTIFY_ZONESVR: "+err.Error())
			} else {
				ctx.Session.CharID = zone.CharID
				ctx.Session.Zone = session.ZoneServer{
					Address: zone.Address,
					Port:    zone.Port,
					MapName: zone.MapName,
				}
				ctx.World.MapName = zone.MapName
				m.status = fmt.Sprintf("zone server: %s %s:%d", zone.MapName, zone.Address, zone.Port)
				glog.Debugf("zone server map=%s addr=%s port=%d char_id=%d", zone.MapName, zone.Address, zone.Port, zone.CharID)
				m.connectMapServer(ctx, zone)
			}
		}
		if pkt.ID == 0x0073 || pkt.ID == 0x02EB {
			enter, err := network.ParseMapAcceptEnter(pkt)
			if err != nil {
				m.packets = append(m.packets, "parse ZC_ACCEPT_ENTER: "+err.Error())
			} else {
				if m.bindings.EnterMap != nil {
					m.bindings.EnterMap(ctx, enter)
				}
				m.status = fmt.Sprintf("entered map %s at %d,%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				glog.Infof("entered map=%s x=%d y=%d dir=%d tick=%d", ctx.World.MapName, enter.X, enter.Y, enter.Dir, enter.ServerTick)
				m.startWorldFade(time.Now())
			}
		}
		if len(m.packets) > 8 {
			m.packets = m.packets[len(m.packets)-8:]
		}
	}
	networkErrors := ctx.Network.DrainErrors()
	if loginRefused {
		// The login server closes this short-lived connection after sending
		// AC_REFUSE_LOGIN. Its EOF is part of the refusal, not a second error.
		networkErrors = nil
	}
	if len(networkErrors) > 0 {
		m.loginPending = false
	}
	if HandleNetworkDisconnectErrors(ctx, &m.disconnectDialog, networkErrors) {
		return false, nil
	}
	for _, err := range networkErrors {
		glog.Errorf("network frame error: %v", err)
		m.packets = append(m.packets, "frame error: "+err.Error())
		if len(m.packets) > 8 {
			m.packets = m.packets[len(m.packets)-8:]
		}
	}

	if m.updateFade(ctx, time.Now()) {
		return true, nil
	}
	return false, nil
}

func (m *Screen) applyLoginMapChange(ctx client.Context, change network.MapChange) {
	if m.bindings.ChangeMap != nil {
		m.bindings.ChangeMap(ctx, change)
	}
	m.status = fmt.Sprintf("map change: %s at %d,%d", change.MapName, change.X, change.Y)
	glog.Debugf("login map change map=%s x=%d y=%d server_move=%t addr=%s port=%d", change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port)
	if change.ServerMove {
		ctx.Session.Zone.Address = change.Address
		ctx.Session.Zone.Port = change.Port
		m.connectMapServer(ctx, network.ZoneServerNotify{
			CharID:  ctx.Session.CharID,
			MapName: change.MapName,
			Address: change.Address,
			Port:    change.Port,
		})
		return
	}
	if m.fade.enterWorld {
		return
	}
	m.startWorldFade(time.Now())
}

func (m *Screen) Draw(ctx client.Context, screen *render.Frame) {
	m.drawBackground(ctx, screen)
}

func (m *Screen) DrawOverlay(ctx client.Context, screen *render.Frame) {
	now := time.Now()
	m.drawFade(ctx, screen, now)
}

func (m *Screen) FrameSubmitted() {
	m.recordCoveredLoginFrame()
}

func (m *Screen) updatePhaseEscape(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	m.cancelLoginPhase(ctx, now)
	return true
}

func (m *Screen) cancelLoginPhase(ctx client.Context, now time.Time) {
	switch m.phase {
	case loginPhaseCreate:
		m.cancelCharacterCreate(now)
	case loginPhaseCharacter:
		m.accountStep = loginAccountCredentials
		m.startPhaseFade(loginPhaseAccount, now)
		if ctx.Network != nil {
			ctx.Network.Close()
		}
		m.status = "char select cancelled"
	case loginPhaseAccount:
		switch {
		case m.accountStep == loginAccountCharacterService || m.accountStep == loginAccountCharacterConnecting:
			m.cancelCharacterServiceSelection(ctx)
		case m.accountStep == loginAccountCredentials && len(loginConnections(ctx)) > 1 && !ctx.Config.Login.AutoLogin:
			m.showLoginServerSelection(ctx)
		default:
			m.updateAccountWindow(ctx)
			m.openQuitConfirm(ctx)
		}
	}
}

func (m *Screen) updateQuitConfirm(ctx client.Context) bool {
	if !m.quitConfirm.IsOpen() {
		return false
	}
	return m.quitConfirm.Update(ctx)
}

func (m *Screen) updateDisconnectDialog(ctx client.Context) bool {
	if !m.disconnectDialog.IsOpen() {
		return false
	}
	return m.disconnectDialog.Update(ctx)
}

func (m *Screen) openQuitConfirm(ctx client.Context) {
	m.quitConfirm.Open(ctx, "Exit", "Do you really want to quit?", func() {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
	}, nil)
}

func (m *Screen) startPhaseFade(target loginPhase, now time.Time) {
	if m.fade.phase != loginFadeNone && m.fade.hasTarget && m.fade.target == target && !m.fade.enterWorld {
		return
	}
	if m.phase == target {
		return
	}
	m.fade = loginFadeState{
		phase:     loginFadeOut,
		started:   now,
		target:    target,
		hasTarget: true,
	}
}

func (m *Screen) startWorldFade(now time.Time) {
	if m.fade.phase != loginFadeNone && m.fade.enterWorld {
		return
	}
	m.fade = loginFadeState{
		phase:      loginFadeOut,
		started:    now,
		enterWorld: true,
	}
}

func (m *Screen) updateFade(ctx client.Context, now time.Time) bool {
	switch m.fade.phase {
	case loginFadeOut:
		if now.Sub(m.fade.started) < loginTransitionDuration {
			return false
		}
		if m.fade.enterWorld {
			m.fade.phase = loginFadeHold
			m.fade.coveredFrames = 0
			return false
		}
		if m.fade.hasTarget {
			m.phase = m.fade.target
			m.publishPhaseWindow(ctx)
		}
		m.fade = loginFadeState{phase: loginFadeIn, started: now}
	case loginFadeHold:
		return m.fade.enterWorld && m.fade.coveredFrames >= loginWorldHandoffFrames
	case loginFadeIn:
		if now.Sub(m.fade.started) >= loginTransitionDuration {
			m.fade = loginFadeState{}
		}
	}
	return false
}

func (m *Screen) publishPhaseWindow(ctx client.Context) {
	m.clearLoginWindows(ctx)
	switch m.phase {
	case loginPhaseCharacter:
		m.showCharacterSelectWindow(ctx)
	case loginPhaseAccount:
		m.updateAccountWindow(ctx)
	case loginPhaseCreate:
		m.showCharacterCreateWindow(ctx)
	}
}

func (m *Screen) clearLoginWindows(ctx client.Context) {
	m.serviceWindow = nil
	m.loginWindow = nil
	m.charSelectWindow = nil
	m.charCreateWindow = nil
	m.bindings.Console.Unpublish(ctx)
	if ctx.UIManager != nil {
		ctx.UIManager.Clear()
	}
}

func (m *Screen) fadeAlpha(now time.Time) uint8 {
	switch m.fade.phase {
	case loginFadeHold:
		return 255
	case loginFadeOut:
		if m.fade.started.IsZero() {
			return 0
		}
		return scene.ClampColor(255 * clampUnit(float64(now.Sub(m.fade.started))/float64(loginTransitionDuration)))
	case loginFadeIn:
		if m.fade.started.IsZero() {
			return 0
		}
		return scene.ClampColor(255 * (1 - clampUnit(float64(now.Sub(m.fade.started))/float64(loginTransitionDuration))))
	default:
		return 0
	}
}

func (m *Screen) recordCoveredLoginFrame() {
	if m.fade.phase == loginFadeHold && m.fade.coveredFrames < loginWorldHandoffFrames {
		m.fade.coveredFrames++
	}
}

func (m *Screen) drawFade(ctx client.Context, screen *render.Frame, now time.Time) {
	alpha := m.fadeAlpha(now)
	if alpha == 0 {
		return
	}
	width, height := ctx.ScreenSize()
	render.DrawRect(screen, 0, 0, float64(width), float64(height), color.RGBA{A: alpha})
}

// Finish releases login UI and keepalives before entering the world.
func (m *Screen) Finish(ctx client.Context) {
	m.clearLoginWindows(ctx)
	m.quitConfirm = gameui.ConfirmModal{}
	m.disableCharServerPing()
}

func (m *Screen) moveSelectedSlot(delta int) {
	m.selectedSlot = clampCharacterSlot(m.selectedSlot+delta, m.maxSlots)
}

func (m *Screen) moveToPreviousCharacterSlot() {
	m.moveSelectedSlot(-1)
}

func (m *Screen) moveToNextCharacterSlot() {
	m.moveSelectedSlot(1)
}

func (m *Screen) submitSelectedCharacter(ctx client.Context) {
	if m.charSelectPending {
		return
	}
	character, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot)
	if !ok {
		m.status = "empty character slot"
		return
	}
	if ctx.Network == nil {
		m.status = "select character failed: not connected"
		return
	}
	if err := ctx.Network.SendSelectCharacter(character.Slot); err != nil {
		m.status = "select character failed: " + err.Error()
		return
	}
	m.charSelectPending = true
	m.playConfirmSFX(ctx)
	ctx.Session.SelectCharacter(character)
	m.status = fmt.Sprintf("selected character %s", character.Name)
}

func (m *Screen) drawBackground(ctx client.Context, screen *render.Frame) {
	screen.Fill(color.Black)
	width, height := ctx.ScreenSize()
	if width <= 0 || height <= 0 {
		return
	}
	if len(m.bgTiles) == 12 {
		cellW := float64(width) / 4
		cellH := float64(height) / 3
		for i, tile := range m.bgTiles {
			if tile == nil {
				continue
			}
			b := tile.Bounds()
			if b.Dx() <= 0 || b.Dy() <= 0 {
				continue
			}
			var opts render.DrawImageOptions
			opts.GeoM.Scale(cellW/float64(b.Dx()), cellH/float64(b.Dy()))
			opts.GeoM.Translate(float64(i%4)*cellW, float64(i/4)*cellH)
			opts.Filter = render.FilterLinear
			screen.DrawImage(tile, &opts)
		}
		return
	}
	if m.background == nil {
		render.DrawRect(screen, 0, 0, float64(width), float64(height), color.RGBA{R: 10, G: 13, B: 22, A: 255})
		return
	}
	b := m.background.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Scale(float64(width)/float64(b.Dx()), float64(height)/float64(b.Dy()))
	opts.Filter = render.FilterLinear
	screen.DrawImage(m.background, &opts)
}

func (m *Screen) loadBackground(ctx client.Context) {
	if m.bgLoaded {
		return
	}
	m.bgLoaded = true
	for _, set := range loginBackgroundSets(ctx.Config.Packet.ClientDate) {
		if len(set) == 1 {
			img, source, ok := loadLoginBackgroundImage(ctx.Resources, set[0])
			if ok {
				m.background = img
				m.bgSource = source
				return
			}
			continue
		}
		tiles := make([]*render.Image, 0, len(set))
		sources := make([]string, 0, len(set))
		ok := true
		for _, name := range set {
			img, source, loaded := loadLoginBackgroundImage(ctx.Resources, name)
			if !loaded {
				ok = false
				break
			}
			tiles = append(tiles, img)
			sources = append(sources, source)
		}
		if ok {
			m.bgTiles = tiles
			m.bgSource = fmt.Sprintf("%d login tiles", len(sources))
			return
		}
	}
	m.bgSource = "fallback"
}

func (m *Screen) playLoginBGM(ctx client.Context) {
	if m.bgmStarted || ctx.Audio == nil {
		return
	}
	m.bgmStarted = true
	_ = ctx.Audio.Play("01.mp3")
}

func (m *Screen) playConfirmSFX(ctx client.Context) {
	if ctx.Audio == nil {
		return
	}
	source, err := ctx.Audio.PlaySFX(loginConfirmSFX)
	if err == nil {
		if source != "" {
			glog.Debugf("sfx playing path=%s source=%s", loginConfirmSFX, source)
		}
	}
}

func loadLoginBackgroundImage(manager *res.Manager, name string) (*render.Image, string, bool) {
	if manager == nil {
		return nil, "", false
	}
	for _, candidate := range loginInterfaceCandidates(name) {
		img, source, err := res.LoadImageExact(manager, []string{candidate})
		if err == nil {
			return render.NewImageFromImage(img), source, true
		}
	}
	return nil, "", false
}

func loginBackgroundSets(clientDate int) [][]string {
	tiles2018 := []string{
		"t_배경1-1.bmp", "t_배경1-2.bmp", "t_배경1-3.bmp", "t_배경1-4.bmp",
		"t_배경2-1.bmp", "t_배경2-2.bmp", "t_배경2-3.bmp", "t_배경2-4.bmp",
		"t_배경3-1.bmp", "t_배경3-2.bmp", "t_배경3-3.bmp", "t_배경3-4.bmp",
	}
	sets := make([][]string, 0, 3)
	if clientDate >= 20221207 {
		sets = append(sets, []string{"t_login.jpg"})
	}
	if clientDate >= 20181114 {
		sets = append(sets, tiles2018)
	}
	sets = append(sets, []string{"bgi_temp.bmp"}, []string{"t_login.jpg"}, tiles2018)
	return sets
}

func loginInterfaceCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	const ui = "data\\texture\\유저인터페이스\\"
	candidates := []string{
		ui + name,
		strings.ReplaceAll(ui, "\\", "/") + name,
		"texture\\유저인터페이스\\" + name,
		"data\\texture\\interface\\" + name,
		"data/texture/interface/" + name,
		name,
	}
	return uniqueLoginStrings(candidates)
}

func uniqueLoginStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (m *Screen) connectAndMaybeLogin(ctx client.Context, conn res.Connection, userConfirmed bool) {
	sendLogin := strings.TrimSpace(m.username) != "" || m.password != ""
	if m.disconnectDialog.IsOpen() || (sendLogin && m.loginPending) {
		return
	}
	if sendLogin {
		m.loginPending = true
	}
	m.disableLoginServerPing()
	m.disableCharServerPing()
	if ctx.Session != nil {
		ctx.Session.AdminList = append([]uint32(nil), conn.AdminList...)
	}
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, conn.Address, conn.Port)
	cancel()
	if err != nil {
		m.loginPending = false
		m.status = err.Error()
		m.showConnectionFailed(ctx)
		return
	}

	m.status = ctx.Network.Status()
	glog.Infof("connected login server %s:%d admins=%v", conn.Address, conn.Port, conn.AdminList)
	if !sendLogin {
		return
	}

	clientType := loginClientType(conn.LangType)
	err = ctx.Network.SendAccountLogin(
		m.username,
		m.password,
		uint32(conn.Version),
		clientType,
	)
	if err != nil {
		m.loginPending = false
		m.status = "login packet failed: " + err.Error()
		return
	}
	if userConfirmed {
		m.playConfirmSFX(ctx)
	}
	m.status = "CA_LOGIN sent"
	glog.Debugf("sent CA_LOGIN user=%s version=%d client_type=%d", m.username, conn.Version, clientType)
}

func loginClientType(langType int) uint8 {
	if langType < 0 || langType > 255 {
		return 0
	}
	return uint8(langType)
}

func (m *Screen) enableCharServerPing(now time.Time) {
	m.charPingActive = true
	m.nextCharPing = now.Add(charServerPingInterval)
}

func (m *Screen) enableLoginServerPing(now time.Time) {
	m.loginPingActive = true
	m.nextLoginPing = now.Add(loginServerPingInterval)
}

func (m *Screen) disableLoginServerPing() {
	m.loginPingActive = false
	m.nextLoginPing = time.Time{}
}

func (m *Screen) maybeSendLoginServerPing(ctx client.Context, now time.Time) {
	if !m.loginPingActive || m.disconnectDialog.IsOpen() || m.accountStep != loginAccountCharacterService {
		return
	}
	if ctx.Network == nil || strings.TrimSpace(m.username) == "" {
		return
	}
	if m.nextLoginPing.IsZero() {
		m.nextLoginPing = now.Add(loginServerPingInterval)
		return
	}
	if now.Before(m.nextLoginPing) {
		return
	}
	m.nextLoginPing = now.Add(loginServerPingInterval)
	if err := ctx.Network.SendLoginServerKeepalive(m.username); err != nil {
		m.disableLoginServerPing()
		glog.Warnf("login server keepalive failed user=%s: %v", m.username, err)
	}
}

func (m *Screen) disableCharServerPing() {
	m.charPingActive = false
	m.nextCharPing = time.Time{}
}

func (m *Screen) maybeSendCharServerPing(ctx client.Context, now time.Time) {
	if !m.charPingActive || m.disconnectDialog.IsOpen() {
		return
	}
	if m.phase != loginPhaseCharacter && m.phase != loginPhaseCreate {
		return
	}
	if ctx.Network == nil || ctx.Session == nil || ctx.Session.AccountID == 0 {
		return
	}
	if m.nextCharPing.IsZero() {
		m.nextCharPing = now.Add(charServerPingInterval)
		return
	}
	if now.Before(m.nextCharPing) {
		return
	}
	m.nextCharPing = now.Add(charServerPingInterval)
	if err := ctx.Network.SendPing(ctx.Session.AccountID); err != nil {
		m.disableCharServerPing()
		m.status = "char ping failed: " + err.Error()
		glog.Warnf("char server ping failed account_id=%d: %v", ctx.Session.AccountID, err)
	}
}

func (m *Screen) connectCharServer(ctx client.Context, server network.CharServer) bool {
	// A new character-server connection allows selection again. Keep the guard
	// set during the map-server handoff, while the old selection UI is visible.
	m.charSelectPending = false
	m.disableLoginServerPing()
	m.disableCharServerPing()
	if ctx.Network == nil {
		m.status = "char connect failed: not connected"
		return false
	}
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, server.Address, int(server.Port))
	cancel()
	if err != nil {
		m.status = "char connect failed: " + err.Error()
		m.showConnectionFailed(ctx)
		return false
	}

	err = ctx.Network.SendCharServerEnter(ctx.Session.AccountID, ctx.Session.AuthCode, ctx.Session.UserLevel, ctx.Session.Sex)
	if err != nil {
		m.status = "CA_ENTER failed: " + err.Error()
		return false
	}
	m.status = "CA_ENTER sent to char server"
	glog.Debugf("sent CA_ENTER account_id=%d addr=%s port=%d", ctx.Session.AccountID, server.Address, server.Port)
	return true
}

func (m *Screen) connectMapServer(ctx client.Context, zone network.ZoneServerNotify) {
	m.disableCharServerPing()
	dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err := ctx.Network.Connect(dialCtx, zone.Address, int(zone.Port))
	cancel()
	if err != nil {
		m.status = "map connect failed: " + err.Error()
		m.showConnectionFailed(ctx)
		return
	}

	err = ctx.Network.SendMapServerEnter(ctx.Session.AccountID, zone.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex)
	if err != nil {
		m.status = "CZ_ENTER2 failed: " + err.Error()
		return
	}
	m.status = "CZ_ENTER2 sent to map server"
	glog.Debugf("sent CZ_ENTER2 account_id=%d char_id=%d addr=%s port=%d", ctx.Session.AccountID, zone.CharID, zone.Address, zone.Port)
}

func convertCharServers(servers []network.CharServer) []session.CharServer {
	out := make([]session.CharServer, 0, len(servers))
	for _, server := range servers {
		out = append(out, session.CharServer{
			Address:   server.Address,
			Port:      server.Port,
			Name:      server.Name,
			UserCount: server.UserCount,
			State:     server.State,
			Property:  server.Property,
		})
	}
	return out
}

func convertCharacters(characters []network.Character) []session.Character {
	out := make([]session.Character, 0, len(characters))
	for _, character := range characters {
		out = append(out, convertCharacter(character))
	}
	return out
}

func convertCharacter(character network.Character) session.Character {
	return session.Character{
		ID:        character.ID,
		Exp:       character.Exp,
		Money:     character.Money,
		Name:      character.Name,
		Slot:      character.Slot,
		Level:     character.Level,
		JobLevel:  character.JobLevel,
		Job:       character.Job,
		HP:        character.HP,
		MaxHP:     character.MaxHP,
		SP:        character.SP,
		MaxSP:     character.MaxSP,
		Str:       character.Str,
		Agi:       character.Agi,
		Vit:       character.Vit,
		Int:       character.Int,
		Dex:       character.Dex,
		Luk:       character.Luk,
		Hair:      character.Hair,
		HairColor: character.HairColor,
		HeadPal:   character.HeadPal,
		BodyPal:   character.BodyPal,
		Weapon:    character.Weapon,
		Shield:    character.Shield,
		HeadTop:   character.HeadTop,
		HeadMid:   character.HeadMid,
		HeadLow:   character.HeadLow,
		Option:    character.Option,
	}
}

func clampUnit(value float64) float64 { return math.Max(0, math.Min(1, value)) }

// ReportMapError returns to an explicit login attempt after a map failed to load.
func (m *Screen) ReportMapError(mapName string) {
	m.autoAttempted = true
	m.autoCharAttempted = true
	m.mapError = fmt.Sprintf("Cannot load map:\n%s\nCheck your game data.\nPlease log in again.", mapName)
}
