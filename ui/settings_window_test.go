package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestSettingsTabsFitSmallWindowAndKeepSelectionAfterToggle(t *testing.T) {
	t.Skip("fork: the handheld settings window is the gamepad-row layout, not upstream's section tabs")
}

func TestSettingsWindowEscapeCloses(t *testing.T) {
	var window SettingsWindow
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume escape")
	}
	if window.IsOpen() {
		t.Fatal("settings window stayed open after escape")
	}
}

func TestSettingsWindowCloseButtonCloses(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, UIManager: manager, ScreenW: 800, ScreenH: 600}
	var window SettingsWindow
	window.OpenWindow(ctx)

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	x, y, _, _ := centeredWindowRect(ctx, settingsWindowW, settingsWindowH)
	buttonX := float32(x + settingsWindowW - 16)
	buttonY := float32(y + ROWindowTitleHeight/2)
	inputState.SetMousePosition(int(buttonX), int(buttonY))
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !window.Update(ctx) {
		t.Fatal("settings window did not consume close-button press")
	}
	if window.dragging || window.dragLayer {
		t.Fatal("close-button press started a window drag")
	}

	app.Window().HandleEvent(uitest.Click(buttonX, buttonY))
	app.Window().HandleEvent(uitest.Release(buttonX, buttonY))

	if window.IsOpen() {
		t.Fatal("settings window stayed open after close button")
	}
}
