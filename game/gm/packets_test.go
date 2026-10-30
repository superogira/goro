package gm

import (
	"strings"
	"testing"

	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func TestGMReplyPresentation(t *testing.T) {
	for _, test := range []struct {
		id   uint16
		data []byte
		want string
	}{
		{0x00cd, []byte{0xcd, 0, 0}, "Unable to disconnect"},
		{0x00cd, []byte{0xcd, 0, 42}, "Target disconnected"},
		{0x014a, []byte{0x4a, 1, 3, 0, 0, 0}, "GM has blocked your chat"},
		{0x014a, []byte{0x4a, 1, 4, 0, 0, 0}, "anti-spam"},
		{0x014a, []byte{0x4a, 1, 5, 0, 0, 0}, ""},
		{0x0214, append([]byte{0x14, 2, 99, 11}, make([]byte, 38)...), "STR 99"},
	} {
		console := &gameui.ChatConsole{}
		if !HandlePacket(network.Packet{ID: test.id, Data: test.data}, console) {
			t.Fatalf("unhandled packet %x", test.id)
		}
		var lines []string
		for _, m := range console.Messages() {
			lines = append(lines, m.Text)
		}
		if text := strings.Join(lines, "\n"); (test.want == "" && text != "") || !strings.Contains(text, test.want) {
			t.Fatalf("packet %x messages=%q, want %q", test.id, text, test.want)
		}
	}
}
