package login

import (
	"fmt"
	"image"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const (
	charSelectPreviewDirection = 4
	charSelectPreviewScale     = 0.92
	charSelectPreviewFeetLift  = 10
	charDeleteKeyMaxBytes      = 40
)

func (m *Screen) updateCharacterSelectInput(ctx client.Context) {
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		m.cancelCharacterSelect(ctx)
		return
	}
	if ctx.Input != nil {
		if ctx.Input.JustPressed(input.KeyArrowLeft) {
			m.moveToPreviousCharacterSlot()
		}
		if ctx.Input.JustPressed(input.KeyArrowRight) {
			m.moveToNextCharacterSlot()
		}
		if ctx.Input.JustPressed(input.KeyEnter) {
			m.activateCharacterSelectSlot(ctx, m.selectedSlot, time.Now())
		}
	}
	if charSelectWebEnabled() {
		// The DOM layer owns rendering and button taps; keyboard shortcuts
		// above still drive the same selection state it displays.
		m.drainCharSelectWebActions(ctx)
		m.syncCharSelectWeb(ctx)
		return
	}
	m.showCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.charSelectWindow.Update(ctx)
		m.charSelectWindow.Publish(ctx)
	}
}

func (m *Screen) updateCharacterSelectWindow(ctx client.Context) {
	if ctx.Config.Headless || charSelectWebEnabled() {
		return
	}
	opts := gameui.CharacterSelectWindowOptions{
		SelectedSlot: m.selectedSlot,
		MaxSlots:     m.maxSlots,
	}
	if ctx.Session != nil {
		opts.Characters = ctx.Session.Characters
	}
	opts.PreviewImages = m.characterSelectPreviewImages(ctx, opts)
	if m.charSelectWindow == nil {
		callbacks := m.characterSelectWindowCallbacks(ctx)
		m.charSelectWindow = gameui.NewCharacterSelectWindow(ctx, opts, callbacks)
		return
	}
	m.charSelectWindow.SetOptions(ctx, opts)
}

func (m *Screen) characterSelectWindowCallbacks(ctx client.Context) gameui.CharacterSelectWindowCallbacks {
	return gameui.CharacterSelectWindowCallbacks{
		OnSelectSlot: func(slot int) {
			m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
		},
		OnActivateSlot: func(slot int) {
			m.activateCharacterSelectSlot(ctx, slot, time.Now())
		},
		OnPreviousSlot: m.moveToPreviousCharacterSlot,
		OnNextSlot:     m.moveToNextCharacterSlot,
		OnMake: func() {
			slot := m.selectedSlot
			if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
				m.status = "character slot occupied"
				return
			}
			m.openCharacterCreate(ctx, slot, time.Now())
		},
		OnOK: func() {
			m.submitSelectedCharacter(ctx)
		},
		OnCancel: func() {
			m.cancelCharacterSelect(ctx)
		},
		OnDelete: func() {
			m.openCharacterDeleteConfirm(ctx)
		},
	}
}

func (m *Screen) activateCharacterSelectSlot(ctx client.Context, slot int, now time.Time) {
	if m.charSelectPending {
		return
	}
	m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); ok {
		m.submitSelectedCharacter(ctx)
		return
	}
	m.openCharacterCreate(ctx, m.selectedSlot, now)
}

func (m *Screen) characterSelectPreviewImages(ctx client.Context, opts gameui.CharacterSelectWindowOptions) map[int]image.Image {
	images := make(map[int]image.Image, 3)
	pageStart := gameui.CharacterSelectPage(opts.SelectedSlot) * 3
	for localSlot := 0; localSlot < 3; localSlot++ {
		slot := pageStart + localSlot
		character, ok := characterBySlot(opts.Characters, slot)
		if !ok {
			continue
		}
		if image := m.characterPreviewImage(ctx, character); image != nil {
			images[slot] = image
		}
	}
	return images
}

func (m *Screen) showCharacterSelectWindow(ctx client.Context) {
	m.updateCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.charSelectWindow.Publish(ctx)
	}
}

func (m *Screen) updateCharacterDelete(ctx client.Context) bool {
	if charSelectWebEnabled() {
		// The DOM layer mirrors the delete flow (confirm, then email key);
		// its actions drive the same packet path as the canvas modals.
		m.drainCharSelectWebActions(ctx)
		m.syncCharSelectWeb(ctx)
		return m.charDeleteWebStep != 0
	}
	if m.charDeleteConfirm.Update(ctx) {
		return true
	}
	consumed := m.charDeletePrompt.Update(ctx)
	action := m.charDeletePrompt.PopAction()
	if action.Submitted {
		m.submitCharacterDelete(ctx, action.Text)
		return true
	}
	return consumed
}

func (m *Screen) openCharacterDeleteConfirm(ctx client.Context) {
	if ctx.Session == nil {
		m.status = "delete character failed: no session"
		return
	}
	character, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot)
	if !ok {
		m.status = "empty character slot"
		return
	}
	m.deleteCharID = character.ID
	if charSelectWebEnabled() {
		m.charDeleteWebStep = 1
		m.charDeleteWebName = character.Name
		return
	}
	m.charDeleteConfirm.Open(ctx, "Delete Character", fmt.Sprintf("Delete %s?", character.Name), func() {
		m.charDeletePrompt.Open(ctx, "Delete Character", "Email", "Email", charDeleteKeyMaxBytes)
	}, nil)
}

func (m *Screen) submitCharacterDelete(ctx client.Context, key string) {
	if m.deleteCharID == 0 {
		m.status = "delete character failed: no character selected"
		return
	}
	if ctx.Network == nil {
		m.status = "delete character failed: not connected"
		return
	}
	if err := ctx.Network.SendDeleteCharacter(m.deleteCharID, key); err != nil {
		m.status = "delete character failed: " + err.Error()
		return
	}
	m.status = "delete character request sent"
}

func (m *Screen) applyDeleteCharacterAccept(ctx client.Context) {
	if m.deleteCharID == 0 {
		m.status = "deleted character"
		return
	}
	if ctx.Session == nil {
		m.deleteCharID = 0
		m.status = "deleted character"
		return
	}
	var deletedName string
	if character, ok := characterByID(ctx.Session.Characters, m.deleteCharID); ok {
		deletedName = character.Name
	}
	ctx.Session.Characters = removeCharacterByID(ctx.Session.Characters, m.deleteCharID)

	m.charPreviewImages = nil
	m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		if len(ctx.Session.Characters) > 0 {
			m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
		} else {
			m.selectedSlot = 0
		}
	}
	m.deleteCharID = 0
	m.charDeleteWebStep = 0
	if deletedName != "" {
		m.status = fmt.Sprintf("deleted character %s", deletedName)
	} else {
		m.status = "deleted character"
	}
	m.showCharacterSelectWindow(ctx)
}

func (m *Screen) applyDeleteCharacterRefuse(ctx client.Context, code uint8) {
	m.deleteCharID = 0
	m.charDeleteWebStep = 0
	m.status = describeDeleteCharacterRefuse(code)
	if charSelectWebEnabled() {
		// The DOM layer shows the status line; a canvas alert would be
		// invisible under it and swallow input.
		return
	}
	m.charDeleteConfirm.OpenAlert(ctx, "Delete Character", m.status, nil)
}

func (m *Screen) cancelCharacterSelect(ctx client.Context) {
	m.charSelectWindow = nil
	m.charDeleteWebStep = 0
	m.disableCharServerPing()
	m.startPhaseFade(loginPhaseAccount, time.Now())
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	m.status = "char select cancelled"
	// Push the phase change so the DOM layer hides immediately.
	m.syncCharSelectWeb(ctx)
}

func (m *Screen) prepareCharacterSelectFromSession(ctx client.Context) {
	m.maxSlots = 9
	m.selectedSlot = 0
	if ctx.Session == nil || len(ctx.Session.Characters) == 0 {
		return
	}
	m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
	if configured := ctx.Config.Login.CharSlot; configured >= 0 {
		m.selectedSlot = clampCharacterSlot(configured, m.maxSlots)
	}
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
	}
}

func (m *Screen) autoSelectCharacter(ctx client.Context) bool {
	slot := ctx.Config.Login.CharSlot
	if !ctx.Config.Login.AutoLogin || slot < 0 {
		return false
	}
	// One-shot per connection: a char-server reconnect after a dropped
	// session delivers a fresh char list, and the auto-select must run
	// again on it — otherwise the client sat on the select screen forever
	// after any mid-handoff disconnect. A small attempt budget keeps a
	// submit-disconnect loop from spinning.
	if m.autoCharAttempted {
		return false
	}
	m.autoCharAttempted = true
	m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		m.status = fmt.Sprintf("character slot %d is empty", slot)
		glog.Warnf("autoselect character slot=%d failed: empty slot", slot)
		return false
	}
	glog.Debugf("autoselect character slot=%d", m.selectedSlot)
	m.submitSelectedCharacter(ctx)
	return true
}

func (m *Screen) reconnectCharacterServer(ctx client.Context) {
	if ctx.Network == nil || ctx.Session == nil || len(ctx.Session.CharServers) == 0 {
		return
	}
	// The previous connection (and its auto-select attempt) died with the
	// session; allow the auto-select to run again on the fresh char list.
	m.autoCharAttempted = false
	index := ctx.Session.CharServerIndex
	if index < 0 || index >= len(ctx.Session.CharServers) {
		index = 0
		ctx.Session.CharServerIndex = 0
	}
	server := ctx.Session.CharServers[index]
	m.connectCharServer(ctx, network.CharServer{
		Address:   server.Address,
		Port:      server.Port,
		Name:      server.Name,
		UserCount: server.UserCount,
		State:     server.State,
		Property:  server.Property,
	})
}

type characterPreviewTarget interface {
	DrawImage(*render.Image, *render.DrawImageOptions)
}

func (m *Screen) drawCharacterPreview(screen characterPreviewTarget, ctx client.Context, character session.Character, centerX, feetY int) {
	if m.bindings.Preview == nil {
		return
	}
	billboard := m.bindings.Preview(ctx, character, charSelectPreviewDirection)
	if billboard == nil || billboard.Image == nil {
		return
	}
	var opts render.DrawImageOptions
	scale := charSelectPreviewScale
	if db.IsBabyJob(int(character.Job)) {
		scale *= scene.BabyPlayerBodyScale
	}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(centerX)-billboard.AnchorX*scale, float64(feetY)-billboard.AnchorY*scale)
	opts.Filter = scene.SpriteDrawFilter()
	screen.DrawImage(billboard.Image, &opts)
}

func (m *Screen) characterPreviewImage(ctx client.Context, character session.Character) image.Image {
	if character.ID == 0 {
		return nil
	}
	if m.charPreviewImages == nil {
		m.charPreviewImages = make(map[uint32]image.Image)
	}
	if img := m.charPreviewImages[character.ID]; img != nil {
		return img
	}
	img := render.NewImage(139, 144)
	m.drawCharacterPreview(img, ctx, character, 139/2, 144-15-charSelectPreviewFeetLift)
	m.charPreviewImages[character.ID] = img.RGBA()
	return m.charPreviewImages[character.ID]
}

func (m *Screen) loadCharacterSelectSkin(ctx client.Context) {
	if m.charWindow == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/win_select.bmp"); ok {
			m.charWindow = img
		}
	}
	if m.charBox == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/box_select.bmp"); ok {
			m.charBox = img
		}
	}
}

func charSelectMaxSlots(characters []session.Character) int {
	maxSlots := 9
	for _, character := range characters {
		if int(character.Slot)+1 > maxSlots {
			maxSlots = int(character.Slot) + 1
		}
	}
	if maxSlots%3 != 0 {
		maxSlots += 3 - maxSlots%3
	}
	return maxSlots
}

func firstOccupiedCharacterSlot(characters []session.Character) int {
	if len(characters) == 0 {
		return 0
	}
	slot := int(characters[0].Slot)
	for _, character := range characters[1:] {
		if int(character.Slot) < slot {
			slot = int(character.Slot)
		}
	}
	return slot
}

func firstEmptyCharacterSlot(characters []session.Character, maxSlots int) (int, bool) {
	if maxSlots <= 0 {
		maxSlots = 9
	}
	occupied := make(map[int]struct{}, len(characters))
	for _, character := range characters {
		occupied[int(character.Slot)] = struct{}{}
	}
	for slot := 0; slot < maxSlots; slot++ {
		if _, ok := occupied[slot]; !ok {
			return slot, true
		}
	}
	return 0, false
}

func clampCharacterSlot(slot, maxSlots int) int {
	if maxSlots <= 0 {
		maxSlots = 1
	}
	if slot < 0 {
		return 0
	}
	if slot >= maxSlots {
		return maxSlots - 1
	}
	return slot
}

func characterBySlot(characters []session.Character, slot int) (session.Character, bool) {
	for _, character := range characters {
		if int(character.Slot) == slot {
			return character, true
		}
	}
	return session.Character{}, false
}

func characterByID(characters []session.Character, id uint32) (session.Character, bool) {
	for _, character := range characters {
		if character.ID == id {
			return character, true
		}
	}
	return session.Character{}, false
}

func upsertCharacter(characters []session.Character, character session.Character) []session.Character {
	for i := range characters {
		if characters[i].ID == character.ID || characters[i].Slot == character.Slot {
			characters[i] = character
			return characters
		}
	}
	return append(characters, character)
}

func removeCharacterByID(characters []session.Character, id uint32) []session.Character {
	out := characters[:0]
	for _, character := range characters {
		if character.ID != id {
			out = append(out, character)
		}
	}
	return out
}

func describeDeleteCharacterRefuse(code uint8) string {
	switch code {
	case 0:
		return "Character deletion refused."
	default:
		return fmt.Sprintf("Character deletion refused (%d).", code)
	}
}
