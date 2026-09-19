package game

import (
	"testing"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
)

func TestWindowShortcutsToggle(t *testing.T) {
	for _, tc := range []struct {
		name   string
		key    input.KeyCode
		ctrl   bool
		window func(*WorldMode) *gameui.Window
	}{
		{"inventory", gpucontext.KeyE, false, func(m *WorldMode) *gameui.Window { return &m.ui.inventoryBag.Window }},
		{"equipment", gpucontext.KeyQ, false, func(m *WorldMode) *gameui.Window { return &m.ui.equipmentWindow.Window }},
		{"skills", gpucontext.KeyS, false, func(m *WorldMode) *gameui.Window { return &m.ui.skillWindow.Window }},
		{"stats", gpucontext.KeyA, false, func(m *WorldMode) *gameui.Window { return &m.ui.statsWindow.Window }},
		{"party", gpucontext.KeyZ, false, func(m *WorldMode) *gameui.Window { return &m.ui.friendsWindow.Window }},
		{"friends", gpucontext.KeyH, false, func(m *WorldMode) *gameui.Window { return &m.ui.friendsWindow.Window }},
		{"cart", gpucontext.KeyW, false, func(m *WorldMode) *gameui.Window { return &m.ui.cartWindow.Window }},
		{"chat room", gpucontext.KeyC, false, func(m *WorldMode) *gameui.Window { return &m.ui.chatRoomCreate.Window }},
		{"pet", gpucontext.KeyJ, false, func(m *WorldMode) *gameui.Window { return &m.ui.petInfoWindow.Window }},
		{"homunculus", gpucontext.KeyR, false, func(m *WorldMode) *gameui.Window { return &m.ui.homunculusInfo.Window }},
		{"mercenary", gpucontext.KeyR, true, func(m *WorldMode) *gameui.Window { return &m.ui.mercenaryInfo.Window }},
		{"settings", gpucontext.KeyO, false, func(m *WorldMode) *gameui.Window { return &m.ui.settingsWindow.Window }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			ctx.Session.Cart.MaxAmount = 100
			ctx.Session.Homunculus.Active = true
			ctx.Session.Mercenary.Active = true
			m := &WorldMode{petID: 1, hasPetProperty: true}
			modifier := gpucontext.KeyLeftAlt
			if tc.ctrl {
				modifier = gpucontext.KeyLeftControl
			}
			ctx.Input.SetKeyCode(modifier, true)
			ctx.Input.SetKeyCode(tc.key, true)
			if !m.PrepareTextInput(ctx, tc.key) || !m.toggleWindowFromInput(ctx) || !tc.window(m).IsOpen() {
				t.Fatal("shortcut did not open its window or leaked text")
			}
			if ctx.Input.KeyCodeJustPressed(tc.key) || m.toggleWindowFromInput(ctx) {
				t.Fatal("held key retriggered the shortcut")
			}
			ctx.Input.SetKeyCode(tc.key, false)
			ctx.Input.SetKeyCode(tc.key, true)
			if !m.PrepareTextInput(ctx, tc.key) || !m.toggleWindowFromInput(ctx) || tc.window(m).IsOpen() {
				t.Fatal("shortcut did not close its window or leaked text")
			}
		})
	}
}

func TestWindowShortcutModifiersAndBlocking(t *testing.T) {
	for _, tc := range []struct {
		name              string
		modifier          input.KeyCode
		blocked, consumed bool
	}{
		{name: "AltGr", modifier: gpucontext.KeyRightAlt},
		{name: "control", modifier: gpucontext.KeyLeftControl},
		{name: "shift", modifier: gpucontext.KeyLeftShift},
		{name: "super", modifier: gpucontext.KeyLeftSuper},
		{name: "other form", blocked: true},
		{name: "Lua consumed key", consumed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			m := &WorldMode{}
			m.ui.settingsWindow.OpenWindow(ctx)
			ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
			ctx.Input.SetKeyCode(gpucontext.KeyO, true)
			if tc.modifier != gpucontext.KeyUnknown {
				ctx.Input.SetKeyCode(tc.modifier, true)
			}
			if tc.blocked {
				m.ui.chatRoomCreate.Open(ctx)
			}
			if tc.consumed {
				ctx.Input.ConsumeKeyCodePress(gpucontext.KeyO)
			} else if m.PrepareTextInput(ctx, gpucontext.KeyO) {
				t.Fatal("blocked shortcut swallowed text")
			}
			if m.toggleWindowFromInput(ctx) || !m.ui.settingsWindow.IsOpen() {
				t.Fatal("blocked shortcut toggled its window")
			}
		})
	}
	ctx := chatShortcutTestContext(t)
	m := &WorldMode{}
	ctx.Input.SetKeyCode(gpucontext.KeyE, true)
	if m.toggleWindowFromInput(ctx) || m.suppressShortcutText(ctx, gpucontext.KeyE) {
		t.Fatal("bare letter activated a window shortcut")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyLeftControl, true)
	if m.toggleWindowFromInput(ctx) || m.suppressShortcutText(ctx, gpucontext.KeyE) {
		t.Fatal("Ctrl+E acted as Alt+E")
	}
	ctx.Input.SetKeyCode(gpucontext.KeyR, true)
	m.ui.escapeMenu.Window.Open(ctx, nil)
	if m.toggleWindowFromInput(ctx) || m.suppressShortcutText(ctx, gpucontext.KeyR) {
		t.Fatal("Ctrl+R bypassed a modal")
	}
}

func TestWindowShortcutsWithoutCartOrCompanion(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := &WorldMode{}
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	for _, key := range []input.KeyCode{gpucontext.KeyW, gpucontext.KeyJ, gpucontext.KeyR} {
		ctx.Input.SetKeyCode(key, true)
		if !m.toggleWindowFromInput(ctx) {
			t.Fatal("unavailable window shortcut was not consumed")
		}
		ctx.Input.SetKeyCode(key, false)
	}
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, false)
	ctx.Input.SetKeyCode(gpucontext.KeyLeftControl, true)
	ctx.Input.SetKeyCode(gpucontext.KeyR, true)
	if !m.toggleWindowFromInput(ctx) {
		t.Fatal("unavailable mercenary shortcut was not consumed")
	}
	if m.ui.cartWindow.IsOpen() || ctx.Session.Cart.Open || m.ui.petInfoWindow.IsOpen() ||
		m.petInfoRequested || m.ui.homunculusInfo.IsOpen() || m.ui.mercenaryInfo.IsOpen() {
		t.Fatal("opened an unavailable cart or companion window")
	}
}

func TestBasicInfoShortcut(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := &WorldMode{}
	m.ui.characterWindow.Update(ctx)
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	ctx.Input.SetKeyCode(gpucontext.KeyV, true)
	if !m.PrepareTextInput(ctx, gpucontext.KeyV) || !m.toggleWindowFromInput(ctx) {
		t.Fatal("Alt+V did not toggle the basic-info window")
	}
}
