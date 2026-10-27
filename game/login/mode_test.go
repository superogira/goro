package login

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestNewCharacterSelectPreparesSavedCharacters(t *testing.T) {
	ctx := client.Context{Session: &session.Session{
		Characters: []session.Character{
			{ID: 150002, Slot: 4, Name: "Second"},
			{ID: 150001, Slot: 2, Name: "First"},
		},
	}}

	mode := NewCharacterSelect(ctx, Bindings{})
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

func TestCharacterSelectFadeStartsAfterEnter(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}, Resources: &res.Manager{}}
	ctx.Config.Headless = true
	mode := NewCharacterSelect(ctx, Bindings{})
	if mode.fade.phase != loginFadeIn || !mode.fade.started.IsZero() {
		t.Fatalf("pending fade = %+v", mode.fade)
	}
	mode.Enter(ctx)
	if mode.fade.started.IsZero() {
		t.Fatal("fade-in did not start after Enter")
	}
	if got := mode.fadeAlpha(mode.fade.started); got != 255 {
		t.Fatalf("fade alpha = %d, want 255", got)
	}
}

func TestMapErrorRequiresExplicitRetry(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}, Resources: &res.Manager{}, ScreenW: 800, ScreenH: 600}
	mode := New(Bindings{})
	mode.ReportMapError("missing.gat")
	mode.Enter(ctx)
	if !mode.disconnectDialog.IsOpen() || mode.fade.phase != loginFadeNone || !mode.autoAttempted || !mode.autoCharAttempted {
		t.Fatal("map error did not stop automatic login with a visible alert")
	}
	mode.disconnectDialog.Confirm(ctx)
	if mode.loginWindow == nil || mode.loginPending {
		t.Fatal("acknowledging error did not leave login ready for an explicit retry")
	}
}
