package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestLoginModeAppliesEarlySpeedParameterChange(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{},
		World:   world,
	}

	if handled, err := applyLoginWorldPacket(ctx, testParameterChangePacket(network.StatusSpeed, 112), &mode.console); !handled || err != nil {
		t.Fatal("speed parameter change was not handled")
	}
	if !ctx.Session.Movement.HasServerSpeed || ctx.Session.Movement.ServerSpeed != 112 {
		t.Fatalf("session speed = %+v, want authoritative 112", ctx.Session.Movement)
	}
	if world.Player.Speed != 112 {
		t.Fatalf("player speed = %d, want 112", world.Player.Speed)
	}
}

func TestLoginModeAppliesStartupCartDuringWorldFade(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Session: &session.Session{},
		World:   worldstate.New(),
	}

	if handled, err := applyLoginWorldPacket(ctx, testCartAmountPacket(1, 100, 450, 80000), &mode.console); !handled || err != nil {
		t.Fatal("cart amount packet was not handled")
	}
	if !ctx.Session.Cart.Open || ctx.Session.Cart.Amount != 1 || ctx.Session.Cart.MaxAmount != 100 || ctx.Session.Cart.Weight != 450 || ctx.Session.Cart.MaxWeight != 80000 {
		t.Fatalf("cart after startup packet = %+v", ctx.Session.Cart)
	}
}

func TestLoginModeAppliesSavedStatusDuringWorldFade(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}}

	if handled, err := applyLoginWorldPacket(ctx, testStatusEffectChangePacket(db.StatusAngelus, 2000000, true), &mode.console); !handled || err != nil {
		t.Fatal("Angelus status packet was not handled")
	}
	effect, ok := ctx.Session.Statuses.Active[db.StatusAngelus]
	if !ok || effect.Source != 2000000 {
		t.Fatalf("saved Angelus status = %+v, present=%t", effect, ok)
	}
}

func TestLoginMapChangeResetsWorldForChangedSelectedCharacter(t *testing.T) {
	change, ok, err := network.ParseMapChange(testMapChangePacket("geffen", 81, 179))
	if err != nil || !ok {
		t.Fatalf("parse map change ok=%t err=%v", ok, err)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150001,
			Selected:  session.Character{ID: 150001, Name: "Wizard", Job: db.JobWizard},
		},
		World: world,
	}

	applyLoginMapChange(ctx, change)

	if world.Player.ID != 150001 || world.Player.Job != db.JobWizard {
		t.Fatalf("world player = %+v, want selected wizard", world.Player)
	}
	if world.Player.HasCart || world.Player.Opt3State != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("old local state leaked after login map change: %+v", world.Player)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("stale actors survived login map change: %+v", world.Actors)
	}
}

func testParameterChangePacket(varID uint16, value uint32) network.Packet {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], 0x00B0)
	binary.LittleEndian.PutUint16(data[2:4], varID)
	binary.LittleEndian.PutUint32(data[4:8], value)
	return network.Packet{ID: 0x00B0, Data: data}
}

func testCartAmountPacket(amount, maxAmount uint16, weight, maxWeight uint32) network.Packet {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], 0x0121)
	binary.LittleEndian.PutUint16(data[2:4], amount)
	binary.LittleEndian.PutUint16(data[4:6], maxAmount)
	binary.LittleEndian.PutUint32(data[6:10], weight)
	binary.LittleEndian.PutUint32(data[10:14], maxWeight)
	return network.Packet{ID: 0x0121, Data: data}
}

func testStatusEffectChangePacket(statusID uint16, actorID uint32, active bool) network.Packet {
	data := make([]byte, 9)
	binary.LittleEndian.PutUint16(data[0:2], 0x0196)
	binary.LittleEndian.PutUint16(data[2:4], statusID)
	binary.LittleEndian.PutUint32(data[4:8], actorID)
	if active {
		data[8] = 1
	}
	return network.Packet{ID: 0x0196, Data: data}
}

func testMapChangePacket(mapName string, x, y int) network.Packet {
	data := make([]byte, 22)
	binary.LittleEndian.PutUint16(data[0:2], 0x0091)
	copy(data[2:18], []byte(mapName))
	binary.LittleEndian.PutUint16(data[18:20], uint16(x))
	binary.LittleEndian.PutUint16(data[20:22], uint16(y))
	return network.Packet{ID: 0x0091, Data: data}
}

func TestLoginToWorldPrewarmsBeforeFadeIn(t *testing.T) {
	ctx := client.Context{
		Resources: &res.Manager{Root: t.TempDir()},
		Session:   &session.Session{},
		World:     worldstate.New(),
		Network:   network.NewClient(20080910, false),
	}
	writeTestGAT(t, ctx.Resources.Root, "prontera.gat")
	next := NewLoginMode().nextWorldMode(ctx)
	next.Enter(ctx)
	if next.mapFade.Phase != maps.FadePrewarm {
		t.Fatalf("world fade phase after Enter = %d, want prewarm", next.mapFade.Phase)
	}
	if got := next.mapFade.Alpha(time.Now()); got != 255 {
		t.Fatalf("world prewarm alpha = %d, want 255", got)
	}
}

func TestLoginCursorUsesGogpuPointerAsROHand(t *testing.T) {
	mode := NewLoginMode()
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		UIApp:   fakeCursorUIApp{cursor: widget.CursorPointer},
		ScreenW: 1280,
		ScreenH: 720,
	}

	if got := mode.cursorAction(ctx); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click hand", got)
	}
}

type fakeCursorUIApp struct {
	cursor  widget.CursorType
	hovered widget.Widget
}

func (fakeCursorUIApp) SetUIRoot(widget.Widget)      {}
func (fakeCursorUIApp) Frame()                       {}
func (fakeCursorUIApp) Invalidate()                  {}
func (fakeCursorUIApp) InvalidateRect(geometry.Rect) {}
func (fakeCursorUIApp) RequestFullRepaint()          {}
func (fakeCursorUIApp) WidgetContext() widget.Context {
	return nil
}
func (a fakeCursorUIApp) Cursor() widget.CursorType {
	return a.cursor
}

func (a fakeCursorUIApp) HoveredWidget() widget.Widget {
	return a.hovered
}

func TestLoginActorBootstrapRestoresWeddingVisualState(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobWizard},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	data := make([]byte, 13)
	binary.LittleEndian.PutUint16(data[0:2], 0x0119)
	binary.LittleEndian.PutUint32(data[2:6], sessionState.AccountID)
	binary.LittleEndian.PutUint16(data[10:12], uint16(db.EffectStateWedding))

	if handled, err := applyLoginWorldPacket(ctx, network.Packet{ID: 0x0119, Data: data}, &mode.console); !handled || err != nil {
		t.Fatal("wedding state packet was not handled during login")
	}
	if world.Player.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("player effect state = 0x%08X, want wedding", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding == 0 {
		t.Fatalf("selected option = 0x%08X, want wedding", sessionState.Selected.Option)
	}
	if got := localPlayerVisualJob(ctx); got != db.JobMarried {
		t.Fatalf("visual job = %d, want married job %d", got, db.JobMarried)
	}
}

func TestLoginActorBootstrapRestoresLocalLook(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard, Weapon: 1201},
	}
	ctx := client.Context{Session: sessionState, World: world}
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data[0:2], 0x01D7)
	binary.LittleEndian.PutUint32(data[2:6], sessionState.AccountID)
	data[6] = 2

	if handled, err := applyLoginWorldPacket(ctx, network.Packet{ID: 0x01D7, Data: data}, &mode.console); !handled || err != nil {
		t.Fatal("look packet was not handled during login")
	}
	if sessionState.Selected.Weapon != 0 || world.Player.Weapon != 0 {
		t.Fatalf("restored look = selected weapon %d, player weapon %d", sessionState.Selected.Weapon, world.Player.Weapon)
	}
}

func TestCharacterSwitchKeepsConsolePublishableAfterLoginClear(t *testing.T) {
	manager := gameui.NewManager()
	ctx := client.Context{
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
		Session:   &session.Session{},
	}
	mode := NewWorldMode()
	mode.ui.console.AddSystemMessage("ready")
	mode.ui.console.Update(ctx)
	// Probe the console's own bounds: the fork docks it bottom-center, so a
	// fixed bottom-left corner is outside the overlay.
	px, py := consoleProbePoint(ctx)
	if !manager.PointerBlocked(px, py) {
		t.Fatal("console did not publish before character switch")
	}

	login := mode.nextCharacterSelectMode(ctx)
	login.Finish(ctx)
	if manager.PointerBlocked(px, py) {
		t.Fatal("login clear left the world console overlay published")
	}

	next := login.nextWorldMode(ctx)
	// The fork's compact-auto console drops its window when idle and reopens
	// on the next message — that reopen is what the handoff must preserve.
	next.ui.console.AddSystemMessage("back")
	next.ui.console.Update(ctx)
	if !manager.PointerBlocked(px, py) {
		t.Fatal("console did not republish after returning from character select")
	}
}

func TestLoginPacketFlowHandsBootstrapStateToWorld(t *testing.T) {
	netClient, server := newBotTestConnection(t, 20080910)
	ctx := client.Context{
		Network: netClient, Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World: worldstate.New(), Resources: &res.Manager{ClientInfo: res.ClientInfo{Connections: []res.Connection{{Address: "127.0.0.1"}}}},
	}
	ctx.Config.Headless = true
	mode := NewLoginMode()
	mode.console.AddSystemMessage("before login")
	mode.Enter(ctx)
	enter := make([]byte, 11)
	binary.LittleEndian.PutUint16(enter, 0x0073)
	stream := append(enter, testParameterChangePacket(network.StatusSpeed, 112).Data...)
	stream = append(stream, testMapChangePacket("geffen", 81, 179).Data...)
	if _, err := server.Write(stream); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for ctx.World.MapName != "geffen" && time.Now().Before(deadline) {
		if next, err := mode.Update(ctx); err != nil || next != nil {
			t.Fatalf("premature handoff: next=%T err=%v", next, err)
		}
		time.Sleep(time.Millisecond)
	}
	if !ctx.Session.Playing || ctx.World.MapName != "geffen" || ctx.Session.PlayerX != 81 || ctx.Session.PlayerY != 179 || ctx.World.Player.Speed != 112 {
		t.Fatalf("login bootstrap was not applied: session=%+v player=%+v", ctx.Session.Movement, ctx.World.Player)
	}
	// Even after the fade has elapsed, an opaque frame must be presented first.
	time.Sleep(550 * time.Millisecond)
	if next, err := mode.Update(ctx); err != nil || next != nil {
		t.Fatalf("handoff before presentation: next=%T err=%v", next, err)
	}
	mode.FrameSubmitted()
	next, err := mode.Update(ctx)
	world, ok := next.(*WorldMode)
	if err != nil || !ok {
		t.Fatalf("handoff: next=%T err=%v", next, err)
	}
	if messages := world.ui.console.Messages(); len(messages) != 1 || messages[0].Text != "before login" {
		t.Fatalf("console history lost: %+v", messages)
	}
}

// consoleProbePoint returns the center of the console's own overlay bounds.
func consoleProbePoint(ctx client.Context) (int, int) {
	x, y, w, h := gameui.ConsoleBoundsForTest(ctx.ScreenW, ctx.ScreenH)
	return x + w/2, y + h/2
}
