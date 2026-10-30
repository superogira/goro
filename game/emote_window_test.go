package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestAltLTogglesEmoteWindow(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.KeyAlt, true)
	inputState.SetKey(input.KeyL, true)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		ScreenW: 800,
		ScreenH: 600,
	}

	if !mode.toggleEmoteWindowFromInput(ctx) {
		t.Fatal("Alt+L was not consumed")
	}
	if !mode.ui.emoteWindow.IsOpen() {
		t.Fatal("emote window did not open")
	}
}

func TestAltGDoesNotOpenGuildWindowWithoutGuild(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.KeyAlt, true)
	inputState.SetKey(input.KeyG, true)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{},
		ScreenW: 800,
		ScreenH: 600,
	}

	if !mode.toggleGuildWindowFromInput(ctx) {
		t.Fatal("Alt+G was not consumed")
	}
	if mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window opened for a guildless character")
	}
}

func TestAltGTogglesGuildWindow(t *testing.T) {
	inputState := input.NewState()
	inputState.SetKey(input.KeyAlt, true)
	inputState.SetKey(input.KeyG, true)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{GuildID: 1},
		ScreenW: 800,
		ScreenH: 600,
	}

	if !mode.toggleGuildWindowFromInput(ctx) {
		t.Fatal("Alt+G was not consumed")
	}
	if !mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window did not open")
	}
}

func TestAltGTogglesGuildWindowDuringChatInput(t *testing.T) {
	mode := &WorldMode{}
	activateInput := input.NewState()
	activateInput.SetKey(input.KeyEnter, true)
	activateCtx := client.Context{
		Input:   activateInput,
		Session: &session.Session{GuildID: 1},
		ScreenW: 800,
		ScreenH: 600,
	}
	if !mode.ui.console.UpdateInput(activateCtx) || !mode.ui.console.Active() {
		t.Fatal("Enter did not activate text input")
	}

	inputState := input.NewState()
	inputState.SetKey(input.KeyAlt, true)
	inputState.SetKey(input.KeyG, true)
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{GuildID: 1},
		ScreenW: 800,
		ScreenH: 600,
	}

	if !mode.toggleGuildWindowFromInput(ctx) {
		t.Fatal("Alt+G was not consumed during chat input")
	}
	if !mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window did not open during chat input")
	}
}
