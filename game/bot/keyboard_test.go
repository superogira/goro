package bot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

func TestLuaKeyboardExposesTextAndNonConsumingEdges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keyboard.lua")
	if err := os.WriteFile(path, []byte(`
function input()
	available = goro.keyboard.available()
	text = goro.keyboard.text()
	down = goro.keyboard.is_down("Enter")
	entered = goro.keyboard.was_pressed("Enter")
	released = goro.keyboard.was_released("Enter")
	unknown = goro.keyboard.is_down("NotAKey")
end
`), 0o644); err != nil {
		t.Fatal(err)
	}

	inputState := input.NewState()
	enter, ok := input.KeyCodeFromName("Enter")
	if !ok {
		t.Fatal("Enter was not recognized")
	}
	inputState.SetKeyCode(enter, true)
	inputState.AddTextInput("hé")
	bot, err := New(client.Context{Input: inputState}, path, Bindings{})
	if err != nil {
		t.Fatal(err)
	}
	defer bot.Close()

	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if text, _ := bot.state.GetGlobal("text").(lua.LString); string(text) != "hé" {
		t.Fatalf("keyboard text = %q, want %q", text, "hé")
	}
	if available, _ := bot.state.GetGlobal("available").(lua.LBool); !bool(available) {
		t.Fatal("keyboard was unavailable without UI capture")
	}
	if down, _ := bot.state.GetGlobal("down").(lua.LBool); !bool(down) {
		t.Fatal("held Enter key was not exposed")
	}
	if entered, _ := bot.state.GetGlobal("entered").(lua.LBool); !bool(entered) {
		t.Fatal("Enter press was not exposed")
	}
	if unknown, _ := bot.state.GetGlobal("unknown").(lua.LBool); bool(unknown) {
		t.Fatal("unknown key code was reported as down")
	}

	if err := bot.InputFrame(false); err != nil {
		t.Fatal(err)
	}
	if available, _ := bot.state.GetGlobal("available").(lua.LBool); bool(available) {
		t.Fatal("keyboard remained available during UI capture")
	}
	if text, _ := bot.state.GetGlobal("text").(lua.LString); string(text) != "" {
		t.Fatalf("captured keyboard text = %q, want empty", text)
	}
	if down, _ := bot.state.GetGlobal("down").(lua.LBool); bool(down) {
		t.Fatal("held key leaked through UI capture")
	}
	if entered, _ := bot.state.GetGlobal("entered").(lua.LBool); bool(entered) {
		t.Fatal("key press leaked through UI capture")
	}

	inputState.EndFrame()
	inputState.SetKeyCode(enter, false)
	if err := bot.InputFrame(true); err != nil {
		t.Fatal(err)
	}
	if released, _ := bot.state.GetGlobal("released").(lua.LBool); !bool(released) {
		t.Fatal("Enter release was not exposed")
	}
}
