package game

import (
	"context"
	"image"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/bot"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/login"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/game/weather"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type WorldMode struct {
	mapScene     maps.System
	effectSystem effects.System

	gamepadDirLogged    bool
	gamepadCameraAt     time.Time
	heldMenuOpen        bool
	menuPanelsShown     bool
	mapOverlay          int
	storageDeposit      storageDepositDialog
	storageWithdraw     storageDepositDialog
	heldMenuSel         int
	heldMenuMovedAt     time.Time
	heldMenuActivatedAt time.Time
	gamepadActionAt     time.Time
	statsSelMovedAt     time.Time
	invSelMovedAt       time.Time
	deathSelMovedAt     time.Time
	mapLoad             *mapLoadState
	mapLoadRSWSource    string
	imageMiss           map[string]struct{}
	vendingBoardsSig    string
	damageMsgMiss       bool
	timeFontMiss        bool
	effectViewMiss      map[string]struct{}
	gr2ModelMiss        map[int]struct{}
	hasPetProperty      bool
	ui                  worldUI
	pendingChatRoom     network.ChatRoomCreate
	pendingTradeName    string
	showcase            *loadingShowcase
	mapUploadBatch      int
	textureUpscaling    bool // Captured when loading the map; cached meshes keep matching UVs.
	mail                mailState
	walkCooldownUntil   time.Time
	walkSequence        uint64
	nextHeldWalkAt      time.Time
	camera              followCamera
	cameraShakeStart    time.Time
	cameraShakeEnd      time.Time
	spriteUpscaler      *sprites.Upscaler
	imageCache          map[string]image.Image

	playerView       *sprites.Humanoid
	shadowView       *scene.SpriteView
	shadowViewMiss   bool
	cartViews        map[int]*scene.SpriteView
	cartViewMiss     map[int]struct{}
	falconViews      map[int]*scene.SpriteView
	falconViewMiss   map[int]struct{}
	falcons          map[uint32]*falconRenderState
	cursorView       *scene.SpriteView
	cursorViewMiss   bool
	slotMachineView  *scene.SpriteView
	slotMachineMiss  bool
	cursorFallback   *render.Image
	cursorAction     int
	cursorStarted    time.Time
	damageNumberView *scene.SpriteView
	damageNumberMiss bool
	damageNumbers    map[string]*scene.Billboard
	cursorLevelNums  map[string]*scene.Billboard
	damageMsgView    *scene.SpriteView
	timeFontView     *scene.SpriteView
	itemMarker       *render.Image
	itemViews        map[itemSpriteKey]*scene.SpriteView
	itemViewMiss     map[itemSpriteKey]struct{}

	actorViews        map[sprites.Appearance]*sprites.Humanoid
	actorViewMiss     map[sprites.Appearance]struct{}
	mercenaryViews    map[sprites.Appearance]*sprites.Humanoid
	mercenaryViewMiss map[sprites.Appearance]struct{}
	nonPCViews        map[int]*scene.SpriteView
	nonPCViewMiss     map[int]struct{}
	gr2Models         map[int]*gr2ModelView
	petAccessoryIDs   map[uint32]uint32
	petAccessoryViews map[petAccessorySpriteKey]*scene.SpriteView
	petAccessoryMiss  map[petAccessorySpriteKey]struct{}
	pendingWarp       bool
	deferredPackets   []network.Packet
	pendingAttack     attackIntent
	pendingPickup     pickupIntent
	pendingSkill      pendingSkillTarget
	pendingSkillText  pendingSkillTextTarget
	senseRequest      senseRequest
	guildAction       gameui.GuildMemberAction
	guildOpenPending  bool
	pendingPetCapture petCaptureState
	petProperty       network.PetProperty
	petOldFullness    uint16
	petLastTalk       time.Time
	petInfoRequested  bool
	petID             uint32
	petSlotMachine    petSlotMachineState
	lockedAttackID    uint32
	attackFocusID     uint32
	attackFocusStart  time.Time
	scriptHighlight   actorHighlight
	lastAttackAt      time.Time
	lastChaseAt       time.Time
	actorAnims        map[uint32]actorAnimation

	damageFloaters []damageFloater

	actorCastBars    map[uint32]actorCastBar
	serverProgress   serverProgressState
	showDigit        showDigitState
	scheduledSounds  []scheduledSound
	scheduledStops   []scheduledActorStop
	scheduledResumes []scheduledWalkResume
	weather          weather.System
	actorDeaths      map[uint32]time.Time
	actorVanishes    map[uint32]actorVanishFade
	actorSoundFrames map[uint32]actorSoundFrame
	actorLife        map[uint32]actorLife
	graffiti         map[uint32]*groundGraffiti
	skillUnitModels  map[uint32]skillUnitModel
	hiddenSkillUnits map[uint32]skillUnitModel
	actorNameReqAt   map[uint32]time.Time
	guildEmblems     map[uint32]guildEmblem
	speechBubbles    map[uint32]speechBubble

	mapFade          maps.Fade
	hoveredWalk      hoveredWalkCellCache
	bot              *bot.Script
	companionSystem  companion.System
	textureMiss      map[string]struct{}
	strEffectMiss    map[string]struct{}
	chatBoardTipSig  string
}

type worldUI struct {
	companions           companion.Windows
	minimap              gameui.Minimap
	statusIcons          gameui.StatusIcons
	pvpCounter           gameui.PvPCounter
	perfHUD              gameui.PerfHUD
	levelUpNotifications gameui.LevelUpNotifications
	announcement         gameui.Announcement
	poptips              gameui.Poptips
	console              gameui.ChatConsole
	hotbar               hotbar
	npcDialog            gameui.NPCDialog
	npcCutin             gameui.NPCCutinOverlay
	escapeMenu           gameui.EscapeMenu
	teleportModal        gameui.TeleportModal
	autoSpellWindow      gameui.AutoSpellWindow
	disconnectDialog     gameui.ConfirmModal
	friendRequest        gameui.ConfirmModal
	friendConfirm        gameui.ConfirmModal
	partyRequest         gameui.ConfirmModal
	guildRequest         gameui.ConfirmModal
	guildAllianceRequest gameui.ConfirmModal
	guildRelationConfirm gameui.ConfirmModal
	guildMemberPrompt    gameui.TextPromptWindow
	tradeRequest         gameui.ConfirmModal
	adoptionRequest      gameui.ConfirmModal
	starPlaceConfirm     gameui.ConfirmModal
	characterWindow      gameui.CharacterWindow
	basicMenu            gameui.BasicMenu
	inventoryBag         gameui.InventoryBagWindow
	equipmentWindow      gameui.EquipmentWindow
	viewEquipWindow      gameui.ViewEquipmentWindow
	storageWindow        gameui.StorageWindow
	storagePassword      gameui.StoragePasswordWindow
	cartWindow           gameui.CartWindow
	changeCartWindow     gameui.ChangeCartWindow
	itemPickup           gameui.ItemPickupNotification
	shopWindow           gameui.ShopWindow
	vendingWindow        gameui.VendingWindow
	itemWindows          gameui.ItemWindows
	cardIllustration     gameui.CardIllustrationWindow
	monsterInfoWindow    gameui.MonsterInfoWindow
	bookWindow           gameui.BookWindow
	identifyWindow       gameui.IdentifyWindow
	cardWindow           gameui.CardCompositionWindow
	makingArrow          gameui.MakingArrowWindow
	makingItem           gameui.MakingItemWindow
	repairItem           gameui.RepairItemWindow
	weaponRefine         gameui.WeaponRefineWindow
	petEggWindow         gameui.PetEggWindow
	petInfoWindow        gameui.PetInfoWindow
	petContext           gameui.PetContextMenu
	petConfirm           gameui.ConfirmModal
	statsWindow          gameui.StatsWindow
	skillWindow          gameui.SkillWindow
	questWindow          gameui.QuestWindow
	worldMap             gameui.WorldMapWindow
	emoteWindow          gameui.EmoteWindow
	chatShortcuts        gameui.ChatShortcutsWindow
	friendsWindow        gameui.FriendsWindow
	guildWindow          gameui.GuildWindow
	friendSettings       gameui.FriendSettingsWindow
	whisperWindows       gameui.WhisperWindows
	chatRoomCreate       gameui.ChatRoomCreateWindow
	chatRoom             gameui.ChatRoomWindow
	partySettings        gameui.PartySettingsWindow
	partyCreate          gameui.PartyCreateWindow
	partyInvite          gameui.PartyInviteWindow
	partyInfo            gameui.ConfirmModal
	skillTextPrompt      gameui.TextPromptWindow
	playerContext        gameui.PlayerContextMenu
	tradeWindow          gameui.TradeWindow
	mailWindow           gameui.MailWindow
	settingsWindow       gameui.SettingsWindow
	shortcutBar          gameui.ShortcutBar
}

func (u *worldUI) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return playerIsDead(ctx) || u.keyboardInputBlocked(ctx)
}

// drainCameraButtons services the on-screen camera controls (web): touch
// devices have no wheel or two-finger rotate, so zoom and rotation ride on
// buttons fed through the same DOM action queue as the other overlays.
func (m *WorldMode) drainCameraButtons(ctx client.Context) {
	for _, action := range gameui.DrainCameraActions() {
		switch action {
		case "zoomin":
			m.camera.ZoomBy(1 / cameraButtonZoomStep)
		case "zoomout":
			m.camera.ZoomBy(cameraButtonZoomStep)
		case "rotl":
			if !scene.CameraRotationLockedForMap(ctx) {
				m.camera.Rotate(-cameraButtonRotateStep)
			}
		case "rotr":
			if !scene.CameraRotationLockedForMap(ctx) {
				m.camera.Rotate(cameraButtonRotateStep)
			}
		case "reset":
			if !scene.CameraRotationLockedForMap(ctx) {
				m.camera.ResetRotation()
			}
		}
	}
}

func (u *worldUI) keyboardInputBlocked(ctx client.Context) bool {
	if u == nil {
		return false
	}
	return u.console.Active() || u.nonConsoleKeyboardInputBlocked(ctx)
}

func (u *worldUI) nonConsoleKeyboardInputBlocked(ctx client.Context) bool {
	return u.nonConsoleKeyboardInputBlockedExcept(ctx, nil)
}

// A window may handle its own toggle shortcut, but not bypass other forms.
func (u *worldUI) nonConsoleKeyboardInputBlockedExcept(ctx client.Context, except *gameui.Window) bool {
	if u == nil {
		return false
	}
	open := func(w *gameui.Window) bool { return w != except && w.IsOpen() }
	return u.npcDialog.IsOpen() ||
		!u.npcInputAvailable() ||
		open(&u.settingsWindow.Window) ||
		u.chatShortcuts.IsOpen() ||
		u.worldMap.IsOpen() ||
		u.monsterInfoWindow.IsOpen() ||
		u.identifyWindow.IsOpen() ||
		u.cardWindow.IsOpen() ||
		u.makingArrow.IsOpen() ||
		u.makingItem.IsOpen() ||
		u.repairItem.IsOpen() ||
		u.weaponRefine.IsOpen() ||
		u.petEggWindow.IsOpen() ||
		(except != &u.petInfoWindow.Window && u.petInfoWindow.KeyboardShortcutsBlocked()) ||
		(except != &u.companions.HomunculusInfo.Window && u.companions.HomunculusInfo.KeyboardShortcutsBlocked()) ||
		u.changeCartWindow.IsOpen() ||
		u.inventoryBag.KeyboardShortcutsBlocked() ||
		u.itemWindows.KeyboardShortcutsBlocked() ||
		u.bookWindow.IsOpen() ||
		u.shopWindow.KeyboardShortcutsBlocked() ||
		u.vendingWindow.KeyboardShortcutsBlocked() ||
		u.tradeWindow.IsOpen() ||
		u.mailWindow.IsOpen() ||
		u.guildWindow.KeyboardShortcutsBlocked() ||
		u.friendSettings.IsOpen() ||
		u.whisperWindows.IsOpen() ||
		open(&u.chatRoomCreate.Window) ||
		u.chatRoom.IsOpen() ||
		u.partySettings.IsOpen() ||
		u.partyCreate.IsOpen() ||
		u.partyInvite.IsOpen() ||
		u.skillTextPrompt.IsOpen()
}

// Higher-priority modals block NPC input in both normal UI dispatch and Lua.
// This does not require an open NPC dialog: Update also unpublishes closed ones.
func (u *worldUI) npcInputAvailable() bool {
	return !u.disconnectDialog.IsOpen() &&
		!u.escapeMenu.IsOpen() &&
		!u.interactionModalOpen() &&
		!u.partyInfo.IsOpen() &&
		!u.petConfirm.IsOpen() &&
		!u.companions.HomunculusConfirm.IsOpen() &&
		!u.companions.MercenaryConfirm.IsOpen() &&
		!u.inventoryBag.KeyboardShortcutsBlocked()
}

func (u *worldUI) interactionModalOpen() bool {
	if u == nil {
		return false
	}
	return u.teleportModal.IsOpen() ||
		u.mailWindow.ModalOpen() ||
		u.autoSpellWindow.IsOpen() ||
		u.storagePassword.IsOpen() ||
		u.friendRequest.IsOpen() ||
		u.friendConfirm.IsOpen() ||
		u.partyRequest.IsOpen() ||
		u.guildRequest.IsOpen() ||
		u.guildAllianceRequest.IsOpen() ||
		u.guildRelationConfirm.IsOpen() ||
		u.guildMemberPrompt.IsOpen() ||
		u.tradeRequest.IsOpen() ||
		u.adoptionRequest.IsOpen() ||
		u.starPlaceConfirm.IsOpen()
}

func (m *WorldMode) KeyboardShortcutsBlocked(ctx client.Context) bool {
	return m.ui.KeyboardShortcutsBlocked(ctx)
}

type pickupIntent struct {
	itemID  uint32
	expires time.Time
	readyAt time.Time
}

type pendingSkillTarget struct {
	skill       session.Skill
	maxLevel    int
	targetID    uint32
	ground      bool
	x, y        int
	text        string
	expires     time.Time
	readyAt     time.Time
	source      string
	started     time.Time
	lastChaseAt time.Time
}

type actorHighlight struct {
	id      uint32
	started time.Time
}

type pendingSkillTextTarget struct {
	skill  session.Skill
	x      int
	y      int
	source string
}

const actorNameRequestCooldown = time.Second

func NewWorldMode() *WorldMode {
	m := &WorldMode{}
	m.bindNPCDialogLifecycle()
	return m
}

func (m *WorldMode) Name() string {
	return "world"
}

func (m *WorldMode) Enter(ctx client.Context) Mode {
	if ctx.Session != nil {
		ctx.Session.SuperNovicePrayer = session.SuperNovicePrayer{}
	}
	m.releaseMapTextures()
	now := time.Now()
	m.bindNPCDialogLifecycle()
	m.mapFade.StartPrewarm()
	// Parse the item tables while the map streams: the first drop's sprite
	// cannot resolve its resource name until these are loaded, and the lazy
	// first-query load lands mid-combat otherwise. Cheap when the login
	// screen already prefetched the files.
	if ctx.Resources != nil {
		ctx.Resources.WarmItemMetadata()
		// The wav web pack only matters in-world; start it streaming now
		// so audio switches to pack lookups as soon as it lands. Sound
		// lookups fall back to loose files until then.
		ctx.Resources.StartDeferredWebPack()
	}
	m.mapFade.StartPrewarm()
	m.camera.ResetTracking()
	ctx.World.GAT = nil
	ctx.World.GND = nil
	ctx.World.RSW = nil
	ctx.World.RSM = nil
	ctx.World.RSMFail = 0
	if ctx.World.MapName != "" {
		// One pack download replaces the per-file round trips for everything
		// this map references; missing packs fall through to loose streaming.
		ctx.Resources.EnsureWebMapPack(strings.TrimSuffix(strings.TrimSuffix(ctx.World.MapName, ".gat"), ".rsw"))
		gat, _, err := maps.LoadGAT(ctx.Resources, ctx.World.MapName)
		if err != nil {
			return newMapErrorMode(ctx, err, m.ui.console)
		}
		ctx.World.GAT = gat
	}
	m.mapScene.Textures = make(map[string]*render.Image)
	m.mapScene.TextureMiss = make(map[string]struct{})
	m.playerView = nil
	m.shadowView = nil
	m.shadowViewMiss = false
	m.cursorView = nil
	m.cursorViewMiss = false
	m.cursorFallback = nil
	m.cursorAction = cursorActionDefault
	m.cursorStarted = now
	m.damageNumberView = nil
	m.damageNumberMiss = false
	m.damageNumbers = make(map[string]*scene.Billboard)
	m.cursorLevelNums = make(map[string]*scene.Billboard)
	m.damageMsgView = nil
	m.damageMsgMiss = false
	m.timeFontView = nil
	m.timeFontMiss = false
	m.itemMarker = nil
	m.itemViews = make(map[itemSpriteKey]*scene.SpriteView)
	m.itemViewMiss = make(map[itemSpriteKey]struct{})
	m.effects().EffectViews = make(map[string]*scene.SpriteView)
	m.effects().EffectViewMiss = make(map[string]struct{})
	m.actorViews = make(map[sprites.Appearance]*sprites.Humanoid)
	m.actorViewMiss = make(map[sprites.Appearance]struct{})
	m.mercenaryViews = make(map[sprites.Appearance]*sprites.Humanoid)
	m.mercenaryViewMiss = make(map[sprites.Appearance]struct{})
	m.nonPCViews = make(map[int]*scene.SpriteView)
	m.nonPCViewMiss = make(map[int]struct{})
	m.gr2Models = make(map[int]*gr2ModelView)
	m.gr2ModelMiss = make(map[int]struct{})
	m.petAccessoryIDs = make(map[uint32]uint32)
	m.petAccessoryViews = make(map[petAccessorySpriteKey]*scene.SpriteView)
	m.petAccessoryMiss = make(map[petAccessorySpriteKey]struct{})
	m.mapScene.ResetGeometry()
	m.pendingWarp = false
	m.pendingAttack = attackIntent{}
	m.pendingPickup = pickupIntent{}
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	m.guildAction = gameui.GuildMemberAction{}
	m.guildOpenPending = false
	m.ui.guildMemberPrompt.Close()
	m.lockedAttackID = 0
	m.clearAttackFocus()
	m.clearScriptHighlight()
	m.lastAttackAt = time.Time{}
	m.lastChaseAt = time.Time{}
	m.actorAnims = make(map[uint32]actorAnimation)
	m.effects().ActorBlurs = nil
	m.damageFloaters = nil
	m.serverProgress = serverProgressState{}
	m.showDigit = showDigitState{}
	m.ui.minimap.ClearBossMarker()
	m.ui.poptips.Clear()
	m.scheduledSounds = nil
	m.scheduledStops = nil
	m.mapScene.MapSoundNext = make(map[int]time.Time)
	m.weather = weather.System{}
	m.effects().StarFlights = nil
	m.actorDeaths = make(map[uint32]time.Time)
	m.actorVanishes = make(map[uint32]actorVanishFade)
	m.actorSoundFrames = make(map[uint32]actorSoundFrame)
	m.actorLife = make(map[uint32]actorLife)
	m.speechBubbles = make(map[uint32]speechBubble)
	m.syncCurrentActorEffectStateEffects(ctx)
	m.ui.npcDialog.ResetPublished(ctx)
	m.ui.npcCutin.Clear()
	ctx.World.Items = make(map[uint32]worldstate.FloorItem)
	// Bots need the collision grid and map acknowledgement, but no scene assets.
	if ctx.Config.Headless {
		if ctx.World.MapName != "" {
			_ = ctx.Network.SendLoadEndAck()
		}
		return nil
	}
	playerStatus := ""
	character := ctx.Session.SelectedCharacter()
	visualCharacter := localPlayerVisualCharacter(ctx)
	if view, status := sprites.LoadPlayer(ctx.Resources, visualCharacter, ctx.Session.Sex, localPlayerIsAdmin(ctx)); view != nil {
		m.playerView = view
		playerStatus = status
	} else {
		playerStatus = status
	}
	if view, status := sprites.LoadShadow(ctx.Resources); view != nil {
		m.shadowView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.shadowViewMiss = true
		glog.Warnf("actor shadow resources unavailable: %s", status)
	}
	m.cartViews = make(map[int]*scene.SpriteView)
	m.cartViewMiss = make(map[int]struct{})
	m.falconViews = make(map[int]*scene.SpriteView)
	m.falconViewMiss = make(map[int]struct{})
	m.falcons = make(map[uint32]*falconRenderState)
	if view, status := sprites.LoadCursor(ctx.Resources); view != nil {
		m.cursorView = view
		if status != "" {
			playerStatus += " " + status
		}
	} else {
		m.cursorViewMiss = true
		glog.Warnf("cursor resources unavailable: %s", status)
	}
	render.SetCursorMode(render.CursorModeHidden)
	glog.Debugf("player sprite resources char_id=%d name=%s admin=%t job=%d visual_job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, localPlayerIsAdmin(ctx), character.Job, visualCharacter.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
	m.rebindPersistentUI(ctx)
	if ctx.World.MapName == "" {
		return nil
	}

	if gnd, _, err := maps.LoadGND(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.GND = gnd
	} else {
		ctx.World.GND = nil
	}
	if rsw, rswSource, err := maps.LoadRSW(ctx.Resources, ctx.World.MapName); err == nil {
		ctx.World.RSW = rsw
		ctx.World.RSM, ctx.World.RSMFail = maps.LoadRSMModels(ctx.Resources, rsw)
		m.playMapBGM(ctx, rswSource)
		m.mapScene.PreloadMapSounds(ctx)
	} else {
		ctx.World.RSW = nil
		ctx.World.RSM = nil
		ctx.World.RSMFail = 0
		m.playMapBGM(ctx, ctx.World.MapName)
	}
	// Server-driven digit displays load their sprite on first use; warm it
	// with the rest of the map.
	ctx.Resources.Prefetch(serverDigitSpritePrefetchGroups()...)
	m.mapScene.PreloadTextures(ctx)
	_ = ctx.Network.SendLoadEndAck()
	return nil
}

func (m *WorldMode) rebindPersistentUI(ctx client.Context) {
	m.ui.console.Rebind(ctx)
	m.ui.console.OnGuildWindow = func() { m.toggleGuildWindow(ctx) }
	m.ui.guildWindow.EmblemImage = func(ctx client.Context) image.Image {
		if ctx.Session == nil || m.guildEmblems == nil {
			return nil
		}
		guildID := ctx.Session.GuildID
		version := ctx.Session.EmblemVersion
		if guildID == 0 {
			guildID = ctx.Session.Guild.ID
		}
		if version == 0 {
			version = ctx.Session.Guild.EmblemVersion
		}
		if guildID == 0 || version == 0 {
			return nil
		}
		emblem := m.guildEmblems[guildID]
		if emblem.image == nil || emblem.version < version {
			return nil
		}
		return emblem.image.RGBA()
	}
	m.setGuildEmblemOptions(ctx)
	m.ui.characterWindow.Rebind(ctx)
	m.ui.basicMenu.Rebind(ctx, m.basicMenuCallbacks(ctx))
	m.ui.inventoryBag.Rebind(ctx, &m.ui.itemWindows)
	m.ui.equipmentWindow.Rebind(ctx, &m.ui.itemWindows, &m.ui.cartWindow, m)
	m.ui.cartWindow.Rebind(ctx, &m.ui.itemWindows)
	m.ui.itemWindows.Rebind(ctx, m)
	m.ui.statsWindow.Rebind(ctx)
	m.ui.skillWindow.Rebind(ctx, m)
	m.ui.questWindow.Rebind(ctx, func(id uint32, active bool) { m.setQuestActive(ctx, id, active) })
	m.ui.worldMap.Rebind(ctx)
	m.ui.levelUpNotifications.Rebind(ctx)
	m.ui.emoteWindow.OnSelect = m.ui.chatShortcuts.SelectEmotion
	m.ui.emoteWindow.Rebind(ctx, &m.ui.console)
	m.ui.chatShortcuts.Rebind(ctx, &m.ui.console, func() {
		m.ui.emoteWindow.OpenWindow(ctx, &m.ui.console)
	})
	m.ui.companions.HomunculusSkill.Rebind(ctx, m)
	m.ui.companions.MercenarySkill.Rebind(ctx, m)
	m.ui.friendsWindow.Rebind(ctx)
	m.ui.guildWindow.Rebind(ctx)
	m.ui.guildMemberPrompt.Rebind(ctx)
	m.ui.friendSettings.Rebind(ctx)
	m.ui.partySettings.Rebind(ctx)
	m.ui.partyCreate.Rebind(ctx)
	m.ui.partyInvite.Rebind(ctx)
	m.ui.chatRoomCreate.Rebind(ctx)
	m.ui.chatRoom.Rebind(ctx)
	m.ui.settingsWindow.Rebind(ctx)
	m.ui.companions.HomunculusInfo.Rebind(ctx)
	m.ui.companions.MercenaryInfo.Rebind(ctx)
	m.ui.whisperWindows.Rebind(ctx)
	m.ui.shortcutBar.ResetOverlay(ctx)
}

func (m *WorldMode) playMapBGM(ctx client.Context, rswName string) {
	if ctx.Audio == nil {
		return
	}
	_, err := ctx.Audio.PlayMap(rswName)
	if err != nil {
		glog.Warnf("bgm failed map=%s: %v", rswName, err)
		return
	}
}

func (m *WorldMode) Update(ctx client.Context) (Mode, error) {
	now := time.Now()
	// DOM settings actions must land before any early-return below (map
	// fade, disconnect dialog): a queued resolution change that skips them
	// leaves the canvas rendering at the old scale while the settings
	// picker keeps showing the picked value.
	m.ui.settingsWindow.UpdateWeb(ctx)
	// Touch overlay actions land with settings priority: a stick held across
	// a map fade must keep steering once the new map is in, and the action
	// button must work during prewarm. Steering itself is throttled inside.
	// Fold a finished wav-pack download into the archive list on this
	// goroutine; no-op until StartDeferredWebPack's fetch lands.
	if ctx.Resources != nil {
		ctx.Resources.PollDeferredWebPack()
	}
	if m.mapFade.Phase == maps.FadeOut {
		if !m.mapFade.Elapsed(now) {
			return nil, nil
		}
		m.mapFade.Phase = maps.FadeHold
		m.mapFade.CoveredFrames = 0
		return nil, nil
	}
	if m.mapFade.Phase == maps.FadeHold && m.mapFade.CoveredFrames >= maps.FadeHandoffFrames {
		if m.mapFade.CharacterSelect {
			return m.nextCharacterSelectMode(ctx), nil
		}
		if m.mapFade.HasChange {
			change := m.mapFade.Change
			m.mapFade = maps.Fade{Phase: maps.FadeHold, Started: now}
			next := m.handleMapChange(ctx, change)
			if next != nil {
				return next, nil
			}
			if !m.pendingWarp {
				m.mapFade.StartIn(now)
			}
			return nil, nil
		}
	}
	m.mapFade.AdvancePrewarm(now, len(m.mapScene.Images.Uploads) != 0)
	if m.mapFade.Phase == maps.FadeIn && m.mapFade.Elapsed(now) {
		m.mapFade = maps.Fade{}
	}

	if m.updateDisconnectDialog(ctx) {
		return nil, nil
	}

	if next, stop := m.handleNetworkPackets(ctx, now); stop {
		return next, nil
	}
	// Status presentation must follow server updates even when a window or
	// modal consumes input for the rest of the frame.
	m.removeExpiredStatusEffects(ctx.Session, now)
	m.effects().AdvanceTaekwonFootprints(ctx, now)
	m.updateMail(ctx, now)
	progressBlocksActions := m.updateServerProgress(ctx, now)
	if !ctx.Config.Headless {
		m.ui.inventoryBag.UpdatePresentation(ctx, &m.ui.itemWindows)
		m.ui.statsWindow.UpdatePresentation(ctx)
		m.ui.statusIcons.Update(ctx, now)
		m.ui.pvpCounter.Update(ctx)
		// The FPS/MEM HUD is a debug instrument (?stats=1 / -render-stats). Kept
		// off by default: its 2Hz text refresh is a recurring dirty region that
		// unions with other UI damage and amplifies partial repaints.
		if ctx.Config.Render.Stats {
			m.ui.perfHUD.Update(ctx)
		}
		if !progressBlocksActions && m.handleLevelUpNotificationAction(ctx, m.ui.levelUpNotifications.Update(ctx)) {
			// The notification click belongs exclusively to the UI. Returning here
			// prevents the same press from reaching the map after the icon closes.
			return nil, nil
		}
	}
	if !progressBlocksActions {
		m.updatePendingAttack(ctx, "update", false)
		m.processPendingAttack(ctx)
		m.updatePendingPickup(ctx, "update", false)
		m.processPendingPickup(ctx)
		m.skills().UpdatePendingTarget(ctx, "update", false)
		m.skills().ProcessPendingTarget(ctx)
		m.processLockedAttack(ctx)
	}
	now = time.Now()
	m.cleanupDeadActors(ctx, now)
	m.cleanupVanishedActors(ctx, now)
	m.processScheduledActorStops(ctx, now)
	m.processScheduledWalkResumes(ctx, now)
	if ctx.Config.Headless {
		m.playDueScheduledSounds(ctx, now)
		if !progressBlocksActions && m.mapFade.Phase != maps.FadeHold && m.mapFade.Phase != maps.FadePrewarm {
			dead := playerIsDead(ctx)
			m.updateBotInput(ctx, !dead)
			if !dead {
				m.companions().Update(ctx, m.actorDeaths, now)
			}
			m.updateBot(ctx, now)
		}
		return nil, nil
	}
	m.processActorMotionSounds(ctx, now)
	m.mapScene.ProcessMapSounds(ctx, now, m.scheduleSoundVolumeAtPosition)
	m.playDueScheduledSounds(ctx, now)

	m.camera.Update(ctx, now)
	if m.mapFade.Phase == maps.FadeHold || m.mapFade.Phase == maps.FadePrewarm {
		return nil, nil
	}
	m.ui.console.UpdatePresentation(ctx)
	m.ui.questWindow.UpdatePresentation(ctx, now)
	m.ui.worldMap.UpdatePresentation(ctx)
	if progressBlocksActions {
		return nil, nil
	}
	if m.updateStoragePasswordWindow(ctx) {
		return nil, nil
	}
	// The drop amount prompt is modal and may overlap the always-present chat
	// field. Give it first refusal on Escape and pointer input so a focused chat
	// field cannot consume the cancellation before the prompt sees it.
	if m.ui.inventoryBag.UpdateDropPrompt(ctx) {
		return nil, nil
	}
	if m.ui.mailWindow.UpdateModal(ctx) || m.ui.mailWindow.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	if m.ui.guildWindow.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	if m.ui.chatShortcuts.UpdateKeyboardInput(ctx) {
		return nil, nil
	}
	dead := playerIsDead(ctx)
	keyboardBlocked := m.ui.keyboardInputBlocked(ctx)
	m.updateBotInput(ctx, !dead && !keyboardBlocked)
	if m.updatePetSlotMachine(ctx) {
		return nil, nil
	}
	// Window.Update consumes pointer hover so that map input does not pass
	// through the UI. Handle keyboard-only window shortcuts before pointer
	// dispatch, otherwise their JustPressed event can be lost.
	if m.chatShortcutFromInput(ctx) || m.toggleEmoteWindowFromInput(ctx) || m.toggleGuildWindowFromInput(ctx) || m.toggleQuestWindowFromInput(ctx) || m.toggleWorldMapFromInput(ctx) || m.toggleWindowFromInput(ctx) {
		return nil, nil
	}
	if !dead && !m.ui.nonConsoleKeyboardInputBlocked(ctx) && m.ui.shortcutBar.UpdateKeyboardInput(ctx, m, m.ui.console.Active()) {
		return nil, nil
	}
	if dead {
		m.updateBot(ctx, now)
		if m.updateDeathUIInput(ctx) {
			return nil, nil
		}
	} else if !keyboardBlocked {
		if m.skills().CancelFromInput(ctx) {
			return nil, nil
		}
		if m.cancelPetCaptureFromInput(ctx) {
			return nil, nil
		}
		// DOM item description/artwork panels sit above every game window
		// on the web build: Escape closes their topmost panel first, before
		// the escape menu can open. No-op on native.
		if gameui.UpdateItemInfoWebWindows(ctx) {
			return nil, nil
		}
		if m.openEscapeMenuFromInput(ctx) {
			return nil, nil
		}
		m.skills().AdjustPendingLevelFromWheel(ctx)
	}
	petContextConsumed := m.ui.petContext.Update(ctx)
	if action := m.ui.petContext.PopAction(); action.Kind != gameui.PetContextActionNone {
		if !dead {
			m.handlePetContextAction(ctx, action)
		}
		return nil, nil
	}
	if petContextConsumed {
		return nil, nil
	}
	if !dead && m.openPetContextFromInput(ctx, now) {
		return nil, nil
	}
	homunculusContextConsumed := m.ui.companions.HomunculusContext.Update(ctx)
	if action := m.ui.companions.HomunculusContext.PopAction(); action.Kind != gameui.HomunculusContextActionNone {
		if !dead {
			m.companions().HandleHomunculusContextAction(ctx, action)
		}
		return nil, nil
	}
	if homunculusContextConsumed {
		return nil, nil
	}
	if !dead && m.openHomunculusContextFromInput(ctx, now) {
		return nil, nil
	}
	mercenaryContextConsumed := m.ui.companions.MercenaryContext.Update(ctx)
	if action := m.ui.companions.MercenaryContext.PopAction(); action.Kind != gameui.MercenaryContextActionNone {
		if !dead {
			m.companions().HandleMercenaryContextAction(ctx, action)
		}
		return nil, nil
	}
	if mercenaryContextConsumed {
		return nil, nil
	}
	if !dead && m.openMercenaryContextFromInput(ctx, now) {
		return nil, nil
	}
	playerContextConsumed := m.ui.playerContext.Update(ctx)
	playerAction := m.ui.playerContext.PopAction()
	if dead && playerAction.Kind != gameui.PlayerContextActionNone {
		return nil, nil
	}
	switch action := playerAction; action.Kind {
	case gameui.PlayerContextActionAddFriend:
		m.sendAddFriend(ctx, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteParty:
		m.sendPartyInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionInviteGuild:
		m.sendGuildInvite(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionGuildAlliance:
		m.sendGuildAllianceRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionGuildHostility:
		m.sendGuildHostilityRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionTrade:
		m.sendTradeRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionSeeEquipment:
		m.sendViewEquipmentRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	case gameui.PlayerContextActionAdopt:
		m.sendAdoptionRequest(ctx, action.ActorID, action.Name)
		return nil, nil
	}
	if playerContextConsumed {
		return nil, nil
	}
	if !dead && m.handleCompanionAICommandClick(ctx, now) {
		return nil, nil
	}
	if !dead && m.openPlayerContextFromInput(ctx, now) {
		return nil, nil
	}
	if !m.mapPointerBlocked(ctx) && !m.ui.escapeMenu.IsOpen() && !m.ui.interactionModalOpen() && !m.ui.settingsWindow.IsOpen() && !m.ui.identifyWindow.IsOpen() && !m.ui.petEggWindow.IsOpen() && !m.ui.petConfirm.IsOpen() && !m.ui.companions.HomunculusConfirm.IsOpen() && !m.ui.companions.MercenaryConfirm.IsOpen() {
		m.updateCameraRotation(ctx)
	}
	if !dead && m.ui.escapeMenu.IsOpen() {
		if m.ui.escapeMenu.Update(ctx) {
			m.handleEscapeMenuAction(ctx)
			return nil, nil
		}
	}
	if m.ui.friendRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.friendConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.partyRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.guildRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.guildAllianceRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.guildRelationConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.partyInfo.Update(ctx) {
		return nil, nil
	}
	if m.ui.tradeRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.adoptionRequest.Update(ctx) {
		return nil, nil
	}
	if m.ui.starPlaceConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.petConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.petInfoWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusInfo.Update(ctx) {
		if action := m.ui.companions.HomunculusInfo.PopAction(); action.Kind != gameui.HomunculusInfoActionNone {
			m.companions().HandleHomunculusInfoAction(ctx, action)
		}
		return nil, nil
	}
	if m.ui.companions.MercenaryConfirm.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.MercenaryInfo.Update(ctx) {
		if action := m.ui.companions.MercenaryInfo.PopAction(); action.Kind != gameui.MercenaryInfoActionNone {
			m.companions().HandleMercenaryInfoAction(ctx, action)
		}
		return nil, nil
	}
	if m.ui.teleportModal.Update(ctx) {
		return nil, nil
	}
	if m.updateAutoSpellWindow(ctx) {
		return nil, nil
	}
	if m.ui.npcInputAvailable() && m.ui.npcDialog.Update(ctx) {
		return nil, nil
	}
	if m.ui.npcCutin.Update(ctx) {
		return nil, nil
	}
	if m.updateWhisperWindow(ctx) {
		return nil, nil
	}
	if m.updateChatRoomWindows(ctx) {
		return nil, nil
	}
	if m.updatePartyHelperWindows(ctx) {
		return nil, nil
	}
	if m.updateSkillTextPrompt(ctx) {
		return nil, nil
	}
	if m.updateGuildMemberPrompt(ctx) {
		return nil, nil
	}
	if m.ui.makingArrow.Update(ctx) {
		return nil, nil
	}
	if m.ui.makingItem.Update(ctx) {
		return nil, nil
	}
	if m.ui.repairItem.Update(ctx) {
		return nil, nil
	}
	if m.ui.weaponRefine.Update(ctx) {
		return nil, nil
	}
	// Let the atlas close on Escape before the console handles that key.
	if m.ui.worldMap.Update(ctx) {
		return nil, nil
	}
	if !dead && !m.ui.chatShortcuts.KeyboardShortcutsBlocked() && m.ui.console.UpdateInput(ctx) {
		return nil, nil
	}
	if m.ui.chatShortcuts.Update(ctx) {
		return nil, nil
	}
	if m.ui.settingsWindow.Update(ctx) {
		return nil, nil
	}
	// DOM item description/artwork panels sit above every game window in
	// the web build: their Escape (close topmost panel) must run before the
	// escape menu can consume the key. No-op on native.
	if gameui.UpdateItemInfoWebWindows(ctx) {
		return nil, nil
	}
	if !dead && m.ui.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return nil, nil
	}
	characterWindowConsumed := m.ui.characterWindow.Update(ctx)
	m.ui.basicMenu.FollowCharacterWindow(ctx, &m.ui.characterWindow)
	if characterWindowConsumed {
		return nil, nil
	}
	if consumed, err := m.ui.itemWindows.Update(ctx, m); consumed {
		if err != nil {
			m.ui.console.AddErrorMessage("Unable to display this item.")
			glog.Warnf("item window: %v", err)
		}
		return nil, nil
	}
	if m.ui.monsterInfoWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.identifyWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.cardWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.petEggWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.inventoryBag.UpdateDrag(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.tradeWindow, &m.ui.equipmentWindow, &m.ui.mailWindow) {
		return nil, nil
	}
	if m.ui.storageWindow.UpdateDrag(ctx, &m.ui.inventoryBag, &m.ui.cartWindow) {
		return nil, nil
	}
	if m.ui.cartWindow.UpdateDrag(ctx, &m.ui.inventoryBag, &m.ui.storageWindow) {
		return nil, nil
	}
	if m.ui.skillWindow.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.companions.HomunculusSkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.companions.MercenarySkill.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.guildWindow.UpdateDrag(ctx, &m.ui.shortcutBar) {
		return nil, nil
	}
	if m.ui.shortcutBar.Update(ctx, m) {
		return nil, nil
	}
	if m.ui.mailWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.inventoryBag.Update(ctx, &m.ui.shortcutBar, &m.ui.storageWindow, &m.ui.cartWindow, &m.ui.tradeWindow, &m.ui.equipmentWindow, &m.ui.itemWindows, &m.ui.mailWindow) {
		return nil, nil
	}
	if m.ui.tradeWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.equipmentWindow.Update(ctx, &m.ui.itemWindows, &m.ui.cartWindow, m) {
		return nil, nil
	}
	if m.ui.viewEquipWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.storageWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.cartWindow, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.cartWindow.Update(ctx, &m.ui.inventoryBag, &m.ui.storageWindow, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.changeCartWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.shopWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.vendingWindow.Update(ctx, &m.ui.itemWindows) {
		return nil, nil
	}
	if m.ui.skillWindow.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.questWindow.Update(ctx) {
		return nil, nil
	}
	if m.ui.companions.HomunculusSkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.companions.MercenarySkill.Update(ctx, &m.ui.shortcutBar, m) {
		return nil, nil
	}
	if m.ui.emoteWindow.Update(ctx, &m.ui.console) {
		return nil, nil
	}
	if m.ui.friendsWindow.Update(ctx) {
		switch action := m.ui.friendsWindow.PopAction(); action.Kind {
		case gameui.FriendsWindowActionPartySettings:
			m.ui.partySettings.Open(ctx)
		case gameui.FriendsWindowActionPartyCreate:
			m.ui.partyCreate.Open(ctx)
		case gameui.FriendsWindowActionPartyInvite:
			m.ui.partyInvite.Open(ctx)
		case gameui.FriendsWindowActionPartyLeave:
			if ctx.Network == nil {
				m.ui.console.AddErrorMessage("Leave party failed: not connected.")
			} else if err := ctx.Network.SendLeaveParty(); err != nil {
				m.ui.console.AddErrorMessage("Leave party failed.")
				glog.Warnf("leave party failed: %v", err)
			}
		case gameui.FriendsWindowActionFriendWhisper:
			m.ui.whisperWindows.Open(ctx, action.Friend.Name)
		case gameui.FriendsWindowActionFriendDelete:
			m.openDeleteFriendConfirm(ctx, action.Friend)
		case gameui.FriendsWindowActionFriendSettings:
			m.ui.friendSettings.Open(ctx)
		case gameui.FriendsWindowActionFriendBlockWhisper:
			name := strings.TrimSpace(action.Friend.Name)
			if ctx.Network == nil {
				m.ui.console.AddErrorMessage("Block whisper failed: not connected.")
			} else if err := ctx.Network.SendWhisperIgnore(name, false); err != nil {
				m.ui.console.AddErrorMessage("Block whisper failed.")
				glog.Warnf("block whisper failed name=%q: %v", name, err)
			}
		case gameui.FriendsWindowActionPartyMemberInfo:
			m.openPartyMemberInfo(ctx, action.PartyMember)
		case gameui.FriendsWindowActionPartyMemberWhisper:
			m.ui.whisperWindows.Open(ctx, action.PartyMember.Name)
		case gameui.FriendsWindowActionPartyMemberExpel:
			m.openExpelPartyMemberConfirm(ctx, action.PartyMember)
		}
		return nil, nil
	}
	if m.ui.guildWindow.Update(ctx, &m.ui.shortcutBar, m) {
		if action := m.ui.guildWindow.PopAction(); action.RequestMenu {
			m.requestGuildWindowTab(ctx, action.MenuTab)
		} else if action.SelectedEmblemPath != "" {
			m.uploadGuildEmblem(ctx, action.SelectedEmblemPath)
		} else if len(action.MemberPositions) > 0 {
			m.changeGuildMemberPositions(ctx, action.MemberPositions)
		} else if len(action.LevelUpSkillIDs) > 0 {
			m.levelUpGuildSkills(ctx, action.LevelUpSkillIDs)
		} else if action.UpdatePositions {
			m.updateGuildPositions(ctx, action.Positions)
		} else if action.UpdateNotice {
			m.updateGuildNotice(ctx, action.NoticeSubject, action.Notice)
		} else if action.DeleteRelation != nil {
			m.openDeleteGuildRelationConfirm(ctx, action.DeleteRelation.Relation)
		} else if action.MemberAction != nil {
			m.openGuildMemberPrompt(ctx, *action.MemberAction)
		}
		return nil, nil
	}
	if m.ui.friendSettings.Update(ctx) {
		return nil, nil
	}
	if m.ui.partySettings.Update(ctx) {
		return nil, nil
	}
	if m.ui.statsWindow.Update(ctx) {
		return nil, nil
	}
	m.drainCameraButtons(ctx)
	m.ui.basicMenu.DrainWebActions(ctx, m.basicMenuCallbacks(ctx))
	if !m.ui.keyboardInputBlocked(ctx) {
		if m.ui.basicMenu.Update(ctx, m.basicMenuCallbacks(ctx)) {
			return nil, nil
		}
	}
	minimapDragging := m.ui.minimap.Update(ctx)
	m.effects().SyncLevel99AuraEffects(ctx, now)
	pointerBlocked := minimapDragging || m.mapPointerBlocked(ctx)
	if !pointerBlocked {
		m.updateCameraZoom(ctx)
	}
	if dead {
		return nil, nil
	}
	m.companions().Update(ctx, m.actorDeaths, now)
	m.updateBot(ctx, now)

	leftClick := !pointerBlocked && ctx.Input.MouseJustPressed(input.MouseButtonLeft)
	if leftClick {
		// One-shot interactions must not be delayed by the short throttle used
		// for repeated walk requests. Scheduling the held-click repeat here also
		// prevents a throttled ground click from turning into a walk on the next
		// frame merely because the button is still down.
		m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)
	}
	if leftClick && m.pendingSkill.skill.ID != 0 {
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		m.skills().HandleClick(ctx, projection, now)
		return nil, nil
	}

	if leftClick {
		screenW, screenH := ctx.ScreenSize()
		projection := m.sceneProjection(ctx, screenW, screenH, now)
		if m.handlePetCaptureClick(ctx, projection, now) {
			return nil, nil
		}
		playerX, playerY := scene.CurrentPlayerCell(ctx, now)
		if actor, ok := m.hoveredVendingBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click vending target mouse=%d,%d id=%d name=%q shop=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.VendingName, playerX, playerY, actor.X, actor.Y)
			m.requestVendingList(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := m.hoveredChatRoomBoard(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click chat room target mouse=%d,%d id=%d name=%q room=%d title=%q player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.ChatRoomID, actor.ChatRoomTitle, playerX, playerY, actor.X, actor.Y)
			m.requestChatRoomEnter(ctx, actor)
			return nil, nil
		}
		if item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
			glog.Debugf("click pickup target mouse=%d,%d id=%d item_id=%d amount=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, item.ID, item.ItemID, item.Amount, playerX, playerY, item.X, item.Y)
			m.clearLockedAttack()
			m.clearAttackFocus()
			m.requestPickup(ctx, item, "click")
			return nil, nil
		}
		if actor, ok := hoveredCursorActor(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok && isWarpActor(actor) {
			m.clearLockedAttack()
			m.clearAttackFocus()
			if targetX, targetY, targetOK := warpWalkTarget(ctx, actor, now); targetOK {
				glog.Debugf("click warp target mouse=%d,%d id=%d job=%d player=%d,%d warp=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Job, playerX, playerY, actor.X, actor.Y, targetX, targetY)
				m.requestWalk(ctx, targetX, targetY, "warp click")
			} else {
				glog.Debugf("click warp target unreachable mouse=%d,%d id=%d job=%d player=%d,%d warp=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Job, playerX, playerY, actor.X, actor.Y)
			}
			return nil, nil
		}
		if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			glog.Debugf("click attack target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.requestAttack(ctx, actor, "click")
			return nil, nil
		}
		if actor, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
			glog.Debugf("click npc talk target mouse=%d,%d id=%d name=%q job=%d object_type=%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, actor.Job, actor.ObjectType, playerX, playerY, actor.X, actor.Y)
			m.clearAttackFocus()
			// The AEOPD registry portal opens the DOM window instead of a
			// server dialog (no-op on native).
			if sarabanPortalActor(actor) && openSarabanWeb() {
				return nil, nil
			}
			m.requestNPCTalk(ctx, actor, "click")
			return nil, nil
		}
		if targetX, targetY, ok := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok && m.walkReady(now) {
			glog.Debugf("click walk target mouse=%d,%d player=%d,%d target=%d,%d", ctx.Input.MouseX, ctx.Input.MouseY, playerX, playerY, targetX, targetY)
			// Walking somewhere else is also the cancel gesture for an
			// in-flight action: the attack intent clears above, and a
			// pending floor-item pickup must stop chasing its item.
			m.pendingPickup = pickupIntent{}
			m.cancelAttackIntent()
			if shouldUseTurnOnlyGroundClick(ctx) {
				m.requestChangeDirection(ctx, targetX, targetY, "click")
				return nil, nil
			}
			m.requestWalk(ctx, targetX, targetY, "click")
		}
	}
	if m.updateHeldWalk(ctx, pointerBlocked, now) {
		return nil, nil
	}
	return nil, nil
}

func (m *WorldMode) handleEscapeMenuAction(ctx client.Context) {
	switch m.ui.escapeMenu.ConsumeAction() {
	case gameui.EscapeMenuActionAutoRevive:
		m.ui.escapeMenu.RequestAutoRevive(ctx)
	case gameui.EscapeMenuActionSavePoint:
		m.ui.escapeMenu.ReturnToSavePoint(ctx)
	case gameui.EscapeMenuActionCharacterSelect:
		m.ui.escapeMenu.RequestCharacterSelect(ctx)
	case gameui.EscapeMenuActionSettings:
		m.ui.settingsWindow.OpenWindow(ctx)
	}
}

func (m *WorldMode) updateDeathUIInput(ctx client.Context) bool {
	// Active chat owns Escape so the first press leaves text entry. Enter also
	// gets priority so chat can be activated even while the pointer is over the
	// death menu. Otherwise the menu owns Escape and may be freely toggled.
	chatFirst := m.ui.console.Active() || (ctx.Input != nil && ctx.Input.JustPressed(input.KeyEnter))
	if chatFirst && m.ui.console.UpdateInput(ctx) {
		return true
	}
	if m.ui.escapeMenu.Update(ctx) {
		m.handleEscapeMenuAction(ctx)
		return true
	}
	return !chatFirst && m.ui.console.UpdateInput(ctx)
}

func (m *WorldMode) applyQuitGameAck(ctx client.Context, ack network.QuitGameAck) {
	handled := m.ui.escapeMenu.ApplyQuitGameAck(ack)
	if ack.Allowed {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
		return
	}
	if handled {
		m.addLeaveWorldRefusalMessage(ctx)
	}
}

func (m *WorldMode) addLeaveWorldRefusalMessage(ctx client.Context) {
	message := login.MessageText(ctx.Resources, 502, "You cannot exit the game right now.")
	m.ui.console.AddErrorMessage("%s", message)
}

func (m *WorldMode) openEscapeMenuFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.escapeMenu.IsOpen() || !ctx.Input.JustPressed(input.KeyEscape) {
		return false
	}
	if m.ui.interactionModalOpen() {
		return false
	}
	m.ui.escapeMenu.Toggle(ctx)
	return true
}

func (m *WorldMode) basicMenuCallbacks(ctx client.Context) gameui.BasicMenuCallbacks {
	return gameui.BasicMenuCallbacks{
		OnStatus: func() { m.ui.statsWindow.Toggle(ctx) },
		OnOption: func() { m.ui.escapeMenu.Toggle(ctx) },
		OnItems:  func() { m.ui.inventoryBag.Toggle(ctx) },
		OnEquip:  func() { m.ui.equipmentWindow.Toggle(ctx) },
		OnSkill:  func() { m.ui.skillWindow.Toggle(ctx) },
		OnMap:    func() { m.ui.minimap.Toggle(ctx) },
		OnComm:   func() { m.ui.chatRoomCreate.Open(ctx) },
		OnFriend: func() { m.ui.friendsWindow.Toggle(ctx) },
	}
}

func (m *WorldMode) toggleEmoteWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	if !plainAltDown(ctx.Input) || !ctx.Input.JustPressed(input.KeyL) {
		return false
	}
	m.ui.emoteWindow.Toggle(ctx, &m.ui.console)
	return true
}

func (m *WorldMode) toggleGuildWindowFromInput(ctx client.Context) bool {
	if ctx.Input == nil || m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	if !plainAltDown(ctx.Input) || !ctx.Input.JustPressed(input.KeyG) {
		return false
	}
	m.toggleGuildWindow(ctx)
	return true
}

func (m *WorldMode) toggleGuildWindow(ctx client.Context) {
	if localGuildIDFromSession(ctx.Session) == 0 {
		return
	}
	if m.ui.guildWindow.IsOpen() {
		m.ui.guildWindow.Close()
		return
	}
	m.openGuildWindow(ctx)
}

func (m *WorldMode) openGuildWindow(ctx client.Context) {
	if localGuildIDFromSession(ctx.Session) == 0 || m.ui.guildWindow.IsOpen() {
		return
	}
	m.setGuildEmblemOptions(ctx)
	m.ui.guildWindow.OpenWindow(ctx)
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendGuildMenuInterfaceRequest(); err != nil {
		m.ui.console.AddErrorMessage("Guild access request failed.")
		glog.Warnf("guild menu access request failed: %v", err)
	}
	m.requestGuildWindowTab(ctx, 0)
}

const maxGuildMenuRequestType = 4

func (m *WorldMode) requestGuildWindowTab(ctx client.Context, tab uint32) {
	if tab > maxGuildMenuRequestType {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild info request failed.")
		return
	}
	if err := ctx.Network.SendGuildMenuRequest(tab); err != nil {
		m.ui.console.AddErrorMessage("Guild info request failed.")
		glog.Warnf("guild menu request failed tab=%d: %v", tab, err)
	}
	if tab == 0 {
		m.requestSessionGuildEmblem(ctx)
	}
}

func (m *WorldMode) requestSessionGuildEmblem(ctx client.Context) {
	if ctx.Session == nil {
		return
	}
	guildID := ctx.Session.GuildID
	version := ctx.Session.EmblemVersion
	if guildID == 0 {
		guildID = ctx.Session.Guild.ID
	}
	if version == 0 {
		version = ctx.Session.Guild.EmblemVersion
	}
	m.requestGuildEmblem(ctx, guildID, version, true)
}

func (m *WorldMode) handleMapChange(ctx client.Context, change network.MapChange) Mode {
	if m.ui.mailWindow.IsOpen() || m.mail.pending.Kind != gameui.MailActionNone {
		m.closeMail(ctx)
	}
	m.clearServerProgress()
	m.showDigit = showDigitState{}
	m.ui.minimap.ClearBossMarker()
	m.ui.poptips.Clear()
	m.pendingAttack = attackIntent{}
	m.clearLockedAttack()
	m.clearAttackFocus()
	m.clearLocalActorAction(ctx)
	m.scheduledStops = nil
	m.ui.npcDialog.ResetPublished(ctx)
	m.ui.npcCutin.Clear()
	m.ui.teleportModal = gameui.TeleportModal{}
	m.ui.autoSpellWindow.Reset(ctx)
	if m.ui.starPlaceConfirm.IsOpen() {
		m.ui.starPlaceConfirm.Close(ctx)
	}
	m.clearLocalDeathState(ctx)
	currentMap := ctx.World.MapName
	reuseLoadedMap := !change.ServerMove && sameLoadedMap(ctx, change.MapName)
	glog.Debugf("map change current=%s target=%s x=%d y=%d server_move=%t addr=%s port=%d reuse_loaded=%t", currentMap, change.MapName, change.X, change.Y, change.ServerMove, change.Address, change.Port, reuseLoadedMap)
	ctx.World.ResetMapProperty()
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	applyWarpPosition(ctx, change.X, change.Y)
	ctx.World.Actors = make(map[uint32]worldstate.Actor)
	if reuseLoadedMap {
		m.camera.ResetTracking()
		m.camera.Update(ctx, time.Now())
		if ctx.Network != nil {
			if err := ctx.Network.SendLoadEndAck(); err != nil {
				glog.Warnf("same-map warp load ack failed map=%s x=%d y=%d: %v", change.MapName, change.X, change.Y, err)
			}
		}
		return nil
	}
	if change.ServerMove {
		m.mail = mailState{}
		m.deferredPackets = nil
		ctx.Session.Zone.Address = change.Address
		ctx.Session.Zone.Port = change.Port
		dialCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := ctx.Network.Connect(dialCtx, change.Address, int(change.Port))
		cancel()
		if err != nil {
			glog.Warnf("map reconnect failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			login.OpenConnectionFailedDialog(ctx, &m.ui.disconnectDialog)
			return nil
		}
		if err := ctx.Network.SendMapServerEnter(ctx.Session.AccountID, ctx.Session.CharID, ctx.Session.AuthCode, uint32(time.Now().UnixMilli()), ctx.Session.Sex); err != nil {
			glog.Warnf("map re-enter failed map=%s addr=%s port=%d: %v", change.MapName, change.Address, change.Port, err)
			message := login.MessageText(ctx.Resources, 2, "Disconnected from Server.")
			login.OpenDisconnectDialog(ctx, &m.ui.disconnectDialog, message)
			return nil
		}
		m.pendingWarp = true
		return nil
	}
	return m.nextWorldMode()
}

func (m *WorldMode) handleLevelUpNotificationAction(ctx client.Context, action gameui.LevelUpNotificationAction) bool {
	if action&gameui.LevelUpNotificationBase != 0 {
		m.ui.statsWindow.OpenWindow(ctx)
	}
	if action&gameui.LevelUpNotificationJob != 0 {
		m.ui.skillWindow.OpenWindow(ctx)
	}
	return action != gameui.LevelUpNotificationNone
}

func (m *WorldMode) nextWorldMode() *WorldMode {
	next := NewWorldMode()
	next.mail = m.mail
	next.deferredPackets = m.deferredPackets
	next.camera.yawOffset = m.camera.yawOffset
	next.camera.pitch = m.camera.pitch
	next.camera.zoom = m.camera.zoom
	next.camera.zoomTarget = m.camera.zoomTarget
	next.ui.console = m.ui.console
	next.ui.characterWindow = m.ui.characterWindow
	next.ui.basicMenu = m.ui.basicMenu
	next.ui.inventoryBag = m.ui.inventoryBag
	next.ui.equipmentWindow = m.ui.equipmentWindow
	next.ui.cartWindow = m.ui.cartWindow
	next.ui.itemWindows = m.ui.itemWindows
	next.ui.cardWindow = m.ui.cardWindow
	next.ui.petEggWindow = m.ui.petEggWindow
	next.ui.petInfoWindow = m.ui.petInfoWindow
	next.ui.companions.HomunculusInfo = m.ui.companions.HomunculusInfo
	next.ui.companions.HomunculusSkill = m.ui.companions.HomunculusSkill
	next.ui.companions.MercenaryInfo = m.ui.companions.MercenaryInfo
	next.ui.companions.MercenarySkill = m.ui.companions.MercenarySkill
	next.petProperty = m.petProperty
	next.hasPetProperty = m.hasPetProperty
	next.petOldFullness = m.petOldFullness
	next.petLastTalk = m.petLastTalk
	next.ui.statsWindow = m.ui.statsWindow
	next.ui.skillWindow = m.ui.skillWindow
	next.ui.questWindow = m.ui.questWindow
	next.ui.worldMap = m.ui.worldMap
	next.ui.emoteWindow = m.ui.emoteWindow
	next.ui.chatShortcuts = m.ui.chatShortcuts
	next.ui.friendsWindow = m.ui.friendsWindow
	next.ui.guildWindow = m.ui.guildWindow
	next.ui.friendSettings = m.ui.friendSettings
	next.ui.whisperWindows = m.ui.whisperWindows
	next.ui.chatRoomCreate = m.ui.chatRoomCreate
	next.ui.chatRoom = m.ui.chatRoom
	next.pendingChatRoom = m.pendingChatRoom
	next.ui.partySettings = m.ui.partySettings
	next.ui.settingsWindow = m.ui.settingsWindow
	next.ui.partyCreate = m.ui.partyCreate
	next.ui.partyInvite = m.ui.partyInvite
	next.ui.shortcutBar = m.ui.shortcutBar
	next.ui.minimap = m.ui.minimap
	next.ui.statusIcons = m.ui.statusIcons
	next.ui.pvpCounter = m.ui.pvpCounter
	next.ui.perfHUD = m.ui.perfHUD
	next.ui.levelUpNotifications = m.ui.levelUpNotifications
	m.companionSystem.Close()
	return next
}

func (m *WorldMode) nextCharacterSelectMode(ctx client.Context) *LoginMode {
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	if ctx.Session != nil {
		ctx.Session.Playing = false
		ctx.Session.Storage = session.Storage{}
		ctx.Session.Cart = session.Cart{}
	}
	next := NewCharacterSelectMode(ctx, m.ui.console)
	return next
}

func (m *WorldMode) updateDisconnectDialog(ctx client.Context) bool {
	if !m.ui.disconnectDialog.IsOpen() {
		return false
	}
	return m.ui.disconnectDialog.Update(ctx)
}

func sameLoadedMap(ctx client.Context, mapName string) bool {
	if ctx.World == nil || ctx.World.MapName == "" || mapName == "" {
		return false
	}
	if !strings.EqualFold(ctx.World.MapName, mapName) {
		return false
	}
	return ctx.World.GND != nil || ctx.World.GAT != nil
}

func (m *WorldMode) requestNPCTalk(ctx client.Context, actor worldstate.Actor, source string) {
	if playerIsDead(ctx) {
		return
	}
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	m.clearLockedAttack()
	if err := ctx.Network.SendNPCContact(actor.ID); err == nil {
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		playerX, playerY := scene.CurrentPlayerCell(ctx, time.Now())
		glog.Warnf("%s npc talk failed target=%d player=%d,%d target=%d,%d: %v", source, actor.ID, playerX, playerY, actor.X, actor.Y, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) Draw(ctx client.Context, screen *render.Frame) {
	render.SetWebLoading(false)
	if m.mapScene.Images.PrepareUploads(screen) {
		maps.ClearWorldScene(screen, ctx.World.MapName)
		return
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	fog := weather.FogFromMap(ctx.Resources, ctx.World.MapName, ctx.Config.Fog)
	var actorOverlays []scene.ActorDrawEntry
	screen.SetCamera3D(projection.RenderCameraWithFog(fog))
	vertexFog := scene.SceneFog{}

	if ctx.World.GND != nil {
		m.mapScene.DrawGNDMeshes(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection)
		m.mapScene.DrawGNDWater(screen, ctx.Resources, ctx.World.GND, ctx.World.RSW, projection, now, vertexFog)
		m.drawGraffiti(screen, ctx, projection)
		if !ctx.Config.Render.NoUI {
			m.drawTileCursor(screen, ctx, projection)
		}
		if ctx.World.RSW != nil && len(ctx.World.RSM) > 0 {
			actorOverlays = m.drawSceneModelsAndActors(screen, ctx, projection, now)
		} else {
			m.drawSkillUnitRSMModels(screen, ctx, projection, now)
			m.effects().Draw(screen, ctx, projection, now, true)
			m.drawGroundItems(screen, ctx, projection, now)
			actorOverlays = m.drawSceneActors(screen, ctx, projection)
		}
	} else if ctx.World.GAT != nil {
		m.effects().Draw(screen, ctx, projection, now, true)
		m.drawGroundItems(screen, ctx, projection, now)
		actorOverlays = m.drawSceneActors(screen, ctx, projection)
	} else {
		m.effects().Draw(screen, ctx, projection, now, true)
	}

	if !ctx.Config.Render.NoUI {
		m.drawSceneActorOverlays(screen, ctx, projection, now, actorOverlays)
	}
	m.mapScene.DrawRSWEffects(screen, ctx, projection, now, m.effects)
	m.weather.Draw(screen, ctx, projection, now, m.effects(), m.scheduleSound)
	m.effects().Draw(screen, ctx, projection, now, false)
	m.drawDamageFloaters(screen, ctx, projection, now)

	if !ctx.Config.Render.NoUI {
		m.ui.npcCutin.Draw(screen)
		m.ui.inventoryBag.Publish(ctx)
		m.ui.storageWindow.Publish(ctx)
		m.ui.cartWindow.Publish(ctx)
		m.ui.shopWindow.Publish(ctx)
		m.ui.vendingWindow.Publish(ctx)
		m.ui.skillWindow.Publish(ctx)
		m.ui.companions.HomunculusSkill.Publish(ctx)
		m.ui.companions.MercenarySkill.Publish(ctx)
		m.drawHoveredGroundItemLabel(screen, ctx, projection, now)
	}
}

func (m *WorldMode) DrawOverlay(ctx client.Context, screen *render.Frame) {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	now := time.Now()
	projection := m.sceneProjection(ctx, width, height, now)
	m.mapFade.Draw(screen, now)
	if !ctx.Config.Render.NoUI {
		m.drawUIDragGhosts(screen, ctx)
		m.drawPetSlotMachine(screen, ctx, now)
		m.drawROCursor(screen, ctx, projection, now)
	}
}

func (m *WorldMode) FrameSubmitted() {
	if m.mapScene.Images.FrameSubmitted() {
		return
	}
	m.mapFade.RecordCoveredFrame()
}

func (m *WorldMode) DrawUIOverlay(ctx client.Context, screen *render.Frame) {
	if ctx.Config.Render.NoUI {
		return
	}
	now := time.Now()
	m.drawShowDigit(screen, ctx, now)
	m.ui.characterWindow.Draw(screen, ctx)
	m.ui.announcement.Draw(screen, now)
	m.drawPendingTargetBanner(screen, ctx)
	m.ui.poptips.Draw(screen, now)
	m.ui.inventoryBag.DrawTooltip(ctx, screen)
	m.ui.equipmentWindow.DrawTooltip(ctx, screen)
	m.ui.cartWindow.DrawTooltip(ctx, screen)
	m.ui.itemWindows.DrawTooltip(ctx, screen)
	m.ui.skillWindow.DrawTooltip(ctx, screen)
	m.ui.companions.HomunculusSkill.DrawTooltip(ctx, screen)
	m.ui.companions.MercenarySkill.DrawTooltip(ctx, screen)
	m.ui.guildWindow.DrawTooltip(ctx, screen)
	m.ui.shortcutBar.DrawTooltip(ctx, screen)
	m.ui.itemPickup.Draw(screen, ctx, m, now)
}

func (m *WorldMode) drawUIDragGhosts(screen *render.Frame, ctx client.Context) {
	m.ui.inventoryBag.DrawDragGhost(screen, ctx, m)
	m.ui.storageWindow.DrawDragGhost(screen, ctx, m)
	m.ui.cartWindow.DrawDragGhost(screen, ctx, m)
	m.ui.shopWindow.DrawDragGhost(screen, ctx, m)
	m.ui.vendingWindow.DrawDragGhost(screen, ctx, m)
	m.ui.skillWindow.DrawDragGhost(screen, ctx, m)
	m.ui.companions.HomunculusSkill.DrawDragGhost(screen, ctx, m)
	m.ui.companions.MercenarySkill.DrawDragGhost(screen, ctx, m)
	m.ui.guildWindow.DrawDragGhost(screen, ctx, m)
}

func clickedAttackTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actorCanBeAttackClicked(ctx, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedSkillTarget(ctx client.Context, projection scene.Projection, skill session.Skill, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range skillTargetCandidates(ctx, skill) {
		if _, dead := deadActors[actor.ID]; dead && !skillCanTargetDeadActor(skill, actor) {
			continue
		}
		if !actorCanBeSkillTargeted(ctx, skill, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func clickedPlayerTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if !actorCanOpenPlayerContext(ctx, actor) {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !pointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func skillTargetCandidates(ctx client.Context, skill session.Skill) []worldstate.Actor {
	if ctx.World == nil {
		return nil
	}
	candidates := make([]worldstate.Actor, 0, len(ctx.World.Actors)+1)
	if skillTargetFlagsIncludeSelfPick(skill.Type) {
		if actor, ok, _ := scene.ActorForCombatID(ctx, scene.LocalSkillTarget(ctx)); ok {
			candidates = append(candidates, actor)
		}
	}
	for _, actor := range ctx.World.Actors {
		candidates = append(candidates, actor)
	}
	return candidates
}

func skillTargetFlagsIncludeSelfPick(flags uint32) bool {
	return flags&(skillTargetFriend|skillTargetSelf) != 0
}

func clampGameInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

type sceneDrawEntry struct {
	depth           float64
	entityID        uint32
	actorIndex      int
	shadowIndex     int
	itemIndex       int
	itemShadowIndex int
}

func (m *WorldMode) drawSceneModelsAndActors(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) []scene.ActorDrawEntry {
	m.mapScene.DrawRSMModels(screen, ctx.Resources, ctx.World.RSW, ctx.World.RSM, ctx.World.GND, projection, now)
	m.drawSkillUnitRSMModels(screen, ctx, projection, now)
	m.effects().Draw(screen, ctx, projection, now, true)
	actors := m.collectSceneActorEntries(screen, ctx, projection)
	items := m.collectSceneItemEntries(screen, ctx, projection, now)
	for _, entry := range sortedSceneDrawEntries(actors, items) {
		if entry.itemShadowIndex >= 0 {
			m.drawGroundItemShadowEntry3D(screen, projection, items[entry.itemShadowIndex])
			continue
		}
		if entry.shadowIndex >= 0 {
			m.drawActorShadowEntry(screen, ctx, projection, actors[entry.shadowIndex])
			continue
		}
		if entry.itemIndex >= 0 {
			m.drawGroundItemEntry3D(screen, projection, items[entry.itemIndex])
			continue
		}
		m.drawSceneActorEntry(screen, ctx, projection, actors[entry.actorIndex])
	}
	m.drawSceneActorFalcons(screen, ctx, projection, actors)
	return actors
}

func sortedSceneDrawEntries(actors []scene.ActorDrawEntry, items []sceneItemDrawEntry) []sceneDrawEntry {
	entries := make([]sceneDrawEntry, 0, len(actors)*2+len(items)*2)
	for i, item := range items {
		entries = append(entries,
			sceneDrawEntry{depth: item.shadowDepth, entityID: item.item.ID, actorIndex: -1, shadowIndex: -1, itemIndex: -1, itemShadowIndex: i},
			sceneDrawEntry{depth: item.depth, entityID: item.item.ID, actorIndex: -1, shadowIndex: -1, itemIndex: i, itemShadowIndex: -1},
		)
	}
	for i, actor := range actors {
		if actor.CastShadow {
			entries = append(entries, sceneDrawEntry{depth: actor.ShadowDepth, entityID: actor.Actor.ID, actorIndex: -1, shadowIndex: i, itemIndex: -1, itemShadowIndex: -1})
		}
		entries = append(entries, sceneDrawEntry{depth: actor.Depth, entityID: actor.Actor.ID, actorIndex: i, shadowIndex: -1, itemIndex: -1, itemShadowIndex: -1})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		// A stable sort alone preserves the world's random map iteration order
		// at equal depths, making overlapping loot flicker from frame to frame.
		if entries[i].depth == entries[j].depth {
			return entries[i].entityID < entries[j].entityID
		}
		return entries[i].depth > entries[j].depth
	})
	return entries
}

func (m *WorldMode) releaseMapTextures() {
	for id := range m.graffiti {
		m.removeGraffiti(id)
	}
	m.graffiti = nil
	if m.spriteUpscaler != nil {
		m.spriteUpscaler.Close()
		m.spriteUpscaler = nil
	}
	m.mapScene.Images.Release()
}

// Leave also covers disconnects, character selection, and failed map loads.
func (m *WorldMode) Leave() {
	m.releaseMapTextures()
}

func (m *WorldMode) startMapFadeOut(change network.MapChange, now time.Time) {
	m.ui.npcCutin.Clear()
	// A queued cell (or message) belongs to the map where it was selected,
	// even when the server warps us within the same loaded map.
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	if m.ui.skillTextPrompt.IsOpen() {
		m.ui.skillTextPrompt.Close()
	}
	m.mapFade.StartOut(change, now)
}

func (m *WorldMode) startCharacterSelectFadeOut(now time.Time) {
	m.ui.npcCutin.Clear()
	m.mapFade.StartCharacterSelectOut(now)
}
