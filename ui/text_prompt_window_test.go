package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/client"
)

func TestPasswordPromptPreservesPasswordAndResetsForText(t *testing.T) {
	var window TextPromptWindow
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	for _, password := range []string{"", " pass ", "12345678"} {
		window.OpenPassword(ctx, "Private room", "Password", 8)
		if window.inputType != textfield.TypePassword {
			t.Fatal("password is not masked")
		}
		window.value = password
		window.submit(ctx)
		if action := window.PopAction(); !action.Submitted || action.Text != password {
			t.Fatalf("password action = %+v", action)
		}
	}
	window.Open(ctx, "Message", "Message", "", 79)
	window.submit(ctx)
	if !window.IsOpen() || window.inputType != textfield.TypeText || window.PopAction().Submitted {
		t.Fatal("password mode leaked into a regular text prompt")
	}
}

func TestPasswordPromptCancelButtons(t *testing.T) {
	for _, index := range []int{0, 2} {
		ctx, _, app := newWindowInstanceTest()
		var window TextPromptWindow
		window.OpenPassword(ctx, "Private room", "Password", 8)
		app.Frame()
		app.Window().DrawTo(&uitest.MockCanvas{})
		buttons := collectEscapeMenuButtons(window.content)
		point := buttons[index].ScreenBounds().Center()
		app.Window().HandleEvent(uitest.Click(point.X, point.Y))
		app.Window().HandleEvent(uitest.Release(point.X, point.Y))
		if action := window.PopAction(); !action.Cancelled || action.Submitted || window.IsOpen() {
			t.Fatalf("button %d: action = %+v", index, action)
		}
	}
}

func TestTextPromptWindowSubmitPublishesAction(t *testing.T) {
	var window TextPromptWindow
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	window.Open(ctx, "Talkie Box Message", "Message", "Message", 79)
	window.value = " hello "
	window.submit(ctx)

	action := window.PopAction()
	if !action.Submitted || action.Text != "hello" {
		t.Fatalf("action = %+v", action)
	}
	if window.IsOpen() {
		t.Fatal("window remained open after submit")
	}
}

func TestTextPromptWindowKeyboardFocus(t *testing.T) {
	for _, kind := range []string{"generic", "party"} {
		t.Run(kind, func(t *testing.T) {
			app := uiapp.New()
			bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
			manager := NewManager()
			manager.SetUIApp(bridge)
			ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
			var prompt TextPromptWindow
			window := &prompt
			open := func() { prompt.Open(ctx, "Title", "Label", "Placeholder", 23) }
			if kind == "party" {
				var invite PartyInviteWindow
				window = &invite.TextPromptWindow
				open = func() { invite.Open(ctx) }
			}
			open()
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			wc := app.Window().Context()
			if wc.FocusedWidget() != window.inputField {
				t.Fatal("initial input focus was not registered")
			}
			app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'a', event.ModNone))
			if window.value != "a" {
				t.Fatal("input did not accept typing before Tab")
			}
			buttons := collectEscapeMenuButtons(window.content)
			app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
			if wc.FocusedWidget() != buttons[1] || window.inputField.IsFocused() {
				t.Fatal("first Tab did not move from input to OK")
			}
			window.Update(ctx)
			window.Publish(ctx)
			if wc.FocusedWidget() != buttons[1] {
				t.Fatal("idle update stole keyboard focus")
			}
			app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
			if wc.FocusedWidget() != window.inputField {
				t.Fatal("Shift+Tab did not return to the input")
			}
			oldInput := window.inputField
			next := *window
			next.Rebind(ctx)
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			if next.inputField == oldInput || oldInput.IsFocused() || wc.FocusedWidget() != next.inputField {
				t.Fatal("rebind did not transfer focus to the new input")
			}
			buttons = collectEscapeMenuButtons(next.content)
			app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
			if wc.FocusedWidget() != buttons[1] {
				t.Fatal("Tab did not advance from the rebound input")
			}
			next.Close()
			if wc.FocusedWidget() != nil {
				t.Fatal("closing retained focus in the old prompt")
			}
			next.Open(ctx, next.title, next.label, next.placeholder, next.maxLength)
			if wc.FocusedWidget() != next.inputField || next.value != "" {
				t.Fatal("reopening did not restore an empty focused input")
			}
		})
	}
}
