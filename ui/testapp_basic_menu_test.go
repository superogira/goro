package ui

import (
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

type basicMenuTestApp struct {
	app *uiapp.App
}

func (a basicMenuTestApp) SetUIRoot(root widget.Widget) {
	a.app.SetRoot(root)
}

func (a basicMenuTestApp) Frame() {
	a.app.Frame()
}

func (a basicMenuTestApp) Invalidate() {
	if a.app.Window() != nil && a.app.Window().Context() != nil {
		a.app.Window().Context().Invalidate()
	}
}

func (a basicMenuTestApp) InvalidateRect(rect geometry.Rect) {
	if a.app.Window() != nil && a.app.Window().Context() != nil && !rect.IsEmpty() {
		a.app.Window().Context().InvalidateRect(rect)
	}
}

func (a basicMenuTestApp) RequestFullRepaint() {
	if a.app.Window() != nil {
		a.app.Window().RequestFullRepaint()
	}
}

// InvalidateLayout mirrors render.uiAppBridge: newly published in-place
// overlays rely on it to schedule the layout pass that sizes their content.
func (a basicMenuTestApp) InvalidateLayout() {
	if a.app.Window() != nil && a.app.Window().Context() != nil {
		a.app.Window().Context().Invalidate()
	}
}

func (a basicMenuTestApp) WidgetContext() widget.Context {
	if a.app.Window() == nil {
		return nil
	}
	return a.app.Window().Context()
}

func (a basicMenuTestApp) Cursor() widget.CursorType {
	return a.app.Window().Context().Cursor()
}

func (a basicMenuTestApp) HoveredWidget() widget.Widget {
	return a.app.Window().HoveredWidget()
}

// inertOverlay is a minimal widget for manager tests.
type inertOverlay struct {
	widget.WidgetBase
}

func newInertOverlay() *inertOverlay {
	w := &inertOverlay{}
	w.SetVisible(true)
	return w
}
func (w *inertOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(100, 50))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *inertOverlay) Draw(widget.Context, widget.Canvas) {}

func (w *inertOverlay) Event(widget.Context, event.Event) bool {
	return false
}

func (w *inertOverlay) Children() []widget.Widget {
	return nil
}

// basicMenuTestApp adapts a ui app for tests.
