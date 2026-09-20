package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type loginWindowTestApp struct{ basicMenuTestApp }

func (a loginWindowTestApp) WidgetContext() widget.Context { return a.app.Window().Context() }

func TestLoginWindowInitialFocusAndTabNavigation(t *testing.T) {
	app := uiapp.New()
	bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
	window := NewLoginWindow(ctx, "", "", false, LoginWindowCallbacks{})
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if app.Window().Context().FocusedWidget() != window.user {
		t.Error("Account's initial focus was not registered with the UI context")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUnknown, 'a', event.ModNone))
	if window.Username != "a" {
		t.Fatal("Account did not accept typing before the first Tab")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
	if !window.password.IsFocused() || window.user.IsFocused() || app.Window().Context().FocusedWidget() != window.password {
		t.Fatal("first Tab did not move from Account to Password")
	}
	// Publishing/updating an unchanged form must not reset the chosen field.
	window.SetContext(ctx)
	window.Publish(ctx)
	if app.Window().Context().FocusedWidget() != window.password {
		t.Fatal("updating the login window stole focus from Password")
	}
	// Resizing rebuilds the fields: the context must follow the new instance.
	oldPassword := window.password
	ctx.ScreenW = 1024
	window.SetContext(ctx)
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if window.password == oldPassword || app.Window().Context().FocusedWidget() != window.password || oldPassword.IsFocused() {
		t.Fatal("resizing did not transfer focus to the rebuilt Password field")
	}
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModShift))
	if !window.user.IsFocused() || window.password.IsFocused() || app.Window().Context().FocusedWidget() != window.user {
		t.Fatal("Shift+Tab did not return to Account after resizing")
	}
}

func TestLoginWindowToggleKeep(t *testing.T) {
	// The handheld's R2 shortcut drives the checkbox through ToggleKeep;
	// OnToggle keeps KeepID in sync so the login flow saves the right ID.
	app := uiapp.New()
	bridge := loginWindowTestApp{basicMenuTestApp{app: app}}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := client.Context{ScreenW: 800, ScreenH: 600, UIApp: bridge, UIManager: manager}
	window := NewLoginWindow(ctx, "", "", false, LoginWindowCallbacks{})
	window.Publish(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})

	window.ToggleKeep()
	if !window.KeepID {
		t.Fatal("first toggle did not turn KeepID on")
	}
	window.ToggleKeep()
	if window.KeepID {
		t.Fatal("second toggle did not turn KeepID off")
	}
}

func TestLoginWindowLabelsFillRightAlignedColumn(t *testing.T) {
	width, height := loginWindowSize()
	window := &LoginWindow{layout: loginWindowLayout{W: width, H: height}}
	tree := window.widgetTree()
	tree.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(float32(width), float32(height))))

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("login window content tree is incomplete")
	}
	rows := windowChildren[1].Children()[0].Children()
	// The fork adds the Keep-ID checkbox inline on the Account row, to the
	// right of the field (upstream has only Account and Password rows).
	if len(rows) != 2 {
		t.Fatalf("login form rows = %d, want Account and Password", len(rows))
	}

	wantRowChildren := []int{3, 2} // Account: label, field, Keep; Password: label, field
	for i, want := range []string{"Account", "Password"} {
		rowChildren := rows[i].Children()
		if len(rowChildren) != wantRowChildren[i] || len(rowChildren[0].Children()) != 1 {
			t.Fatalf("%s row has %d children, want %d starting with a label and field", want, len(rowChildren), wantRowChildren[i])
		}
		labelSlot := rowChildren[0]
		label, ok := labelSlot.Children()[0].(*primitives.TextWidget)
		if !ok {
			t.Fatalf("%s label = %T, want text", want, labelSlot.Children()[0])
		}
		if label.Content() != want {
			t.Fatalf("login label = %q, want %q", label.Content(), want)
		}
		if label.Style().Align != widget.TextAlignRight {
			t.Fatalf("%s alignment = %v, want right", label.Style().Align, want)
		}

		slotBounds := labelSlot.(interface{ Bounds() geometry.Rect }).Bounds()
		labelBounds := label.Bounds()
		if labelBounds.Min.X != 0 || labelBounds.Width() != slotBounds.Width() {
			t.Fatalf("%s label bounds = %v, want full %.1fpx column width", want, labelBounds, slotBounds.Width())
		}
	}

	// The Keep checkbox rides on the Account row, after the text field —
	// and the whole row must stay inside the window's padded content area
	// (the row once overflowed the right edge and the window clipped the
	// checkbox out entirely).
	keepSlot := rows[0].Children()[2]
	keepChildren := keepSlot.Children()
	if len(keepChildren) != 1 {
		t.Fatalf("Keep slot has %d children, want the checkbox", len(keepChildren))
	}
	if _, ok := keepChildren[0].(*checkbox.Widget); !ok {
		t.Fatalf("Keep slot holds %T, want a checkbox", keepChildren[0])
	}
	keepBounds := keepSlot.(interface{ Bounds() geometry.Rect }).Bounds()
	if keepBounds.Max.X > float32(width)-loginWindowFieldRightPad {
		t.Fatalf("Keep checkbox right edge %.1f overflows the window's padded content (max %.1f)",
			keepBounds.Max.X, float32(width)-loginWindowFieldRightPad)
	}
	if !window.keep.SkipTabTraversal() {
		t.Fatal("Keep checkbox must opt out of tab traversal so Tab advances Account -> Password")
	}
}
