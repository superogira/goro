package game

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestNewCharacterSelectModePreparesSavedCharacters(t *testing.T) {
	ctx := client.Context{Session: &session.Session{
		Characters: []session.Character{
			{ID: 150002, Slot: 4, Name: "Second"},
			{ID: 150001, Slot: 2, Name: "First"},
		},
	}}

	mode := NewCharacterSelectMode(ctx, gameui.ChatConsole{})
	if mode.phase != loginPhaseCharacter {
		t.Fatalf("phase = %d, want character select", mode.phase)
	}
	if mode.selectedSlot != 2 {
		t.Fatalf("selected slot = %d, want first occupied slot 2", mode.selectedSlot)
	}
	if mode.maxSlots < 6 {
		t.Fatalf("max slots = %d, want enough slots for saved characters", mode.maxSlots)
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
	if !manager.PointerBlocked(400, 500) {
		t.Fatal("console did not publish before character switch")
	}

	login := mode.nextCharacterSelectMode(ctx)
	login.clearLoginWindows(ctx)
	if manager.PointerBlocked(400, 500) {
		t.Fatal("login clear left the world console overlay published")
	}

	next := login.nextWorldMode(ctx)
	next.ui.console.Update(ctx)
	if !manager.PointerBlocked(400, 500) {
		t.Fatal("console did not republish after returning from character select")
	}
}
