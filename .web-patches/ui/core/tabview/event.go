package tabview

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/widget"
)

// handleEvent processes input events for the tabview widget.
func handleEvent(w *Widget, ctx widget.Context, e event.Event) bool {
	switch ev := e.(type) {
	case *event.MouseEvent:
		return handleMouseEvent(w, ctx, ev)
	case *event.KeyEvent:
		return handleKeyEvent(w, ctx, ev)
	default:
		return false
	}
}

// handleMouseEvent processes mouse events for tab selection and close.
func handleMouseEvent(w *Widget, ctx widget.Context, e *event.MouseEvent) bool {
	switch e.MouseType {
	case event.MousePress:
		return handleMousePress(w, ctx, e)
	case event.MouseMove:
		return handleMouseMove(w, ctx, e)
	case event.MouseLeave:
		return handleMouseLeave(w, ctx)
	default:
		return false
	}
}

// handleMousePress handles mouse click on tabs and close buttons.
func handleMousePress(w *Widget, ctx widget.Context, e *event.MouseEvent) bool {
	if e.Button != event.ButtonLeft {
		return false
	}

	// Check if click is in the tab bar area.
	if !w.tabBarBounds.Contains(e.Position) {
		return false
	}

	ctx.RequestFocus(w)

	// Check close buttons first.
	for i := range w.tabStates {
		ts := &w.tabStates[i]
		if !ts.Closeable || ts.CloseButtonBounds.IsEmpty() {
			continue
		}
		if ts.CloseButtonBounds.Contains(e.Position) {
			if w.cfg.onClose != nil {
				w.cfg.onClose(i)
			}
			// ADR-028: layout change — tab removal changes tab strip layout.
			ctx.Invalidate()
			return true
		}
	}

	// Check tab selection.
	for i := range w.tabStates {
		ts := &w.tabStates[i]
		if ts.Disabled {
			continue
		}
		if ts.Bounds.Contains(e.Position) {
			w.selectTab(ctx, i)
			return true
		}
	}

	// ADR-028: visual only — focus requested but no tab selected.
	w.SetNeedsRedraw(true)
	ctx.InvalidateRect(w.Bounds())
	return true
}

// handleMouseMove updates hover state.
func handleMouseMove(w *Widget, ctx widget.Context, e *event.MouseEvent) bool {
	changed := false
	for i := range w.tabStates {
		ts := &w.tabStates[i]
		wasHovered := ts.Hovered
		ts.Hovered = ts.Bounds.Contains(e.Position) && !ts.Disabled
		if ts.Hovered != wasHovered {
			changed = true
		}
	}
	if changed {
		w.SetNeedsRedraw(true)
		ctx.InvalidateRect(w.Bounds())
	}
	return changed
}

// handleMouseLeave clears all hover states.
func handleMouseLeave(w *Widget, ctx widget.Context) bool {
	changed := false
	for i := range w.tabStates {
		if w.tabStates[i].Hovered {
			w.tabStates[i].Hovered = false
			changed = true
		}
	}
	if changed {
		w.SetNeedsRedraw(true)
		ctx.InvalidateRect(w.Bounds())
	}
	return changed
}

// handleKeyEvent processes keyboard events for tab navigation.
func handleKeyEvent(w *Widget, ctx widget.Context, e *event.KeyEvent) bool {
	if !w.IsFocused() {
		return false
	}
	if e.KeyType != event.KeyPress {
		return false
	}

	switch e.Key {
	case event.KeyLeft:
		return navigatePrev(w, ctx)
	case event.KeyRight:
		return navigateNext(w, ctx)
	case event.KeyHome:
		return navigateFirst(w, ctx)
	case event.KeyEnd:
		return navigateLast(w, ctx)
	default:
		return false
	}
}

// navigatePrev selects the previous enabled tab.
func navigatePrev(w *Widget, ctx widget.Context) bool {
	current := w.cfg.ResolvedSelected()
	tabCount := len(w.cfg.tabs)
	for i := 1; i < tabCount; i++ {
		idx := (current - i + tabCount) % tabCount
		if !w.cfg.tabs[idx].Disabled {
			w.selectTab(ctx, idx)
			return true
		}
	}
	return false
}

// navigateNext selects the next enabled tab.
func navigateNext(w *Widget, ctx widget.Context) bool {
	current := w.cfg.ResolvedSelected()
	tabCount := len(w.cfg.tabs)
	for i := 1; i < tabCount; i++ {
		idx := (current + i) % tabCount
		if !w.cfg.tabs[idx].Disabled {
			w.selectTab(ctx, idx)
			return true
		}
	}
	return false
}

// navigateFirst selects the first enabled tab.
func navigateFirst(w *Widget, ctx widget.Context) bool {
	for i := range w.cfg.tabs {
		if !w.cfg.tabs[i].Disabled {
			w.selectTab(ctx, i)
			return true
		}
	}
	return false
}

// navigateLast selects the last enabled tab.
func navigateLast(w *Widget, ctx widget.Context) bool {
	for i := len(w.cfg.tabs) - 1; i >= 0; i-- {
		if !w.cfg.tabs[i].Disabled {
			w.selectTab(ctx, i)
			return true
		}
	}
	return false
}

// selectTab sets the selected tab index and fires callbacks.
func (w *Widget) selectTab(ctx widget.Context, idx int) {
	if idx == w.cfg.ResolvedSelected() {
		return
	}
	w.cfg.setSelected(idx)
	if w.cfg.onSelect != nil {
		w.cfg.onSelect(idx)
	}
	// ADR-028: layout change — tab switch changes content panel.
	ctx.Invalidate()
}
