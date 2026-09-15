package ui

import (
	"testing"

	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func battleModeConsole(t *testing.T) (*ChatConsole, Context, *uiapp.App) {
	t.Helper()
	ctx := shortcutWindowTestContext(t)
	ctx.Session = session.New()
	app := uiapp.New()
	ctx.UIApp = instanceTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(ctx.UIApp)
	ctx.UIManager = manager
	console := &ChatConsole{}
	console.Publish(ctx)
	app.Frame()
	return console, ctx, app
}

// Match native event ordering: physical input, preparation, then UI text.
func typeConsoleText(console *ChatConsole, ctx Context, app *uiapp.App, code input.KeyCode, text string) {
	ctx.Input.SetKeyCode(code, true)
	if !console.PrepareTextInput(ctx, code) {
		for _, r := range text {
			app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, event.ModNone))
		}
	}
	ctx.Input.SetKeyCode(code, false)
}

func TestClassicConsoleTypesImmediately(t *testing.T) {
	console, ctx, app := battleModeConsole(t)
	if ctx.Session.BattleMode {
		t.Fatal("Battle Mode must default to off")
	}
	typeConsoleText(console, ctx, app, gpucontext.KeyE, "éclair")
	if console.currentInput() != "éclair" || !console.Active() {
		t.Fatalf("direct typing: text=%q active=%v", console.currentInput(), console.Active())
	}
	if app.Window().Context().FocusedWidget() != console.inputField {
		t.Fatal("direct typing did not update the UI focus manager")
	}
	console.inputField.SetText("/ns")
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
	console.UpdateInput(ctx)
	if !ctx.Session.NoShift || console.currentInput() != "" {
		t.Fatal("directly typed command was not submitted")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
	typeConsoleText(console, ctx, app, gpucontext.KeyH, "hello")
	if console.currentInput() != "hello" {
		t.Fatal("typing after submission required Enter")
	}
	console.inputField.SetText("")
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
	console.UpdateInput(ctx)
	if !console.Active() {
		t.Fatal("empty Enter left classic chat")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
	ctx.Input.SetMousePosition(700, 300)
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
	if console.UpdateInput(ctx) || console.Active() {
		t.Fatal("classic chat swallowed a click on the map")
	}
}

func TestClassicConsoleEditsImmediatelyAfterMapClick(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   input.KeyCode
		key    event.Key
		setup  []event.Key
		want   string
		cursor int
	}{
		{"Delete", gpucontext.KeyDelete, event.KeyDelete, []event.Key{event.KeyLeft, event.KeyLeft}, "abd", 2},
		{"Backspace", gpucontext.KeyBackspace, event.KeyBackspace, nil, "abc", 3},
		{"Left", gpucontext.KeyLeft, event.KeyLeft, nil, "abcd", 3},
		{"Right", gpucontext.KeyRight, event.KeyRight, []event.Key{event.KeyHome}, "abcd", 1},
		{"Home", gpucontext.KeyHome, event.KeyHome, nil, "abcd", 0},
		{"End", gpucontext.KeyEnd, event.KeyEnd, []event.Key{event.KeyHome}, "abcd", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			console, ctx, app := battleModeConsole(t)
			typeConsoleText(console, ctx, app, gpucontext.KeyA, "abcd")
			for _, key := range tc.setup {
				app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, key, 0, event.ModNone))
			}
			ctx.Input.SetMousePosition(700, 300)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			if console.UpdateInput(ctx) || console.Active() {
				t.Fatal("map click did not release chat without swallowing movement")
			}
			ctx.Input.EndFrame()
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			// Key preparation precedes both UI dispatch and input-state updates.
			console.PrepareKeyInput(ctx, tc.code, 0)
			app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, tc.key, 0, event.ModNone))
			ctx.Input.SetKeyCode(tc.code, true)
			console.UpdateInput(ctx)
			if !console.Active() || console.currentInput() != tc.want || console.inputField.CursorPosition() != tc.cursor {
				t.Fatalf("first edit: text=%q cursor=%d active=%v", console.currentInput(), console.inputField.CursorPosition(), console.Active())
			}
			if app.Window().Context().FocusedWidget() != console.inputField {
				t.Fatal("editing did not restore UI focus")
			}
		})
	}
}

func TestConsoleEnterAfterMapClick(t *testing.T) {
	for _, battle := range []bool{false, true} {
		name := "normal"
		if battle {
			name = "battle"
		}
		t.Run(name, func(t *testing.T) {
			console, ctx, app := battleModeConsole(t)
			typeConsoleText(console, ctx, app, gpucontext.KeyN, "/ns")
			ctx.Session.BattleMode = battle
			ctx.Input.EndFrame()
			ctx.Input.SetMousePosition(700, 300)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			if console.UpdateInput(ctx) || console.Active() {
				t.Fatal("map click did not release chat")
			}
			ctx.Input.EndFrame()
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			console.PrepareKeyInput(ctx, gpucontext.KeyEnter, 0)
			app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
			ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
			console.UpdateInput(ctx)
			if battle {
				if ctx.Session.NoShift || console.currentInput() != "/ns" || !console.Active() {
					t.Fatal("Battle Mode must open chat without sending the draft")
				}
			} else if !ctx.Session.NoShift || console.currentInput() != "" {
				t.Fatal("normal mode did not submit the draft on the first Enter")
			}
			ctx.Input.EndFrame()
			console.UpdateInput(ctx)
			if ctx.Session.NoShift != !battle {
				t.Fatal("held Enter submitted the draft again")
			}
		})
	}
}

func TestClassicConsoleRestoresSelectionEditing(t *testing.T) {
	console, ctx, app := battleModeConsole(t)
	typeConsoleText(console, ctx, app, gpucontext.KeyA, "abcd")
	console.setActive(false)
	console.PrepareKeyInput(ctx, gpucontext.KeyA, gpucontext.ModControl)
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyA, 0, event.ModCtrl))
	start, end := console.inputField.Selection()
	if start != 0 || end != 4 {
		t.Fatal("Ctrl+A did not select the existing draft")
	}
	console.setActive(false)
	console.PrepareKeyInput(ctx, gpucontext.KeyDelete, 0)
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyDelete, 0, event.ModNone))
	if console.currentInput() != "" {
		t.Fatal("Delete did not preserve and remove the selection")
	}
}

func TestConsoleKeyPreparationRespectsBattleModeAndModifiers(t *testing.T) {
	console, ctx, _ := battleModeConsole(t)
	ctx.Session.BattleMode = true
	console.PrepareKeyInput(ctx, gpucontext.KeyDelete, 0)
	if console.Active() {
		t.Fatal("Delete opened chat in Battle Mode")
	}
	ctx.Session.BattleMode = false
	for _, mods := range []gpucontext.Modifiers{gpucontext.ModAlt, gpucontext.ModAlt | gpucontext.ModControl, gpucontext.ModSuper} {
		console.PrepareKeyInput(ctx, gpucontext.KeyDelete, mods)
		if console.Active() {
			t.Fatal("modified Delete opened chat")
		}
	}
	for _, code := range []input.KeyCode{gpucontext.KeyEscape, gpucontext.KeyF1, gpucontext.KeyTab, gpucontext.KeyA} {
		console.PrepareKeyInput(ctx, code, 0)
		if console.Active() {
			t.Fatalf("non-editing key %v opened chat", code)
		}
	}
}

func TestBattleModeSpaceTemporarilyEnablesChat(t *testing.T) {
	console, ctx, app := battleModeConsole(t)
	console.SendText(ctx, "/bm")
	if !ctx.Session.BattleMode || console.Active() {
		t.Fatal("/bm did not enable Battle Mode")
	}
	typeConsoleText(console, ctx, app, gpucontext.KeyQ, "a") // AZERTY
	if console.currentInput() != "" || console.Active() {
		t.Fatal("Battle Mode typed a skill key into chat")
	}
	typeConsoleText(console, ctx, app, gpucontext.KeySpace, " ")
	if console.currentInput() != "" || !console.Active() {
		t.Fatal("Space did not open chat without inserting itself")
	}
	typeConsoleText(console, ctx, app, gpucontext.KeyN, "/ns")
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
	console.UpdateInput(ctx)
	if !ctx.Session.NoShift || console.Active() || console.currentInput() != "" {
		t.Fatal("submission did not return to Battle Mode")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKeyCode(gpucontext.KeyEnter, false)
	console.SendText(ctx, "/battlemode")
	if ctx.Session.BattleMode {
		t.Fatal("/battlemode did not turn Battle Mode off")
	}
	typeConsoleText(console, ctx, app, gpucontext.KeyQ, "a")
	if console.currentInput() != "a" {
		t.Fatal("disabling Battle Mode did not restore direct typing")
	}
}

func TestConsoleEnterAndTextInOneFrame(t *testing.T) {
	for _, battle := range []bool{false, true} {
		console, ctx, app := battleModeConsole(t)
		ctx.Session.BattleMode = battle
		console.PrepareKeyInput(ctx, gpucontext.KeyEnter, 0)
		app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
		ctx.Input.SetKeyCode(gpucontext.KeyEnter, true)
		typeConsoleText(console, ctx, app, gpucontext.KeyH, "hello")
		console.UpdateInput(ctx)
		if console.currentInput() != "hello" || !console.Active() || console.hasPendingSubmit {
			t.Fatalf("Battle Mode %v: the opening Enter lost or submitted the first text", battle)
		}
	}
}

func TestBattleModeDoesNotChangeShortcutDraft(t *testing.T) {
	console := &ChatConsole{input: "draft", active: true}
	ctx := Context{Session: session.New()}
	console.SendText(ctx, "/bm")
	if console.currentInput() != "draft" || !ctx.Session.BattleMode || console.Active() {
		t.Fatal("/bm shortcut did not preserve the draft while changing modes")
	}
}

func TestConsoleBattleModeAfterRebind(t *testing.T) {
	original, ctx, app := battleModeConsole(t)
	ctx.Session.BattleMode = true
	next := *original
	next.Rebind(ctx)
	app.Frame()
	typeConsoleText(&next, ctx, app, gpucontext.KeySpace, " ")
	typeConsoleText(&next, ctx, app, gpucontext.KeyB, "/bm")
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	next.UpdateInput(ctx)
	if ctx.Session.BattleMode || next.currentInput() != "" || original.hasPendingSubmit {
		t.Fatal("chat callbacks still target the previous world after a map change")
	}
}

func TestConsoleRebindPreservesEditingState(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup []*event.KeyEvent
		want  string
	}{
		{"caret", []*event.KeyEvent{
			event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModNone),
		}, "hé🙂lao"},
		{"backward selection", []*event.KeyEvent{
			event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModShift),
			event.NewKeyEvent(event.KeyPress, event.KeyLeft, 0, event.ModShift),
		}, "hé🙂a"},
		{"forward selection", []*event.KeyEvent{
			event.NewKeyEvent(event.KeyPress, event.KeyHome, 0, event.ModNone),
			event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModNone),
			event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModShift),
			event.NewKeyEvent(event.KeyPress, event.KeyRight, 0, event.ModShift),
		}, "halo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, active := range []bool{false, true} {
				original, ctx, app := battleModeConsole(t)
				typeConsoleText(original, ctx, app, gpucontext.KeyH, "hé🙂lo")
				for _, key := range tc.setup {
					app.Window().HandleEvent(key)
				}
				original.setActive(active)
				start, end := original.inputField.Selection()
				cursor := original.inputField.CursorPosition()
				next := *original
				next.Rebind(ctx)
				app.Frame()
				nextStart, nextEnd := next.inputField.Selection()
				if nextStart != start || nextEnd != end || next.inputField.CursorPosition() != cursor || next.Active() != active {
					t.Fatalf("map change lost editing state: selection %d:%d -> %d:%d, cursor %d -> %d, active %v -> %v",
						start, end, nextStart, nextEnd, cursor, next.inputField.CursorPosition(), active, next.Active())
				}
				typeConsoleText(&next, ctx, app, gpucontext.KeyA, "a")
				if next.currentInput() != tc.want || next.input != tc.want || original.input != "hé🙂lo" {
					t.Fatalf("continued editing: field=%q new console=%q old console=%q", next.currentInput(), next.input, original.input)
				}
			}
		})
	}
}

func TestShortcutBarDropsFollowDisplayedRows(t *testing.T) {
	for activeRow := range shortcutMaxRows {
		for visibleRows := 1; visibleRows <= shortcutMaxRows; visibleRows++ {
			ctx := shortcutBarActionContext(input.NewState())
			bar := &ShortcutBar{activeRow: activeRow, visibleRows: visibleRows}
			for row := range shortcutMaxRows {
				slot := row * shortcutCols
				x, y := bar.slotBounds(ctx, slot)
				got, ok := bar.slotAt(ctx, x+1, y+1)
				visible := (row-activeRow+shortcutMaxRows)%shortcutMaxRows < visibleRows
				if ok != visible || ok && got != slot {
					t.Fatalf("active=%d visible=%d slot=%d hit=(%d,%v)", activeRow, visibleRows, slot, got, ok)
				}
				if visible {
					bar.AcceptSkillDrop(ctx, session.Skill{ID: 6, Level: 2, Type: 1}, x+1, y+1)
					if bar.slots[slot].skillID != 6 || ctx.Session.Hotkeys.Slots[slot].ID != 6 {
						t.Fatal("drop changed a different server hotkey slot")
					}
				}
			}
		}
	}
}

func TestBattleModePhysicalKeysAndChatIsolation(t *testing.T) {
	for slot, key := range shortcutBattleKeys {
		t.Run(key.String(), func(t *testing.T) {
			ctx := shortcutBarActionContext(input.NewState())
			actions := &skillWindowTestRenderer{}
			bar := &ShortcutBar{}
			bar.slots[slot] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
			ctx.Input.SetKeyCode(key, true)
			if bar.UpdateKeyboardInput(ctx, actions, false) || actions.used.ID != 0 {
				t.Fatal("letter key activated a skill outside Battle Mode")
			}
			ctx.Session.BattleMode = true
			if bar.UpdateKeyboardInput(ctx, actions, true) || actions.used.ID != 0 {
				t.Fatal("letter key activated a skill during chat")
			}
			if !bar.UpdateKeyboardInput(ctx, actions, false) || actions.used.ID != 6 {
				t.Fatalf("physical key %v did not activate slot %d", key, slot)
			}
			if bar.UpdateKeyboardInput(ctx, actions, false) {
				t.Fatal("held letter retriggered")
			}
		})
	}
}

func TestFunctionKeysFollowSelectedRowDuringChat(t *testing.T) {
	for row := range shortcutMaxRows {
		ctx := shortcutBarActionContext(input.NewState())
		actions := &skillWindowTestRenderer{}
		bar := &ShortcutBar{activeRow: row}
		bar.slots[row*shortcutCols] = shortcutSlotState{kind: shortcutSkill, skillID: 6, skillLevel: 2}
		ctx.Input.SetKey(input.KeyF1, true)
		if !bar.UpdateKeyboardInput(ctx, actions, true) || actions.used.ID != 6 {
			t.Fatalf("F1 did not activate row %d during chat", row+1)
		}
	}
}

func TestBattleModeLeavesAltGrAndNumberKeysAlone(t *testing.T) {
	ctx := shortcutBarActionContext(input.NewState())
	ctx.Session.BattleMode = true
	bar := &ShortcutBar{}
	for _, modifier := range []input.KeyCode{gpucontext.KeyLeftAlt, gpucontext.KeyRightAlt, gpucontext.KeyLeftControl, gpucontext.KeyLeftSuper} {
		ctx.Input.ResetKeyboard()
		ctx.Input.SetKeyCode(modifier, true)
		ctx.Input.SetKeyCode(gpucontext.KeyQ, true)
		if bar.UpdateKeyboardInput(ctx, nil, false) {
			t.Fatalf("modified shortcut was consumed: %v", modifier)
		}
	}
	ctx.Input.ResetKeyboard()
	ctx.Input.SetKey(input.Key1, true)
	if bar.UpdateKeyboardInput(ctx, nil, false) {
		t.Fatal("classic Battle Mode should not bind the number row")
	}
}
