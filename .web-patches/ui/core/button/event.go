package button

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/widget"
)

// handleEvent processes input events for the button widget.
// It manages hover, press, and keyboard activation states.
func handleEvent(w *Widget, ctx widget.Context, e event.Event) bool {
	// Disabled buttons ignore all interaction.
	if w.cfg.ResolvedDisabled() {
		return false
	}

	switch ev := e.(type) {
	case *event.MouseEvent:
		return handleMouseEvent(w, ctx, ev)
	case *event.KeyEvent:
		return handleKeyEvent(w, ev)
	default:
		return false
	}
}

// requestStateRepaint marks the button for repaint after a visual state
// transition. On the web build this is a no-op: each transition re-rasters
// the button on the CPU, and a tap walks hover→press→release in quick
// succession — three rasters per tap read as a stutter on tablets. The
// state machine still runs; only the visual feedback is dropped.
func requestStateRepaint(w *Widget, ctx widget.Context) {
	if !visualStateTransitionsEnabled {
		return
	}
	w.SetNeedsRedraw(true)
	// ScreenBounds: the invalidation rect is consumed in window space —
	// local bounds (origin 0,0) repainted a phantom top-left corner.
	ctx.InvalidateRect(w.ScreenBounds())
}

// handleMouseEvent processes mouse events for hover, press, and click.
func handleMouseEvent(w *Widget, ctx widget.Context, e *event.MouseEvent) bool {
	switch e.MouseType {
	case event.MouseEnter:
		w.state = stateHover
		ctx.SetCursor(widget.CursorPointer)
		requestStateRepaint(w, ctx)
		return true

	case event.MouseLeave:
		w.state = stateNormal
		ctx.SetCursor(widget.CursorDefault)
		requestStateRepaint(w, ctx)
		return true

	case event.MousePress:
		if e.Button != event.ButtonLeft {
			return false
		}
		w.state = statePressed
		ctx.RequestFocus(w)
		requestStateRepaint(w, ctx)
		return true

	case event.MouseRelease:
		if e.Button != event.ButtonLeft {
			return false
		}
		wasPressed := w.state == statePressed
		// Check if release is inside bounds.
		if w.Bounds().Contains(e.Position) {
			w.state = stateHover
		} else {
			w.state = stateNormal
		}
		requestStateRepaint(w, ctx)
		if wasPressed && w.Bounds().Contains(e.Position) {
			fireOnClick(w)
		}
		return true

	default:
		return false
	}
}

// handleKeyEvent processes keyboard events for Enter/Space activation.
func handleKeyEvent(w *Widget, e *event.KeyEvent) bool {
	if !w.IsFocused() {
		return false
	}

	switch e.Key {
	case event.KeyEnter, event.KeySpace:
		return handleActivationKey(w, e)
	default:
		return false
	}
}

// handleActivationKey processes Enter/Space key press and release for button activation.
func handleActivationKey(w *Widget, e *event.KeyEvent) bool {
	switch e.KeyType {
	case event.KeyPress:
		w.state = statePressed
		return true
	case event.KeyRelease:
		wasPressed := w.state == statePressed
		w.state = stateNormal
		if wasPressed {
			fireOnClick(w)
		}
		return true
	default:
		return false
	}
}

// fireOnClick calls the configured onClick handler if present.
func fireOnClick(w *Widget) {
	if w.cfg.onClick != nil {
		w.cfg.onClick()
	}
}
