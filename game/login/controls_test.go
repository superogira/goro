package login

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/bot"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func loginGamepadTestContext() client.Context {
	return client.Context{
		Input: input.NewState(), Session: session.New(),
		Resources: loginTestResources(res.Connection{Display: "Local"}, res.Connection{Display: "LAN"}),
		UIManager: &loginTestUIManager{}, ScreenW: 1280, ScreenH: 720,
	}
}

func TestWASDLoginMenusAndModalPriority(t *testing.T) {
	ctx := loginGamepadTestContext()
	ctx.Config.Script.Path = "builtin:wasd"
	mode := New(Bindings{})
	mode.updateAccountWindow(ctx)
	script, err := bot.New(ctx, ctx.ScriptPath(), bot.Bindings{ControlUI: mode.ControlUI})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(script.Close)
	press := func(button input.GamepadButton) {
		t.Helper()
		ctx.Input.SetGamepad(input.GamepadFrame{ID: "test"})
		script.HandleGamepad(ctx, 0.02)
		ctx.Input.EndFrame()
		pad := input.GamepadFrame{ID: "test"}
		pad.Buttons[button] = true
		ctx.Input.SetGamepad(pad)
		capture := script.HandleGamepad(ctx, 0.02)
		if script.Disabled {
			t.Fatal("login script failed")
		}
		if !capture.Buttons[input.GamepadSouth] || !capture.Buttons[input.GamepadEast] {
			t.Fatal("menu buttons leaked into pointer fallback")
		}
		ctx.Input.EndFrame()
	}
	press(input.GamepadDown)
	if mode.serviceWindow.SelectedIndex() != 1 {
		t.Fatal("D-pad did not select LAN")
	}
	press(input.GamepadSouth)
	if mode.selectedLoginServer != 1 || mode.loginWindow == nil || mode.serviceWindow != nil {
		t.Fatal("South did not open credentials for LAN")
	}
	press(input.GamepadEast)
	if mode.serviceWindow == nil || mode.loginWindow != nil {
		t.Fatal("East did not return to servers")
	}
	press(input.GamepadEast)
	if !mode.quitConfirm.IsOpen() {
		t.Fatal("East did not open exit confirmation")
	}
	selected := mode.serviceWindow.SelectedIndex()
	press(input.GamepadDown)
	if mode.serviceWindow.SelectedIndex() != selected {
		t.Fatal("modal navigation changed server behind it")
	}
	press(input.GamepadEast)
	if mode.quitConfirm.IsOpen() || mode.serviceWindow == nil {
		t.Fatal("cancel leaked through exit dialog")
	}
	mode.hideServiceWindow(ctx)
	mode.phase = loginPhaseCharacter
	mode.selectedSlot = 0
	press(input.GamepadRight)
	if mode.selectedSlot != 1 {
		t.Fatal("D-pad did not navigate character slots")
	}
	press(input.GamepadSouth)
	if mode.fade.target != loginPhaseCreate || mode.create.slot != 1 {
		t.Fatal("South did not activate empty character slot")
	}
	press(input.GamepadRight)
	if mode.selectedSlot != 1 {
		t.Fatal("input changed character selection during fade")
	}
}

func TestWASDLoginRetainsPointerControls(t *testing.T) {
	ctx := loginGamepadTestContext()
	ctx.Config.Script.Path = "builtin:wasd"
	mode := New(Bindings{})
	mode.updateAccountWindow(ctx)
	script, err := bot.New(ctx, ctx.ScriptPath(), bot.Bindings{ControlUI: mode.ControlUI})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(script.Close)
	pad := input.GamepadFrame{ID: "test"}
	pad.Axes[input.GamepadRightX] = 1
	ctx.Input.SetGamepad(pad)
	script.HandleGamepad(ctx, 0.02)
	ctx.Input.EndFrame()
	pad.Axes[input.GamepadRightX] = 0
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture := script.HandleGamepad(ctx, 0.02)
	if capture.Pointer || capture.Buttons[input.GamepadSouth] || mode.serviceWindow == nil {
		t.Fatal("moving the pointer did not restore normal UI clicks")
	}
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadSouth] = false
	pad.Buttons[input.GamepadDown] = true
	ctx.Input.SetGamepad(pad)
	capture = script.HandleGamepad(ctx, 0.02)
	if !capture.Buttons[input.GamepadSouth] || mode.serviceWindow.SelectedIndex() != 1 {
		t.Fatal("D-pad did not restore menu navigation")
	}
	mode.hideServiceWindow(ctx)
	mode.phase = loginPhaseCreate
	ctx.Input.EndFrame()
	pad.Buttons[input.GamepadDown] = false
	pad.Buttons[input.GamepadSouth] = true
	ctx.Input.SetGamepad(pad)
	capture = script.HandleGamepad(ctx, 0.02)
	if capture.Buttons[input.GamepadSouth] || capture.Pointer || script.Disabled {
		t.Fatal("character creation lost its pointer controls")
	}
}
