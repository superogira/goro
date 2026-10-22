package game

import (
	"fmt"
	"time"


	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/login"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

type LoginMode struct {
	*login.Screen
	console gameui.ChatConsole
	cursor  roCursorState
}

func (m *LoginMode) bindings() login.Bindings {
	return login.Bindings{
		Console: &m.console,
		ApplyWorldPacket: func(ctx client.Context, pkt network.Packet) (bool, error) {
			return applyLoginWorldPacket(ctx, pkt, &m.console)
		},
		EnterMap: func(ctx client.Context, enter network.MapAcceptEnter) {
			applyMapAcceptEnter(ctx, enter)
			sendLessEffectPreference(ctx)
		},
		ChangeMap: applyLoginMapChange,
		Preview:   loginCharacterPreview,
	}
}

func NewLoginMode() *LoginMode {
	m := &LoginMode{}
	m.Screen = login.New(m.bindings())
	return m
}

func NewCharacterSelectMode(ctx client.Context, console gameui.ChatConsole) *LoginMode {
	m := &LoginMode{console: console}
	m.Screen = login.NewCharacterSelect(ctx, m.bindings())
	return m
}

func (m *LoginMode) Enter(ctx client.Context) Mode {
	if !ctx.Config.Headless {
		m.cursor.ensureLoaded(ctx)
		render.SetCursorMode(render.CursorModeHidden)
	}
	m.Screen.Enter(ctx)
	return nil
}

func (m *LoginMode) Update(ctx client.Context) (Mode, error) {
	ready, err := m.Screen.Update(ctx)
	if err != nil {
		return nil, err
	}
	if ready {
		return m.nextWorldMode(ctx), nil
	}
	return nil, nil
}


func (m *LoginMode) DrawOverlay(ctx client.Context, screen *render.Frame) {
	m.Screen.DrawOverlay(ctx, screen)
	if !ctx.Config.Render.NoUI {
		m.drawROCursor(screen, ctx, time.Now())
	}
}

func (m *LoginMode) nextWorldMode(ctx client.Context) *WorldMode {
	m.Finish(ctx)
	next := NewWorldMode()
	next.ui.console = m.console
	return next
}

func (m *LoginMode) controlScriptUI(ctx client.Context, action string) bool {
	return m.ControlUI(ctx, action)
}

func loginCharacterPreview(ctx client.Context, character session.Character, direction int) *scene.Billboard {
	character = characterWithVisualJob(character)
	view, status := sprites.LoadPlayer(ctx.Resources, character, ctx.Session.Sex, false)
	if view == nil {
		glog.Debugf("login sprite resources char_id=%d name=%s job=%d %s", character.ID, character.Name, character.Job, status)
		return nil
	}
	now := time.Now()
	billboard, _ := sprites.HumanoidBillboard(view, sprites.State{ActionFamily: scene.SpriteActionIdle, Direction: direction, Started: now, LoopIdle: true}, now)
	return billboard
}

func (m *LoginMode) drawROCursor(screen *render.Frame, ctx client.Context, now time.Time) {
	if ctx.Input == nil {
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	m.cursor.draw(screen, ctx, m.cursorAction(ctx), now, 0, 0)
}

func (m *LoginMode) cursorAction(ctx client.Context) int {
	if ctx.Input == nil {
		return cursorActionDefault
	}
	if action, ok := uiCursorAction(ctx); ok {
		return action
	}
	return cursorActionDefault
}

func applyLoginWorldPacket(ctx client.Context, pkt network.Packet, console *gameui.ChatConsole) (bool, error) {
	if handled, err := applyLoginParameterChange(ctx, pkt); handled {
		return true, err
	}
	if handled, err := applyLoginActorBootstrapPacket(ctx, pkt); handled {
		return true, err
	}
	if handled, err := applyLoginStatusEffectPacket(ctx, pkt); handled {
		return true, err
	}
	if handled, err := applyLoginCartPacket(ctx, pkt); handled {
		return true, err
	}
	if hotkeys, ok, err := network.ParseHotkeyList(pkt); err != nil {
		return true, fmt.Errorf("parse hotkey list: %w", err)
	} else if ok {
		applyHotkeyList(ctx, hotkeys)
		return true, nil
	}
	if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
		return true, fmt.Errorf("parse chat message: %w", err)
	} else if ok {
		addConsoleMessage(console, ctx.Resources, chat)
		return true, nil
	}
	if whisper, ok, err := network.ParseWhisperMessage(pkt); err != nil {
		return true, fmt.Errorf("parse whisper message: %w", err)
	} else if ok {
		addWhisperMessage(console, whisper)
		return true, nil
	}
	if ack, ok, err := network.ParseWhisperAck(pkt); err != nil {
		return true, fmt.Errorf("parse whisper ack: %w", err)
	} else if ok {
		addWhisperAck(console, ctx.Resources, ack)
		return true, nil
	}
	if ack, ok, err := network.ParseWhisperIgnoreAck(pkt); err != nil {
		return true, fmt.Errorf("parse whisper ignore ack: %w", err)
	} else if ok {
		addWhisperIgnoreAck(console, ack)
		return true, nil
	}
	if friends, ok, err := network.ParseFriendsList(pkt); err != nil {
		return true, fmt.Errorf("parse friends list: %w", err)
	} else if ok {
		applyFriendsList(ctx, friends)
		return true, nil
	}
	if friendState, ok, err := network.ParseFriendState(pkt); err != nil {
		return true, fmt.Errorf("parse friend state: %w", err)
	} else if ok {
		applyFriendState(ctx, friendState)
		return true, nil
	}
	if board, ok, err := network.ParseVendingBoard(pkt); err != nil {
		return true, fmt.Errorf("parse vending board: %w", err)
	} else if ok {
		applyVendingBoardToWorld(ctx, board)
		glog.Debugf("login vending board actor=%d name=%q", board.OwnerAID, board.Name)
		return true, nil
	}
	if board, ok, err := network.ParseVendingBoardDisappear(pkt); err != nil {
		return true, fmt.Errorf("parse vending board disappear: %w", err)
	} else if ok {
		applyVendingBoardDisappearToWorld(ctx, board)
		glog.Debugf("login vending board removed actor=%d", board.OwnerAID)
		return true, nil
	}
	if board, ok, err := network.ParseChatRoomBoard(pkt); err != nil {
		return true, fmt.Errorf("parse chat room board: %w", err)
	} else if ok {
		applyChatRoomBoardToWorld(ctx, board)
		glog.Debugf("login chat room board actor=%d room=%d title=%q", board.OwnerID, board.RoomID, board.Title)
		return true, nil
	}
	if destroy, ok, err := network.ParseChatRoomDestroy(pkt); err != nil {
		return true, fmt.Errorf("parse chat room destroy: %w", err)
	} else if ok {
		applyChatRoomDestroyToWorld(ctx, destroy)
		glog.Debugf("login chat room board removed room=%d", destroy.RoomID)
		return true, nil
	}
	if entry, ok, err := network.ParseActorEntry(pkt); err != nil {
		return true, fmt.Errorf("parse actor entry: %w", err)
	} else if ok {
		upsertNetworkActor(ctx, entry)
		return true, nil
	}
	return false, nil
}

func applyLoginParameterChange(ctx client.Context, pkt network.Packet) (bool, error) {
	if change, ok, err := network.ParseCoupleStatus(pkt); err != nil {
		return true, fmt.Errorf("parse couple status: %w", err)
	} else if ok {
		applyCoupleStatus(ctx, change)
		return true, nil
	}
	change, ok, err := network.ParseParameterChange(pkt)
	if err != nil {
		return true, fmt.Errorf("parse parameter change: %w", err)
	}
	if !ok {
		return false, nil
	}
	applyParameterChange(ctx, change)
	return true, nil
}

func applyLoginActorBootstrapPacket(ctx client.Context, pkt network.Packet) (bool, error) {
	if update, ok, err := network.ParseNPCShowEffectUpdate(pkt); err != nil {
		return true, fmt.Errorf("parse NPC effect update: %w", err)
	} else if ok {
		applyNPCShowEffectUpdate(ctx, update)
		return true, nil
	}
	if spheres, ok, err := network.ParseSpiritSphereCount(pkt); err != nil {
		return true, fmt.Errorf("parse spirit spheres: %w", err)
	} else if ok {
		applySpiritSphereCount(ctx, spheres)
		return true, nil
	}
	if change, ok, err := network.ParseNPCSpriteChange(pkt); err != nil {
		return true, fmt.Errorf("parse NPC sprite change: %w", err)
	} else if ok {
		applyNPCSpriteChange(ctx, change)
		return true, nil
	}
	if look, ok, err := network.ParseActorLookChange(pkt); err != nil {
		return true, fmt.Errorf("parse actor look change: %w", err)
	} else if ok {
		applyActorLookChange(ctx, look)
		return true, nil
	}
	if state, ok, err := network.ParseActorStateChange(pkt); err != nil {
		return true, fmt.Errorf("parse actor state change: %w", err)
	} else if ok {
		applyActorStateSnapshot(ctx, state)
		return true, nil
	}
	return false, nil
}

func applyLoginStatusEffectPacket(ctx client.Context, pkt network.Packet) (bool, error) {
	change, ok, err := network.ParseStatusEffectChange(pkt)
	if err != nil {
		return true, fmt.Errorf("parse status effect change: %w", err)
	}
	if !ok {
		return false, nil
	}
	applyActorOpt3StateStatus(ctx, change)
	if applyLocalStatusEffectChange(ctx.Session, change, time.Now()) {
		glog.Debugf("login status effect id=%d actor=%d active=%t duration_ms=%d", change.StatusID, change.ActorID, change.Active, change.Duration.Milliseconds())
	}
	return true, nil
}

func applyLoginCartPacket(ctx client.Context, pkt network.Packet) (bool, error) {
	if cartItems, ok, err := network.ParseCartItemList(pkt); err != nil {
		return true, fmt.Errorf("parse cart item list: %w", err)
	} else if ok {
		glog.Debugf("login cart item list items=%d", len(cartItems))
		applyCartItemList(ctx, cartItems)
		return true, nil
	}
	if cartAmount, ok, err := network.ParseCartAmount(pkt); err != nil {
		return true, fmt.Errorf("parse cart amount: %w", err)
	} else if ok {
		glog.Debugf("login cart amount count=%d/%d weight=%d/%d", cartAmount.Amount, cartAmount.MaxAmount, cartAmount.Weight, cartAmount.MaxWeight)
		applyCartAmount(ctx, cartAmount)
		return true, nil
	}
	if cartItem, ok, err := network.ParseCartItemAdded(pkt); err != nil {
		return true, fmt.Errorf("parse cart item added: %w", err)
	} else if ok {
		glog.Debugf("login cart item added index=%d item=%d amount=%d", cartItem.Index, cartItem.ItemID, cartItem.Amount)
		applyCartItemAdded(ctx, cartItem)
		return true, nil
	}
	if cartItem, ok, err := network.ParseCartItemRemoved(pkt); err != nil {
		return true, fmt.Errorf("parse cart item removed: %w", err)
	} else if ok {
		applyCartItemRemoved(ctx, cartItem)
		return true, nil
	}
	if network.ParseCartClosed(pkt) {
		applyCartClosed(ctx)
		return true, nil
	}
	return false, nil
}

func applyLoginMapChange(ctx client.Context, change network.MapChange) {
	ctx.World.ResetMapProperty()
	ctx.World.MapName = change.MapName
	ctx.Session.Zone.MapName = change.MapName
	resetWorldForSelectedCharacterIfNeeded(ctx)
	applyWarpPosition(ctx, change.X, change.Y)
	ctx.Session.Playing = true
}

// LoginCredentials returns the current credential values (OSK submit sync).
func (m *LoginMode) LoginCredentials() (string, string) {
	return m.Screen.LoginCredentials()
}

// LoginKeepID reports the Keep-ID state.
func (m *LoginMode) LoginKeepID() bool {
	return m.Screen.LoginKeepID()
}
