package ui

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestCharacterHUDOpensAndCloses(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{Selected: session.Character{Name: "Kivutar"}},
		ScreenW: 800,
		ScreenH: 600,
	}
	var w CharacterWindow
	if w.IsOpen() {
		t.Fatal("fresh HUD must start closed")
	}
	w.Update(ctx)
	if !w.IsOpen() {
		t.Fatal("HUD must auto-open with a session")
	}
	// Click the close button.
	closeX := w.x + w.width - characterHUDCloseSize/2 - 3
	closeY := w.y + characterHUDCloseSize/2
	inputState.SetMousePosition(closeX, closeY)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !w.Update(ctx) {
		t.Fatal("close press was not consumed")
	}
	if w.IsOpen() {
		t.Fatal("close button must hide the HUD")
	}
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	inputState.EndFrame()
	// Stays dismissed across updates until the next login.
	w.Update(ctx)
	if w.IsOpen() {
		t.Fatal("dismissed HUD must not reopen on its own")
	}
}

func TestCharacterHUDDragClampsToScreen(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{Selected: session.Character{Name: "Kivutar"}},
		ScreenW: 800,
		ScreenH: 300,
	}
	var w CharacterWindow
	w.Update(ctx)
	inputState.SetMousePosition(w.x+10, w.y+5)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !w.Update(ctx) {
		t.Fatal("drag start was not consumed")
	}
	if !w.dragLayer {
		t.Fatal("HUD did not enter drag state")
	}
	inputState.EndFrame()
	inputState.SetMousePosition(700, 1000)
	w.Update(ctx)
	if w.x > ctx.ScreenW-w.width {
		t.Fatalf("drag x = %d beyond right edge", w.x)
	}
	_, menuHeight := basicMenuSize()
	maxY := ctx.ScreenH - w.height - basicMenuFollowGap - menuHeight
	if w.y != maxY {
		t.Fatalf("drag y = %d, want clamped at %d (screen minus menu)", w.y, maxY)
	}
	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	w.Update(ctx)
	if w.dragLayer {
		t.Fatal("drag state did not end on release")
	}
}

func TestCharacterWindowDataFallsBackToSelectedCharacter(t *testing.T) {
	character, vitals, progress, inventory := characterWindowData(&session.Session{
		Selected: session.Character{ID: 7, Name: "Kivutar", HP: 12, MaxHP: 34, SP: 5, MaxSP: 6, Level: 7, JobLevel: 4},
	})
	if character.Name != "Kivutar" {
		t.Fatalf("character name = %q", character.Name)
	}
	if vitals.HP != 12 || vitals.MaxHP != 34 || vitals.SP != 5 || vitals.MaxSP != 6 {
		t.Fatalf("vitals fallback = %+v", vitals)
	}
	if progress.BaseLevel != 7 || progress.JobLevel != 4 {
		t.Fatalf("progress fallback = %+v", progress)
	}
	if inventory.Weight != 0 || inventory.Zeny != 0 {
		t.Fatalf("inventory fallback = %+v", inventory)
	}
}

func TestFormatHUDNumberGroupsThousands(t *testing.T) {
	cases := map[int64]string{
		0:        "0",
		999:      "999",
		1000:     "1,000",
		114527:   "114,527",
		-1234567: "-1,234,567",
	}
	for value, want := range cases {
		if got := formatHUDNumber(value); got != want {
			t.Fatalf("formatHUDNumber(%d) = %q, want %q", value, got, want)
		}
	}
}
