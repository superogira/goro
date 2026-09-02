package toolbar

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"
)

// --- Granular Invalidation Tests (TASK-UI-INVAL-001i) ---
//
// These tests verify that press/release/hover use granular invalidation
// (SetNeedsRedraw + InvalidateRect) instead of full-tree ctx.Invalidate().

func TestGranularInvalidation_Press_NoFullInvalidate(t *testing.T) {
	clicked := false
	tb := New(Items(
		IconButton("A", icon.Add, func() { clicked = true }),
	))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(400, 40))
	tb.Layout(ctx, constraints)
	ctx = widget.NewContext()

	// Press on the button item.
	center := tb.itemStates[0].bounds.Center()
	consumed := tb.handlePress(ctx, center)

	if !consumed {
		t.Error("press on button should be consumed")
	}
	if ctx.IsInvalidated() {
		t.Error("press should NOT trigger full invalidation (ctx.Invalidate)")
	}
	if !tb.NeedsRedraw() {
		t.Error("press should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("press should trigger InvalidateRect")
	}
	if clicked {
		t.Error("click callback should not fire on press")
	}
}

func TestGranularInvalidation_Release_NoFullInvalidate(t *testing.T) {
	clicked := false
	tb := New(Items(
		IconButton("A", icon.Add, func() { clicked = true }),
	))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(400, 40))
	tb.Layout(ctx, constraints)

	// Press first.
	center := tb.itemStates[0].bounds.Center()
	tb.handlePress(widget.NewContext(), center)

	// Reset context and redraw flag for release test.
	ctx = widget.NewContext()
	tb.ClearRedraw()

	consumed := tb.handleRelease(ctx, center)

	if !consumed {
		t.Error("release should be consumed")
	}
	if ctx.IsInvalidated() {
		t.Error("release should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("release should set needsRedraw on widget")
	}
	if !clicked {
		t.Error("click callback should fire on release inside same item")
	}
}

func TestGranularInvalidation_HoverChange_NoFullInvalidate(t *testing.T) {
	tb := New(Items(
		IconButton("A", icon.Add, nil),
		IconButton("B", icon.Close, nil),
	))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(400, 40))
	tb.Layout(ctx, constraints)
	ctx = widget.NewContext()

	// Move over second item.
	center := tb.itemStates[1].bounds.Center()
	changed := tb.handleMove(ctx, center)

	if !changed {
		t.Error("handleMove should report changed=true when hover changes")
	}
	if ctx.IsInvalidated() {
		t.Error("hover change should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("hover change should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("hover change should trigger InvalidateRect")
	}
}

func TestGranularInvalidation_ClearHoverStates_NoFullInvalidate(t *testing.T) {
	tb := New(Items(
		IconButton("A", icon.Add, nil),
		IconButton("B", icon.Close, nil),
	))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(400, 40))
	tb.Layout(ctx, constraints)

	// Set one item to hover state.
	tb.itemStates[0].interaction = stateHover

	ctx = widget.NewContext()
	changed := tb.clearHoverStates(ctx)

	if !changed {
		t.Error("clearHoverStates should report changed=true")
	}
	if ctx.IsInvalidated() {
		t.Error("clearHoverStates should NOT trigger full invalidation")
	}
	if !tb.NeedsRedraw() {
		t.Error("clearHoverStates should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("clearHoverStates should trigger InvalidateRect")
	}
}

func TestGranularInvalidation_InvalidateRect_MatchesBounds(t *testing.T) {
	tb := New(Items(
		IconButton("A", icon.Add, nil),
	))
	ctx := widget.NewContext()
	constraints := geometry.Loose(geometry.Sz(400, 40))
	tb.Layout(ctx, constraints)

	bounds := tb.Bounds()
	ctx = widget.NewContext()

	center := tb.itemStates[0].bounds.Center()
	tb.handlePress(ctx, center)

	got := ctx.InvalidatedRect()
	if got != bounds {
		t.Errorf("InvalidatedRect = %v, want %v (widget bounds)", got, bounds)
	}
}
