package game

import (
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
)

func TestClassicChatRespectsFormsAndTransitions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*WorldMode, client.Context)
	}{
		{"settings", func(m *WorldMode, ctx client.Context) { m.ui.settingsWindow.OpenWindow(ctx) }},
		{"shortcut editor", func(m *WorldMode, ctx client.Context) { m.ui.chatShortcuts.Toggle(ctx, &m.ui.console, nil) }},
		{"progress", func(m *WorldMode, _ client.Context) { m.serverProgress.started = time.Now() }},
		{"map transition", func(m *WorldMode, _ client.Context) { m.mapFade.phase = mapFadeHold }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := chatShortcutTestContext(t)
			m := NewWorldMode()
			tc.setup(m, ctx)
			if m.PrepareTextInput(ctx, gpucontext.KeyA) || m.ui.console.Active() {
				t.Fatal("default chat stole text or focus from a form/transition")
			}
			m.PrepareKeyInput(ctx, gpucontext.KeyDelete, 0)
			if m.ui.console.Active() {
				t.Fatal("editing key stole chat focus from a form/transition")
			}
		})
	}
}

func TestClassicEditingPreparationUsesCurrentMode(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := NewWorldMode()
	manager := &Manager{mode: NewLoginMode()}
	manager.PrepareKeyInput(ctx, gpucontext.KeyDelete, 0)
	if m.ui.console.Active() {
		t.Fatal("login routed an editing key to world chat")
	}
	manager.mode = m
	manager.PrepareKeyInput(ctx, gpucontext.KeyDelete, 0)
	if !m.ui.console.Active() {
		t.Fatal("world did not restore chat focus for editing")
	}
}

func TestClassicChatPreparationUsesCurrentMode(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	m := NewWorldMode()
	manager := &Manager{mode: m}
	if manager.PrepareTextInput(ctx, gpucontext.KeyA) || !m.ui.console.Active() {
		t.Fatal("world mode did not route the first character to chat")
	}
	manager.mode = NewLoginMode()
	ctx.Input.SetKeyCode(gpucontext.KeyLeftAlt, true)
	if manager.PrepareTextInput(ctx, gpucontext.KeyM) {
		t.Fatal("login intercepted an Alt/Option character")
	}
}

func TestBattleModeSurvivesMapChangeAndResetsAtLogin(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Session.BattleMode = true
	m := NewWorldMode().nextWorldMode()
	m.rebindPersistentUI(ctx)
	if !ctx.Session.BattleMode {
		t.Fatal("map change reset Battle Mode")
	}
	ctx.Config.Headless = true
	ctx.Config.Login.AutoLogin = false
	ctx.Input = input.NewState()
	NewLoginMode().Enter(ctx)
	if ctx.Session.BattleMode {
		t.Fatal("returning to login did not reset Battle Mode")
	}
}
