package rotheme

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestTextFieldInsetShadowMeetsTopAndSideBorders(t *testing.T) {
	bounds := geometry.NewRect(3, 5, 80, 22)
	canvas := &uitest.MockCanvas{}

	TextFieldPainter{}.PaintTextField(canvas, &textfield.PaintState{Bounds: bounds})

	const shadowRows = 4
	if len(canvas.Rects) != 1+shadowRows {
		t.Fatalf("textfield rectangles = %d, want background and %d shadow rows", len(canvas.Rects), shadowRows)
	}
	for row, call := range canvas.Rects[1:] {
		want := geometry.NewRect(bounds.Min.X, bounds.Min.Y+float32(row), bounds.Width(), 1)
		if call.Bounds != want {
			t.Fatalf("shadow row %d bounds = %v, want %v", row, call.Bounds, want)
		}
	}
	if len(canvas.StrokeRects) != 1 || canvas.StrokeRects[0].Bounds != bounds {
		t.Fatalf("textfield border strokes = %v, want one around %v", canvas.StrokeRects, bounds)
	}
}

func TestFormFocusTransferRepaintsBothFields(t *testing.T) {
	for _, navigation := range []string{"mouse", "tab", "programmatic", "release"} {
		t.Run(navigation, func(t *testing.T) {
			app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
			first := TextField("First", textfield.TypeText, nil, nil)
			second := TextField("Second", textfield.TypeText, nil, nil)
			app.SetRoot(primitives.Box(
				primitives.Box(first).Width(160).Height(22),
				primitives.Box(second).Width(160).Height(22),
			).Gap(30).PaddingXY(100, 100))
			app.Frame()
			draw := func() {
				t.Helper()
				if !app.Window().DrawTo(&uitest.MockCanvas{}) {
					t.Fatal("expected a repaint")
				}
			}
			mouse := func(kind event.MouseEventType, pos geometry.Point) {
				button := event.ButtonNone
				if kind == event.MousePress || kind == event.MouseRelease {
					button = event.ButtonLeft
				}
				app.HandleEvent(event.NewMouseEvent(kind, button, 0, pos, pos, event.ModNone))
			}
			draw()
			mouse(event.MouseMove, first.ScreenBounds().Center())
			mouse(event.MousePress, first.ScreenBounds().Center())
			mouse(event.MouseRelease, first.ScreenBounds().Center())
			draw()
			if !first.IsFocused() {
				t.Fatal("first field did not receive focus")
			}
			// Paint the old field's mouse-leave before transferring focus. Its
			// hover damage must not mask missing focus-loss invalidation.
			mouse(event.MouseMove, second.ScreenBounds().Center())
			draw()
			switch navigation {
			case "mouse":
				mouse(event.MousePress, second.ScreenBounds().Center())
				mouse(event.MouseRelease, second.ScreenBounds().Center())
			case "tab":
				app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyTab, 0, event.ModNone))
			case "programmatic":
				app.Window().Context().RequestFocus(second)
			case "release":
				app.Window().Context().ReleaseFocus(first)
			}
			if first.IsFocused() || second.IsFocused() != (navigation != "release") {
				t.Fatalf("unexpected focus: first=%t second=%t", first.IsFocused(), second.IsFocused())
			}
			draw()
			if app.Window().WasFullRepaint() {
				t.Fatal("focus change should not require a full repaint")
			}
			fields := []*textfield.Widget{first}
			if navigation != "release" {
				fields = append(fields, second)
			}
			for _, field := range fields {
				if clip := app.Window().LastDirtyUnion(); !clip.ContainsRect(field.ScreenBounds().Expand(0.5)) {
					t.Errorf("focus repaint clip %v misses field border %v", clip, field.ScreenBounds())
				}
			}
			if widget.NeedsRedrawInTree(app.Window().Root()) {
				t.Error("form remained dirty after repaint")
			}
		})
	}
}
