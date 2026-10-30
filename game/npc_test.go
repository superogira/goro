package game

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestHiddenWarpNPCIsNotVisibleOrSelectable(t *testing.T) {
	for _, name := range []string{"", "boom1#airplane"} {
		for _, hasObjectType := range []bool{false, true} {
			t.Run(fmt.Sprintf("name=%q/object-type=%t", name, hasObjectType), func(t *testing.T) {
				actor := worldstate.Actor{ID: 300, Job: 139, Name: name, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeNPC, HasObjectType: hasObjectType}
				ctx, projection := cursorHoverTestContext(actor)
				ctx.Resources = &res.Manager{}
				mode := &WorldMode{}
				now := time.Now()
				for _, entry := range mode.actors().CollectSceneActorEntries(render.NewFrame(800, 600), ctx, projection) {
					if entry.Actor.ID == actor.ID {
						t.Error("hidden warp NPC entered the actor draw list")
					}
				}
				if actors.ActorCastsShadow(actor) {
					t.Error("hidden warp NPC casts a shadow")
				}
				if got := actorDisplayName(ctx, actor, false); got != "" {
					t.Errorf("display name = %q, want empty", got)
				}
				if labels := mode.actors().HoveredActorDisplayLabels(ctx, actor, now); len(labels) != 0 {
					t.Errorf("hover labels = %v, want none", labels)
				}
				mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
				if got, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, nil); ok {
					t.Errorf("hovered hidden NPC: %+v", got)
				}
				if _, ok := clickedTalkTarget(ctx, projection, mouseX, mouseY, now, nil); ok || npc.CanTalk(actor) {
					t.Error("hidden warp NPC can be talked to")
				}
				ctx.Session.NoShift = true
				if actors.ActorCanBeSkillTargeted(ctx, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, actor) || actors.ActorCanBeAttackClicked(ctx, actor) {
					t.Error("hidden warp NPC can be targeted")
				}
				if mode.scriptHighlightActor(ctx, actor.ID) {
					t.Error("hidden warp NPC can be highlighted by a script")
				}
			})
		}
	}
}

func TestHiddenWarpNPCRetainsAirshipExplosionAnchor(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{AccountID: 100, CharID: 200}}
	entry := network.ActorEntry{ID: 300, Job: 139, X: 239, Y: 62, ObjectType: scene.ActorObjectTypeNPC, HasObjectType: true, Appearance: true}
	mode := &WorldMode{}
	mode.upsertNetworkActor(ctx, entry)
	actor, ok := ctx.World.Actors[entry.ID]
	if !ok || actor.Job != entry.Job || actor.X != entry.X || actor.Y != entry.Y {
		t.Fatalf("hidden effect anchor was not retained: %+v, exists=%t", actor, ok)
	}
	mode.actors().ApplyWarpPortalEntry(ctx, entry)
	if len(mode.effects().Active) != 0 {
		t.Fatal("hidden warp NPC generated a visible warp portal")
	}
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: entry.ID, EffectID: effects.SuiExplosion})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want one explosion", len(mode.effects().Active))
	}
	effect := mode.effects().Active[0]
	if effect.EffectID != effects.SuiExplosion || effect.ActorID != entry.ID || effect.X != entry.X || effect.Y != entry.Y {
		t.Fatalf("explosion lost its hidden NPC anchor: %+v", effect)
	}
}

func TestHiddenNPCRemainsInteractive(t *testing.T) {
	// Unlike class 139, the original client gives class 111 a pick rectangle.
	actor := worldstate.Actor{ID: 300, Job: 111, X: 11, Y: 20, ObjectType: scene.ActorObjectTypeNPC, HasObjectType: true}
	ctx, projection := cursorHoverTestContext(actor)
	got, ok := clickedTalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, time.Now(), nil)
	if !ok || got.ID != actor.ID {
		t.Fatalf("interactive hidden NPC was not selectable: %+v, ok=%t", got, ok)
	}
}

func TestHiddenWarpNPCDoesNotLoadPoringRealData(t *testing.T) {
	manager := testutil.RealDataManager(t)
	if view, status := sprites.LoadNonPC(manager, 139, "hidden warp NPC"); view != nil {
		t.Fatalf("hidden warp NPC loaded a sprite: %s", status)
	}
	if view, status := sprites.LoadNonPC(manager, 1002, "Poring"); view == nil {
		t.Fatalf("normal Poring failed to load: %s", status)
	}
}

func TestMapAndCharacterTransitionsClearNPCCutins(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	mode.startMapFadeOut(network.MapChange{MapName: "geffen"}, testTime())
	if mode.ui.npc.Cutin.Visible() {
		t.Fatal("map transition left cut-in visible")
	}

	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	mode.startCharacterSelectFadeOut(testTime())
	if mode.ui.npc.Cutin.Visible() {
		t.Fatal("character transition left cut-in visible")
	}
}

func TestNPCCutinBlocksMapPointerOnlyWhileUIIsVisible(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinRight}, render.NewImage(20, 30))
	inputState := input.NewState()
	inputState.MouseX = 95
	inputState.MouseY = 75
	ctx := client.Context{Input: inputState, ScreenW: 100, ScreenH: 80}
	if !mode.mapPointerBlocked(ctx) {
		t.Fatal("visible cut-in did not block map pointer")
	}

	ctx.Config = config.Config{Render: config.RenderConfig{NoUI: true}}
	if mode.mapPointerBlocked(ctx) {
		t.Fatal("hidden cut-in blocked map pointer with no-UI rendering")
	}
}

func TestNPCDialogCloseCallbackClearsCutinBetweenWorldUpdates(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Hello"})
	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		UIManager: &worldModeTestUIManager{},
		ScreenW:   800,
		ScreenH:   600,
	}
	if !mode.ui.npc.Dialog.Update(ctx) {
		t.Fatal("dialog did not publish")
	}
	// gogpu/ui button callbacks are dispatched after the coarse Window.Update
	// hit test. Simulate Cancel closing the dialog between world updates.
	mode.ui.npc.Dialog.Reset()
	if mode.ui.npc.Dialog.IsOpen() {
		t.Fatal("dialog remained open")
	}
	if mode.ui.npc.Cutin.Visible() {
		t.Fatal("dialog close left cut-in visible")
	}
}

func TestDisconnectPacketClearsNPCCutins(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	packet := network.Packet{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0x00, 15}}
	mode.handleNetworkPacket(ctx, packet, testTime())
	if mode.ui.npc.Cutin.Visible() {
		t.Fatal("disconnect packet left cut-in visible")
	}
}

func TestDialogClearOnlyClearsCutinForActiveDialog(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Hello"})
	mode.ui.npc.Cutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	ctx := client.Context{ScreenW: 800, ScreenH: 600}

	mode.handleNetworkPacket(ctx, npcDialogClearPacket(20), testTime())
	if !mode.ui.npc.Cutin.Visible() {
		t.Fatal("clear for another NPC removed the active dialog cut-in")
	}

	mode.handleNetworkPacket(ctx, npcDialogClearPacket(10), testTime())
	if mode.ui.npc.Cutin.Visible() {
		t.Fatal("clear for the active NPC left its cut-in visible")
	}
}

func npcDialogClearPacket(npcID uint32) network.Packet {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x08D6)
	binary.LittleEndian.PutUint32(data[2:6], npcID)
	return network.Packet{ID: 0x08D6, Data: data}
}

func testTime() time.Time {
	return time.Unix(1, 0)
}

func npcSpriteChangeTestPacket(id, job uint32, packetType byte) network.Packet {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data, 0x01B0)
	binary.LittleEndian.PutUint32(data[2:], id)
	data[6] = packetType
	binary.LittleEndian.PutUint32(data[7:], job)
	return network.Packet{ID: 0x01B0, Data: data}
}

func TestNPCSpriteChangePreservesActorAndMovement(t *testing.T) {
	now := time.Now()
	before := worldstate.Actor{
		ID: 300, Job: 1008, Name: "Pupa", Appearance: true, X: 15, Y: 20, Dir: 6,
		ObjectType: scene.ActorObjectTypeMob, HasObjectType: true, Speed: 200,
		Moving: true, FromX: 10, FromY: 20, ToX: 15, ToY: 20,
		MoveStarted: now.Add(-time.Second), MoveDuration: 2 * time.Second,
		MoveStartX: 10.5, MoveStartY: 20.5, HasMoveStart: true,
		MovePath: []worldstate.WalkStep{{X: 10, Y: 20}, {X: 15, Y: 20}},
	}
	for _, login := range []bool{false, true} {
		for _, packetType := range []byte{0, 1, 2} {
			ctx := client.Context{World: worldstate.New(), Session: &session.Session{}}
			ctx.World.UpsertActor(before)
			want := ctx.World.Actors[before.ID]
			want.Job = 1018
			packet := npcSpriteChangeTestPacket(before.ID, 1018, packetType)
			if login {
				if handled, err := applyLoginActorBootstrapPacket(ctx, packet); !handled || err != nil {
					t.Fatal("login dropped transformation")
				}
			} else {
				(&WorldMode{}).handleNetworkPacket(ctx, packet, now)
			}
			if got := ctx.World.Actors[before.ID]; !reflect.DeepEqual(got, want) {
				t.Fatalf("login=%t type=%d: sprite change modified movement or failed: %+v", login, packetType, got)
			}
			// Later movement-only entries must retain the transformed class.
			actors.UpsertNetworkActor(ctx, network.ActorEntry{ID: before.ID, Moving: true, FromX: 15, FromY: 20, ToX: 16, ToY: 20, X: 16, Y: 20})
			if got := ctx.World.Actors[before.ID]; got.Job != 1018 || !got.Moving {
				t.Fatalf("move lost transformation: %+v", got)
			}
		}
	}
}

func TestNPCSpriteChangeIgnoresMissingActorsAndPlayers(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{AccountID: 100, CharID: 200, Selected: session.Character{ID: 200, Job: 1}}}
	ctx.World.Player = worldstate.Actor{ID: 200, Job: 1, Appearance: true}
	ctx.World.UpsertActor(worldstate.Actor{ID: 300, Job: 1, Appearance: true, ObjectType: scene.ActorObjectTypePC, HasObjectType: true})
	ctx.World.UpsertActor(worldstate.Actor{ID: 301, Job: 1, Appearance: true})
	mode := &WorldMode{}
	for _, id := range []uint32{0, 100, 200, 300, 301, 999} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(id, 1018, 0), time.Now())
	}
	if len(ctx.World.Actors) != 2 || ctx.World.Player.Job != 1 || ctx.Session.Selected.Job != 1 || ctx.World.Actors[300].Job != 1 || ctx.World.Actors[301].Job != 1 {
		t.Fatal("NPC sprite change created an actor or changed a player")
	}
	// A class cannot wrap around the signed 16-bit job stored in the world.
	ctx.World.UpsertActor(worldstate.Actor{ID: 400, Job: 1008, Appearance: true})
	for _, job := range []uint32{0, 1, 32768, 0x10000 + 1018} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(400, job, 0), time.Now())
		if ctx.World.Actors[400].Job != 1008 {
			t.Fatalf("accepted invalid NPC class %d", job)
		}
	}
}

func TestNPCSpriteChangeHandlesHiddenClasses(t *testing.T) {
	ctx := client.Context{World: worldstate.New()}
	ctx.World.UpsertActor(worldstate.Actor{ID: 300, Job: 1002, Appearance: true, ObjectType: scene.ActorObjectTypeNPC, HasObjectType: true})
	mode := &WorldMode{}
	for _, job := range []uint32{139, 111, 1002} {
		mode.handleNetworkPacket(ctx, npcSpriteChangeTestPacket(300, job, 1), time.Now())
		actor := ctx.World.Actors[300]
		if uint32(actor.Job) != job || sprites.JobHasNoSprite(int(actor.Job)) != (job != 1002) {
			t.Fatalf("class %d: %+v", job, actor)
		}
	}
}

func TestPupaTransformationsSwitchSpritesRealData(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Session: &session.Session{}, Resources: testutil.RealDataManager(t)}
	mode := &WorldMode{}
	now := time.Now()
	// Keep one unchanged Pupa to catch accidental mutation of the shared view.
	for id := uint32(300); id <= 364; id++ {
		ctx.World.UpsertActor(worldstate.Actor{ID: id, Job: 1008, Appearance: true, X: 10, Y: 20, Speed: 200})
	}
	pupa := mode.actors().NonPCSpriteView(ctx, ctx.World.Actors[300])
	if pupa == nil || !strings.Contains(strings.ToLower(pupa.Source), "pupa.spr") {
		t.Fatal("Pupa resource did not load")
	}
	var stream []byte
	for id := uint32(301); id <= 364; id++ {
		stream = append(stream, npcSpriteChangeTestPacket(id, 1018, byte(id%2)).Data...)
		name := make([]byte, 30)
		binary.LittleEndian.PutUint16(name, 0x0095)
		binary.LittleEndian.PutUint32(name[2:], id)
		copy(name[6:], "Creamy")
		stream = append(stream, name...)
	}
	framer := network.NewFramer(network.PacketLengths2008())
	for len(stream) > 0 {
		n := min(7, len(stream))
		packets, err := framer.Push(stream[:n])
		stream = stream[n:]
		if err != nil {
			t.Fatal(err)
		}
		for _, packet := range packets {
			mode.handleNetworkPacket(ctx, packet, now)
		}
	}
	for id := uint32(301); id <= 364; id++ {
		actors.UpsertNetworkActor(ctx, network.ActorEntry{ID: id, Moving: true, FromX: 10, FromY: 20, ToX: 15, ToY: 20, X: 15, Y: 20})
		actor := ctx.World.Actors[id]
		view := mode.actors().NonPCSpriteView(ctx, actor)
		if actor.Job != 1018 || actor.Name != "Creamy" || view == nil || view == pupa || !strings.Contains(strings.ToLower(view.Source), "creamy.spr") {
			t.Fatalf("actor %d did not become Creamy: %+v", id, actor)
		}
		state := mode.actors().NonPCSpriteState(actor, time.Now())
		if state.ActionFamily != scene.SpriteActionWalk || !state.Moving {
			t.Fatalf("Creamy lost movement: %+v", state)
		}
	}
	if mode.actors().NonPCSpriteView(ctx, ctx.World.Actors[300]) != pupa {
		t.Fatal("transformation changed other Pupas' shared sprite")
	}
	t.Log("64 Pupas switched to Creamy sprites; unchanged Pupa retained its sprite")
}

func TestServerProgressCancellationSendsAcknowledgement(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	now := time.Unix(100, 0)
	mode := NewWorldMode()
	ctx := client.Context{Network: networkClient}
	mode.startServerProgress(ctx, network.ProgressBar{Duration: time.Minute}, now)

	packet := network.Packet{
		ID:   network.PacketZCProgressCancel,
		Data: []byte{0xF2, 0x02},
	}
	if next, stop := mode.handleNetworkPacket(ctx, packet, now.Add(time.Second)); next != nil || stop {
		t.Fatalf("progress cancellation changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.npcSystem.Progress.Started.IsZero() {
		t.Fatal("server cancellation did not clear progress")
	}
	readBotTestPackets(t, serverConn, network.BuildProgressBarDonePacket())
}

func TestStoragePasswordPromptGatesStorageUntilSuccess(t *testing.T) {
	sessionState := &session.Session{
		Storage: session.Storage{Open: true},
	}
	ctx := client.Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	promptData := make([]byte, 4)
	binary.LittleEndian.PutUint16(promptData[0:2], network.PacketZCStoragePasswordPrompt)
	binary.LittleEndian.PutUint16(promptData[2:4], uint16(network.StoragePasswordSet))

	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCStoragePasswordPrompt, Data: promptData}, time.Now()); next != nil || stop {
		t.Fatalf("password prompt changed mode: next=%T stop=%t", next, stop)
	}
	if !sessionState.Storage.PasswordPending || sessionState.Storage.Open {
		t.Fatalf("storage state after prompt = %+v", sessionState.Storage)
	}
	if !mode.ui.npc.StoragePassword.IsOpen() {
		t.Fatal("password prompt did not open")
	}

	applyStorageAmount(ctx, network.StorageAmount{Amount: 2, MaxAmount: 300})
	applyStorageItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Amount: 4}})
	if sessionState.Storage.Open {
		t.Fatal("storage opened while password authentication was pending")
	}
	if sessionState.Storage.Amount != 2 || len(sessionState.Storage.Items) != 1 {
		t.Fatalf("storage data was not retained behind password gate: %+v", sessionState.Storage)
	}

	resultData := make([]byte, 6)
	binary.LittleEndian.PutUint16(resultData[0:2], network.PacketZCStoragePasswordResult)
	binary.LittleEndian.PutUint16(resultData[2:4], uint16(network.StoragePasswordCheckSucceeded))
	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCStoragePasswordResult, Data: resultData}, time.Now()); next != nil || stop {
		t.Fatalf("password result changed mode: next=%T stop=%t", next, stop)
	}
	if sessionState.Storage.PasswordPending || !sessionState.Storage.Open {
		t.Fatalf("storage state after success = %+v", sessionState.Storage)
	}
	if mode.ui.npc.StoragePassword.IsOpen() {
		t.Fatal("password prompt remained open after success")
	}
	if !mode.ui.storageWindow.IsOpen() {
		t.Fatal("storage window did not open after authentication")
	}
}
