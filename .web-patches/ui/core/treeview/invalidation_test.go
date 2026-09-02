package treeview

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
)

// --- Granular Invalidation Tests (TASK-UI-INVAL-001h) ---
//
// These tests verify that hover changes use granular invalidation
// (SetNeedsRedraw + InvalidateRect) instead of full-tree ctx.Invalidate().
// Selection and toggle operations KEEP full invalidation (structural changes).

func TestGranularInvalidation_HoverMove_NoFullInvalidate(t *testing.T) {
	root := makeTestTree()
	w := New(Root(root), ItemHeight(28), SelectionModeOpt(SelectionSingle))
	ctx := makeCtx()
	layoutAndBounds(w, ctx, 200, 200)

	// Reset context for clean test.
	ctx = makeCtx()

	// Move mouse over a row to change hoveredIndex.
	bounds := w.Bounds()
	moveY := bounds.Min.Y + 14 // middle of first row (itemHeight=28)
	move := event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, moveY), geometry.Pt(100, moveY), event.ModNone)
	handleEvent(w, ctx, move)

	if ctx.IsInvalidated() {
		t.Error("hover move should NOT trigger full invalidation (ctx.Invalidate)")
	}
	if !w.NeedsRedraw() {
		t.Error("hover move should set needsRedraw on widget")
	}
	if ctx.InvalidatedRect().IsEmpty() {
		t.Error("hover move should trigger InvalidateRect with widget bounds")
	}
}

func TestGranularInvalidation_HoverLeave_NoFullInvalidate(t *testing.T) {
	root := makeTestTree()
	w := New(Root(root), ItemHeight(28))
	ctx := makeCtx()
	layoutAndBounds(w, ctx, 200, 200)

	// Hover a row first.
	bounds := w.Bounds()
	moveY := bounds.Min.Y + 14
	move := event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, moveY), geometry.Pt(100, moveY), event.ModNone)
	handleEvent(w, ctx, move)

	if w.hoveredIndex == noHoveredIndex {
		t.Fatal("expected a row to be hovered after move")
	}

	// Reset context and redraw flag.
	ctx = makeCtx()
	w.ClearRedraw()

	// Mouse leave.
	leave := event.NewMouseEvent(event.MouseLeave, event.ButtonNone, 0,
		geometry.Pt(300, 300), geometry.Pt(300, 300), event.ModNone)
	handleEvent(w, ctx, leave)

	if ctx.IsInvalidated() {
		t.Error("MouseLeave should NOT trigger full invalidation")
	}
	if !w.NeedsRedraw() {
		t.Error("MouseLeave should set needsRedraw on widget")
	}
	if w.hoveredIndex != noHoveredIndex {
		t.Errorf("hoveredIndex = %d, want %d (noHoveredIndex)", w.hoveredIndex, noHoveredIndex)
	}
}

func TestGranularInvalidation_InvalidateRect_MatchesBounds(t *testing.T) {
	root := makeTestTree()
	w := New(Root(root), ItemHeight(28))
	ctx := makeCtx()
	layoutAndBounds(w, ctx, 200, 200)

	bounds := w.Bounds()
	ctx = makeCtx()

	moveY := bounds.Min.Y + 14
	move := event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, moveY), geometry.Pt(100, moveY), event.ModNone)
	handleEvent(w, ctx, move)

	got := ctx.InvalidatedRect()
	if got != bounds {
		t.Errorf("InvalidatedRect = %v, want %v (widget bounds)", got, bounds)
	}
}
