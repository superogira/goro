package client

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/kivutar/goro/session"
)

func TestGMCommandPacketsMatch2008Layouts(t *testing.T) {
	ctx := Context{Session: &session.Session{Selected: session.Character{Name: "GM"}}}
	name := "506c61796572204f6e65" + strings.Repeat("00", 14)
	for _, test := range []struct{ input, hex string }{
		{"/b hi", "99000c00474d203a20686900"},
		{"/broadcast hi", "99000c00474d203a20686900"},
		{"/nb Hello", "99000a0048656c6c6f00"},
		{"/lb hi", "9c010c00474d203a20686900"},
		{"/localbroadcast hi", "9c010c00474d203a20686900"},
		{"/nlb Hello", "9c010a0048656c6c6f00"},
		{"/mm prontera 100 200", "400170726f6e7465726100000000000000006400c800"},
		{"/mapmove prontera 100 200", "400170726f6e7465726100000000000000006400c800"},
		{"/shift Player One", "bb01" + name},
		{"/summon \"Player One\"", "bd01" + name},
		{"/rc Player One", "1202" + name},
		{" /CHECK Player One ", "1302" + name},
		{`/check ""Alice""`, "130222416c69636522" + strings.Repeat("00", 17)},
		{`/check ""Alice"`, "130222416c696365" + strings.Repeat("00", 18)},
		{"/item Poring", "3f01506f72696e67" + strings.Repeat("00", 18)},
		{"/monster Poring", "3f01506f72696e67" + strings.Repeat("00", 18)},
		{"/hide", "9d0140000000"},
		{"/kill 123456", "cc0040e20100"},
		{"/kick 123456", "cc0040e20100"},
		{"/killall", "ce00"},
		{"/resetstate", "97010000"},
		{"/resetskill", "97010100"},
		{"/mute 123456 60", "490140e20100013c00"},
		{"/unmute 123456", "490140e2010000ffff"},
		{"/cmt 10 20 1", "98010a0014000100"},
		{"/changemaptype 10 20 1", "98010a0014000100"},
	} {
		t.Run(test.input, func(t *testing.T) {
			command, args := splitChatCommand(test.input)
			packet, handled, err := buildGMCommand(ctx, command, args)
			if !handled || err != nil {
				t.Fatalf("handled=%t err=%v", handled, err)
			}
			if got := hex.EncodeToString(packet); got != test.hex {
				t.Fatalf("packet=%s, want %s", got, test.hex)
			}
		})
	}
}

func TestGMCommandsRejectAmbiguousOrOutOfRangeArguments(t *testing.T) {
	for _, input := range []string{
		"/b", "/nb " + strings.Repeat("x", 256), "/b hello\x00there", "/b hello\nthere",
		"/check", "/shift \"\"", "/summon \"unfinished", "/check " + strings.Repeat("x", 24),
		"/item foo\x00bar", "/check " + strings.Repeat("é", 12),
		"/mm prontera 1", "/mm prontera 1 2 3", "/mm prontera -1 2", "/mm prontera 65536 2",
		"/mm " + strings.Repeat("x", 16) + " 1 2", "/mm prontera 1.5 2",
		"/kill", "/kill 0", "/kill -1", "/kill 4294967296", "/kill 1 2",
		"/mute 1", "/mute 1 0", "/mute 1 32768", "/mute 1 65536", "/unmute 0", "/unmute 1 60",
		"/hide now", "/killall oops", "/resetstate 1", "/resetskill 0",
		"/cmt 1 2", "/cmt 1 2 7", "/cmt 65536 2 0",
	} {
		t.Run(input, func(t *testing.T) {
			command, args := splitChatCommand(input)
			packet, handled, err := buildGMCommand(Context{}, command, args)
			if !handled || err == nil || packet != nil {
				t.Fatalf("packet=%x handled=%t err=%v", packet, handled, err)
			}
		})
	}
	for _, command := range []string{"hello", "@item", "/fog", "/aura", "/remove", "/recall"} {
		if _, handled, _ := buildGMCommand(Context{}, command, ""); handled {
			t.Fatalf("%s unexpectedly handled as a GM command", command)
		}
	}
}
