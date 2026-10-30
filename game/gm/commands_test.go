package gm_test

import (
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestGMCommandsFromConsoleSendNativePackets(t *testing.T) {
	networkClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: networkClient, Session: session.New()}
	ctx.Session.Selected.Name = "GM"
	console := &gameui.ChatConsole{}
	for _, test := range []struct {
		command string
		opcode  uint16
		size    int
	}{
		{"/b hello", 0x0099, 15}, {"/lb hello", 0x019c, 15},
		{"/kill 2000000", 0x00cc, 6}, {"/killall", 0x00ce, 2},
		{"/item Poring", 0x013f, 26}, {"/mm prontera 100 100", 0x0140, 22},
		{"/mute 2000000 60", 0x0149, 9}, {"/resetskill", 0x0197, 4},
		{"/cmt 100 100 1", 0x0198, 8}, {"/hide", 0x019d, 6},
		{"/shift Alice", 0x01bb, 26}, {"/summon Alice", 0x01bd, 26},
		{"/rc Alice", 0x0212, 26}, {"/check Alice", 0x0213, 26},
	} {
		if !console.SendText(ctx, test.command) {
			t.Fatalf("command failed: %s: %+v", test.command, console.Messages())
		}
		server.SetReadDeadline(time.Now().Add(time.Second))
		packet := make([]byte, test.size)
		if _, err := io.ReadFull(server, packet); err != nil {
			t.Fatalf("%s: %v", test.command, err)
		}
		if got := binary.LittleEndian.Uint16(packet); got != test.opcode {
			t.Fatalf("%s: opcode=%x want=%x", test.command, got, test.opcode)
		}
	}
	for _, command := range []string{"/killall oops", "/kill 0", "/check", "/mute 1 -60"} {
		if console.SendText(ctx, command) {
			t.Fatalf("invalid command accepted: %s", command)
		}
	}
	if !console.SendText(ctx, "/nb marker") {
		t.Fatal("marker failed")
	}
	testutil.ReadPackets(t, server, []byte{0x99, 0, 11, 0, 'm', 'a', 'r', 'k', 'e', 'r', 0})
}
