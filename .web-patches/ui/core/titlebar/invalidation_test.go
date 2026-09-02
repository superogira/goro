package titlebar

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- Granular Invalidation Tests (TASK-UI-INVAL-001j) ---
//
// These tests verify that control button press/release/hover use granular
// invalidation (SetNeedsRedraw + InvalidateRect) instead of full-tree
// ctx.Invalidate(). Control actions (minimize, maximize, close) fire
// on release but the visual state change is still granular.

func TestGranularInvalidation_ControlPress_NoFullInvalidate(t *testing.T) {
	chrome := &mockChrome{}
	tb := New(Chrome(chrome))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(800, 40))
	tb.Layout(ctx, constraints)

	ctx = widget.NewContext()

	// Press on close button.
	closeCenter := tb.controlBounds[controlIdxClose].Center()
	consumed := tb.handlePress(ctx, closeCenter)

	if !consumed {
		t.Error("press on control button should be consumed")
	}
	if ctx.IsInvalidated() {
		t.Error("control press should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("control press should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("control press should trigger InvalidateRect")
	}
	if tb.controlStates[controlIdxClose] != statePressed {
		t.Errorf("close state = %v, want statePressed", tb.controlStates[controlIdxClose])
	}
}

func TestGranularInvalidation_ControlRelease_NoFullInvalidate(t *testing.T) {
	chrome := &mockChrome{}
	tb := New(Chrome(chrome))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(800, 40))
	tb.Layout(ctx, constraints)

	// Press first.
	closeCenter := tb.controlBounds[controlIdxClose].Center()
	tb.handlePress(widget.NewContext(), closeCenter)

	// Reset context and redraw for release test.
	ctx = widget.NewContext()
	tb.ClearRedraw()

	consumed := tb.handleRelease(ctx, closeCenter)

	if !consumed {
		t.Error("release on control should be consumed")
	}
	if ctx.IsInvalidated() {
		t.Error("control release should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("control release should set needsRedraw on widget")
	}
	if !chrome.closeCalled {
		t.Error("Close() should have been called on release inside button")
	}
}

func TestGranularInvalidation_ControlHover_NoFullInvalidate(t *testing.T) {
	chrome := &mockChrome{}
	tb := New(Chrome(chrome))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(800, 40))
	tb.Layout(ctx, constraints)

	ctx = widget.NewContext()

	// Move over close button.
	closeCenter := tb.controlBounds[controlIdxClose].Center()
	changed := tb.handleMove(ctx, closeCenter)

	if !changed {
		t.Error("handleMove should report changed=true when control hover changes")
	}
	if ctx.IsInvalidated() {
		t.Error("control hover should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("control hover should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("control hover should trigger InvalidateRect")
	}
	if tb.controlStates[controlIdxClose] != stateHover {
		t.Errorf("close state = %v, want stateHover", tb.controlStates[controlIdxClose])
	}
}

func TestGranularInvalidation_ClearControlHovers_NoFullInvalidate(t *testing.T) {
	chrome := &mockChrome{}
	tb := New(Chrome(chrome))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(800, 40))
	tb.Layout(ctx, constraints)

	// Set one control to hover state.
	tb.controlStates[controlIdxMinimize] = stateHover

	ctx = widget.NewContext()
	changed := tb.clearControlHovers(ctx)

	if !changed {
		t.Error("clearControlHovers should report changed=true")
	}
	if ctx.IsInvalidated() {
		t.Error("clearControlHovers should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("clearControlHovers should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("clearControlHovers should trigger InvalidateRect")
	}
}

func TestGranularInvalidation_InvalidateRect_MatchesBounds(t *testing.T) {
	chrome := &mockChrome{}
	tb := New(Chrome(chrome))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(800, 40))
	tb.Layout(ctx, constraints)

	bounds := tb.Bounds()
	ctx = widget.NewContext()

	closeCenter := tb.controlBounds[controlIdxClose].Center()
	tb.handlePress(ctx, closeCenter)

	got := ctx.InvalidatedRect()
	if got != bounds {
		t.Errorf("InvalidatedRect = %v, want %v (widget bounds)", got, bounds)
	}
}
