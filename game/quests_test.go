package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestQuestPacketsReachSessionAndActivationWaitsForAck(t *testing.T) {
	netClient, server := newBotTestConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Session: &session.Session{}}
	mode := &WorldMode{}
	send := func(id uint16, data []byte) {
		t.Helper()
		binary.LittleEndian.PutUint16(data, id)
		mode.handleNetworkPacket(ctx, network.Packet{ID: id, Data: data}, time.Now())
	}
	list := []byte{0, 0, 13, 0, 1, 0, 0, 0, 0xe9, 3, 0, 0, 1}
	send(network.PacketZCQuestList, list)
	mission := make([]byte, 112)
	binary.LittleEndian.PutUint16(mission[2:], 112)
	binary.LittleEndian.PutUint32(mission[4:], 1)
	binary.LittleEndian.PutUint32(mission[8:], 1001)
	binary.LittleEndian.PutUint16(mission[20:], 1)
	binary.LittleEndian.PutUint32(mission[22:], 1002)
	copy(mission[28:], "Poring")
	send(network.PacketZCQuestMissions, mission)
	hunt := []byte{0, 0, 18, 0, 1, 0, 0xe9, 3, 0, 0, 0xea, 3, 0, 0, 10, 0, 4, 0}
	send(network.PacketZCQuestHunt, hunt)
	quest := ctx.Session.Quests.Entries[1001]
	if !quest.Active || len(quest.Objectives) != 1 || quest.Objectives[0].Current != 4 || quest.Objectives[0].Required != 10 {
		t.Fatalf("quest after login and hunt = %+v", quest)
	}
	mode.setQuestActive(ctx, 1001, false)
	readBotTestPackets(t, server, []byte{0xb6, 2, 0xe9, 3, 0, 0, 0})
	if !ctx.Session.Quests.Entries[1001].Active {
		t.Fatal("quest changed before server ACK")
	}
	send(network.PacketZCQuestActive, []byte{0, 0, 0xe9, 3, 0, 0, 0})
	if ctx.Session.Quests.Entries[1001].Active {
		t.Fatal("activation ACK ignored")
	}
	// Malformed packets must leave the previous snapshot untouched.
	version := ctx.Session.Quests.Version
	send(network.PacketZCQuestList, []byte{0, 0, 8, 0, 255, 255, 255, 255})
	if ctx.Session.Quests.Version != version || len(ctx.Session.Quests.Entries) != 1 {
		t.Fatal("malformed list changed journal")
	}
	send(network.PacketZCQuestDelete, []byte{0, 0, 0xe9, 3, 0, 0})
	if len(ctx.Session.Quests.Entries) != 0 {
		t.Fatal("delete did not remove quest")
	}
}

func TestQuestShortcutAndMapTransition(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}, Input: input.NewState(), UIManager: &worldModeTestUIManager{}, ScreenW: 1024, ScreenH: 768}
	ctx.Session.Quests.Set(session.Quest{ID: 1001, Active: true})
	ctx.Input.SetKey(input.KeyAlt, true)
	ctx.Input.SetKeyCode(gpucontext.KeyU, true)
	mode := &WorldMode{}
	if !mode.PrepareTextInput(ctx, gpucontext.KeyU) {
		t.Fatal("Alt+U leaked a character to chat")
	}
	if !mode.toggleQuestWindowFromInput(ctx) || !mode.ui.questWindow.IsOpen() {
		t.Fatal("Alt+U did not open journal")
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyU) {
		t.Fatal("shortcut did not consume U")
	}
	next := mode.nextWorldMode()
	next.rebindPersistentUI(ctx)
	if !next.ui.questWindow.IsOpen() || len(ctx.Session.Quests.Entries) != 1 {
		t.Fatal("map transition lost journal")
	}
	ctx.Input = input.NewState()
	ctx.Input.SetKey(input.KeyAlt, true)
	ctx.Input.SetKeyCode(gpucontext.KeyU, true)
	if !next.toggleQuestWindowFromInput(ctx) || next.ui.questWindow.IsOpen() {
		t.Fatal("Alt+U did not close carried journal")
	}
	if !mode.ui.questWindow.IsOpen() {
		t.Fatal("carried window still changed the old mode")
	}
}

func TestQuestShortcutDoesNotStealAltGr(t *testing.T) {
	ctx := client.Context{Input: input.NewState()}
	ctx.Input.SetKeyCode(gpucontext.KeyRightAlt, true)
	ctx.Input.SetKeyCode(gpucontext.KeyU, true)
	mode := &WorldMode{}
	if mode.toggleQuestWindowFromInput(ctx) || mode.suppressShortcutText(ctx, gpucontext.KeyU) {
		t.Fatal("journal stole AltGr typing")
	}
}
