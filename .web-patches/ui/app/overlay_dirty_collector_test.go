package app

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/overlay"
	"github.com/gogpu/ui/widget"
)

// TestCollectDirtyRegions_FindsOverlayContent verifies that CollectDirtyRegions
// walks overlay content widgets and adds their dirty regions to the tracker.
//
// This is the core test for the dropdown menu debug overlay bug: menu hover
// calls SetNeedsRedraw(true) on the menu widget, but CollectDirtyRegions
// did not find it because overlay content was not walked.
func TestCollectDirtyRegions_FindsOverlayContent(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	root.SetRepaintBoundary(true)
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	// Push overlay via production path (windowOverlayManager wraps in Container,
	// sets RepaintBoundary=true on content).
	menuContent := newOverlayContent(100, 200, 200, 150)
	mgr := &windowOverlayManager{window: win}
	mgr.PushOverlay(menuContent, nil)

	// Set ScreenOrigin so markWidgetDirty uses correct screen-space bounds.
	// In production, this is done by desktop.draw after CollectDirtyRegions
	// on the first frame, then persists for subsequent frames.
	menuContent.SetScreenOrigin(geometry.Pt(100, 200))

	// Clear initial dirty state (simulates first frame having painted).
	widget.ClearRedrawInTree(root)
	widget.ClearRedrawInTree(menuContent)

	// Verify menuContent is clean.
	if menuContent.NeedsRedraw() {
		t.Fatal("menu should be clean after ClearRedrawInTree")
	}
	if root.NeedsRedraw() {
		t.Fatal("root should be clean after ClearRedrawInTree")
	}

	// Simulate hover: mark content dirty.
	menuContent.SetNeedsRedraw(true)

	if !menuContent.NeedsRedraw() {
		t.Fatal("menu should be dirty after SetNeedsRedraw(true)")
	}

	// CollectDirtyRegions should find the overlay content widget.
	win.CollectDirtyRegions()
	regions := win.DirtyRegions()

	if len(regions) == 0 {
		t.Fatal("CollectDirtyRegions found 0 dirty regions — overlay content not walked")
	}

	// Find a region that matches menu content bounds (100,200 to 300,350).
	found := false
	for _, r := range regions {
		if r.Min.X >= 90 && r.Min.X <= 110 &&
			r.Min.Y >= 190 && r.Min.Y <= 210 &&
			r.Width() >= 150 && r.Width() <= 250 &&
			r.Height() >= 100 && r.Height() <= 200 {
			found = true
			t.Logf("found overlay dirty region: %v (%.0f x %.0f)", r, r.Width(), r.Height())
		}
	}
	if !found {
		t.Errorf("no dirty region matching overlay content bounds (100,200 to 300,350); got %v", regions)
	}
}

// TestCollectDirtyRegions_CleanOverlay_NoDirtyRegions verifies that when
// overlay content is clean (NeedsRedraw=false), CollectDirtyRegions does
// NOT add spurious dirty regions for it.
func TestCollectDirtyRegions_CleanOverlay_NoDirtyRegions(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	root.SetRepaintBoundary(true)
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	// Push clean overlay.
	menuContent := newOverlayContent(100, 200, 200, 150)
	mgr := &windowOverlayManager{window: win}
	mgr.PushOverlay(menuContent, nil)
	menuContent.SetScreenOrigin(geometry.Pt(100, 200))

	// Clear dirty state.
	widget.ClearRedrawInTree(root)
	widget.ClearRedrawInTree(menuContent)

	// Both root and overlay are clean.
	win.CollectDirtyRegions()
	regions := win.DirtyRegions()

	if len(regions) != 0 {
		t.Errorf("clean tree + clean overlay should produce 0 dirty regions, got %d: %v",
			len(regions), regions)
	}
}

// TestCollectDirtyRegions_OverlayWithChildren verifies that when an overlay
// content widget has children with NeedsRedraw=true, the collector finds
// the dirty child (leaf-dirty pattern).
func TestCollectDirtyRegions_OverlayWithChildren(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	root.SetRepaintBoundary(true)
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	// Create a parent overlay content with a dirty child.
	parent := &overlayContentWithChild{}
	parent.SetVisible(true)
	parent.SetEnabled(true)
	parent.SetBounds(geometry.NewRect(100, 200, 200, 150))
	parent.SetScreenOrigin(geometry.Pt(100, 200))

	child := &overlayContent{width: 180, height: 40}
	child.SetVisible(true)
	child.SetEnabled(true)
	child.SetBounds(geometry.NewRect(110, 210, 180, 40))
	child.SetScreenOrigin(geometry.Pt(110, 210))
	parent.child = child

	container := overlay.NewContainer(parent, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Clear initial state.
	widget.ClearRedrawInTree(root)
	widget.ClearRedrawInTree(parent)

	// Mark only the child dirty (simulates hover on one menu item).
	child.SetNeedsRedraw(true)

	win.CollectDirtyRegions()
	regions := win.DirtyRegions()

	if len(regions) == 0 {
		t.Fatal("CollectDirtyRegions found 0 regions — dirty child in overlay not found")
	}

	// Should find a region near child bounds (110,210 to 290,250).
	found := false
	for _, r := range regions {
		if r.Min.X >= 100 && r.Min.Y >= 200 && r.Width() <= 250 {
			found = true
			t.Logf("found dirty child region: %v", r)
		}
	}
	if !found {
		t.Errorf("expected dirty region near child bounds, got %v", regions)
	}
}

// TestCollectDirtyRegions_OverlayHover_NoFullWindowDirty verifies that when
// only the overlay menu is dirty (hover), the dirty collector does NOT report
// a full-window dirty region. This is the regression test for the bug where
// ctx.InvalidateRect from the menu hover handler forced root re-recording,
// producing Rect(0,0,800x600) that masked the menu's small region.
//
// Fix: menu hover uses SetNeedsRedraw only (boundary self-dirty via
// InvalidateScene + onBoundaryDirty callback), NOT ctx.InvalidateRect.
func TestCollectDirtyRegions_OverlayHover_NoFullWindowDirty(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	root.SetRepaintBoundary(true)
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	menuContent := newOverlayContent(100, 200, 200, 150)
	mgr := &windowOverlayManager{window: win}
	mgr.PushOverlay(menuContent, nil)
	menuContent.SetScreenOrigin(geometry.Pt(100, 200))

	// Simulate: first frame painted (root + overlay), everything clean.
	widget.ClearRedrawInTree(root)
	widget.ClearRedrawInTree(menuContent)

	// Simulate hover on menu: ONLY the menu widget is dirty.
	// In production (after fix), menuWidget.handleMouseEvent calls
	// SetNeedsRedraw(true) but NOT ctx.InvalidateRect.
	menuContent.SetNeedsRedraw(true)

	// Root must stay clean.
	if root.NeedsRedraw() {
		t.Fatal("root should NOT be dirty — hover on overlay boundary must not pollute root")
	}

	win.CollectDirtyRegions()
	regions := win.DirtyRegions()

	if len(regions) == 0 {
		t.Fatal("expected at least 1 dirty region for overlay hover")
	}

	// No region should be full-window (800x600).
	for _, r := range regions {
		if r.Width() > 400 || r.Height() > 400 {
			t.Errorf("dirty region %v is too large — expected menu area (~200x150), not full window. "+
				"Root was polluted by ctx.InvalidateRect from overlay hover handler", r)
		}
	}

	t.Logf("dirty regions: %v (should be ~200x150 at (100,200))", regions)
}

// overlayContentWithChild is an overlay content widget that has one child
// (simulates a menu widget that contains items).
type overlayContentWithChild struct {
	widget.WidgetBase
	child widget.Widget
}

func (o *overlayContentWithChild) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(o.Bounds().Size())
}

func (o *overlayContentWithChild) Draw(_ widget.Context, canvas widget.Canvas) {
	if canvas != nil {
		canvas.DrawRect(o.Bounds(), widget.RGBA8(100, 100, 255, 255))
	}
}

func (o *overlayContentWithChild) Event(_ widget.Context, _ event.Event) bool { return false }

func (o *overlayContentWithChild) Children() []widget.Widget {
	if o.child == nil {
		return nil
	}
	return []widget.Widget{o.child}
}
