package app

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/overlay"
	"github.com/gogpu/ui/widget"
)

// --- Test helpers for overlay hover (Bug 1 + Bug 2) ---
//
// Reuses hoverTrackingWidget and hoverContainer from window_test.go.
// Additional helpers below for overlay-specific scenarios.

// overlayMenuWidget is a widget that tracks hover events and has configurable
// bounds. Unlike hoverTrackingWidget from window_test.go, it also calls
// SetNeedsRedraw on hover (simulating real dropdown menu item behavior).
type overlayMenuWidget struct {
	widget.WidgetBase
	enterCount int
	leaveCount int
}

func newOverlayMenuWidget(x, y, w, h float32) *overlayMenuWidget {
	ow := &overlayMenuWidget{}
	ow.SetVisible(true)
	ow.SetEnabled(true)
	ow.SetBounds(geometry.NewRect(x, y, w, h))
	ow.SetScreenOrigin(geometry.Pt(x, y))
	return ow
}

func (o *overlayMenuWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(o.Bounds().Size())
}

func (o *overlayMenuWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (o *overlayMenuWidget) Event(_ widget.Context, e event.Event) bool {
	if me, ok := e.(*event.MouseEvent); ok {
		switch me.MouseType {
		case event.MouseEnter:
			o.enterCount++
			o.SetNeedsRedraw(true)
			return true
		case event.MouseLeave:
			o.leaveCount++
			o.SetNeedsRedraw(true)
			return true
		}
	}
	return false
}

func (o *overlayMenuWidget) Children() []widget.Widget { return nil }

// overlayMenuContainer holds children for overlay menu content.
type overlayMenuContainer struct {
	widget.WidgetBase
	kids []widget.Widget
}

func newOverlayMenuContainer(x, y, w, h float32, kids ...widget.Widget) *overlayMenuContainer {
	c := &overlayMenuContainer{kids: kids}
	c.SetVisible(true)
	c.SetEnabled(true)
	c.SetBounds(geometry.NewRect(x, y, w, h))
	c.SetScreenOrigin(geometry.Pt(x, y))
	return c
}

func (c *overlayMenuContainer) Layout(_ widget.Context, cs geometry.Constraints) geometry.Size {
	return cs.Constrain(c.Bounds().Size())
}

func (c *overlayMenuContainer) Draw(_ widget.Context, _ widget.Canvas) {}

func (c *overlayMenuContainer) Event(_ widget.Context, _ event.Event) bool {
	return false
}

func (c *overlayMenuContainer) Children() []widget.Widget {
	return c.kids
}

// --- Tests ---

// TestOverlayBlocksBackgroundHover verifies that when a dropdown overlay is
// open, mouse hover events go to the overlay content widget, NOT to the
// background widget tree behind it. This is the Flutter ModalBarrier pattern.
//
// Setup: root has a background item at (50,50), overlay menu has an item
// at (60,60). Mouse moves to (100,70) which is inside both.
// Expected: overlay item receives hover, background item does NOT.
func TestOverlayBlocksBackgroundHover(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	// Background widget at (50,50)-(250,90) — simulates a ListView item.
	bgItem := newHoverWidget(geometry.NewRect(50, 50, 250, 90))

	// Root container with the background item.
	root := newHoverContainer(bgItem)
	win.SetRoot(root)

	// Overlay content at (50,50)-(250,200) — simulates a dropdown menu.
	overlayItem := newOverlayMenuWidget(60, 60, 180, 130)
	menuContent := newOverlayMenuContainer(50, 50, 200, 150, overlayItem)

	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	// Container covers full window. Set its screen bounds so hitTest works.
	container.SetBounds(geometry.NewRect(0, 0, 800, 600))
	container.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(container)

	// Simulate mouse move to (100, 70) — inside both overlay and background.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, 70), geometry.Pt(100, 70), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	// Overlay item should receive hover.
	if overlayItem.enterCount == 0 {
		t.Error("overlay item should receive MouseEnter — hover passed through to background")
	}

	// Background item should NOT receive hover.
	if bgItem.enterCount > 0 {
		t.Error("background item should NOT receive MouseEnter while overlay is open")
	}

	// Hovered widget should be the overlay item.
	if win.HoveredWidget() != overlayItem {
		t.Errorf("HoveredWidget = %T, want overlay menu item", win.HoveredWidget())
	}
}

// TestOverlayBlocksBackgroundHover_OutsideContent verifies that when the
// mouse is outside the overlay content (but still inside the window),
// neither the overlay content nor the background widgets receive hover.
func TestOverlayBlocksBackgroundHover_OutsideContent(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	// Background widget at (400,400)-(600,440).
	bgItem := newHoverWidget(geometry.NewRect(400, 400, 600, 440))
	root := newHoverContainer(bgItem)
	win.SetRoot(root)

	// Overlay content at (50,50)-(250,200) — dropdown menu.
	menuItem := newOverlayMenuWidget(60, 60, 180, 130)
	menuContent := newOverlayMenuContainer(50, 50, 200, 150, menuItem)

	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	container.SetBounds(geometry.NewRect(0, 0, 800, 600))
	container.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(container)

	// Mouse at (450, 420) — inside background item, outside overlay content.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(450, 420), geometry.Pt(450, 420), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	// Neither should receive hover.
	if bgItem.enterCount > 0 {
		t.Error("background item should NOT receive hover when overlay is open")
	}
	if menuItem.enterCount > 0 {
		t.Error("overlay item should NOT receive hover — mouse is outside its bounds")
	}

	// Hovered widget should be nil (overlay blocks background).
	if win.HoveredWidget() != nil {
		t.Errorf("HoveredWidget = %T, want nil (overlay blocks background)", win.HoveredWidget())
	}
}

// TestNoOverlay_NormalHoverBehavior verifies that when no overlays are open,
// hover works normally through the root widget tree (regression test).
func TestNoOverlay_NormalHoverBehavior(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	bgItem := newHoverWidget(geometry.NewRect(50, 50, 250, 90))
	root := newHoverContainer(bgItem)
	win.SetRoot(root)

	// Mouse at (100, 70) — inside background item, no overlay.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, 70), geometry.Pt(100, 70), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	// Background item should receive hover normally.
	if bgItem.enterCount == 0 {
		t.Error("background item should receive MouseEnter when no overlay is open")
	}
	if win.HoveredWidget() != bgItem {
		t.Errorf("HoveredWidget = %T, want background item", win.HoveredWidget())
	}
}

// TestOverlayClose_RestoresNormalHover verifies that after closing an
// overlay, hover events resume going to the root widget tree.
func TestOverlayClose_RestoresNormalHover(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	bgItem := newHoverWidget(geometry.NewRect(50, 50, 250, 90))
	root := newHoverContainer(bgItem)
	win.SetRoot(root)

	// Open overlay.
	menuContent := newOverlayMenuWidget(50, 50, 200, 150)
	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	container.SetBounds(geometry.NewRect(0, 0, 800, 600))
	container.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(container)

	// Mouse move while overlay open — should not hover background.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, 70), geometry.Pt(100, 70), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	if bgItem.enterCount > 0 {
		t.Error("background should not get hover while overlay open")
	}

	// Close overlay.
	win.Overlays().Pop()

	// Move mouse away then back to force a hover change.
	moveAway := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(700, 500), geometry.Pt(700, 500), event.ModNone,
	)
	win.HandleEvent(moveAway)

	moveBack := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(100, 70), geometry.Pt(100, 70), event.ModNone,
	)
	win.HandleEvent(moveBack)

	// Now background should receive hover.
	if bgItem.enterCount == 0 {
		t.Error("background should receive hover after overlay is closed")
	}
}

// TestOverlayHoverProducesGreenDamage verifies that when an overlay content
// boundary goes dirty (from hover), DirtyOverlayContentRects captures the
// overlay content rect. This is the data source for green debug overlay in
// desktop.go (boundaryDamageLogical → TrackDamageRect).
func TestOverlayHoverProducesGreenDamage(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Create overlay with content at known position.
	menuItem := newOverlayMenuWidget(110, 210, 180, 130)
	menuContent := newOverlayMenuContainer(100, 200, 200, 150, menuItem)

	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	container.SetBounds(geometry.NewRect(0, 0, 800, 600))
	container.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(container)

	// Clear initial dirty state.
	win.ClearOverlayRedraw()

	// Simulate hover: move mouse into overlay content.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(150, 250), geometry.Pt(150, 250), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	// Overlay content should be dirty from hover (MouseEnter on menuItem).
	if !win.HasDirtyOverlays() {
		t.Fatal("HasDirtyOverlays should be true after hover on overlay content")
	}

	// DirtyOverlayContentRects should contain the overlay content rect.
	// desktop.go uses this to add TrackDamageRect for green debug overlay.
	rects := win.DirtyOverlayContentRects()
	if len(rects) == 0 {
		t.Fatal("DirtyOverlayContentRects should contain overlay content rect after hover")
	}

	// The damage rect should be the content area, not full window.
	r := rects[0]
	if r.Width() > 300 || r.Height() > 300 {
		t.Errorf("damage rect %v too large — should be content area (~200x150), not full window", r)
	}
	if r.Width() < 100 || r.Height() < 50 {
		t.Errorf("damage rect %v too small — should cover content area", r)
	}

	t.Logf("overlay content bounds: %v", menuContent.Bounds())
	t.Logf("damage rect: %v", r)
}

// TestMultipleOverlays_TopOverlayGetsHover verifies that with stacked
// overlays, the topmost overlay receives hover priority.
func TestMultipleOverlays_TopOverlayGetsHover(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newHoverContainer()
	win.SetRoot(root)

	// Bottom overlay at (50,50)-(250,210).
	bottomItem := newOverlayMenuWidget(60, 60, 180, 140)
	bottomContent := newOverlayMenuContainer(50, 50, 200, 160, bottomItem)
	bottomContainer := overlay.NewContainer(bottomContent, geometry.Sz(800, 600))
	bottomContainer.SetBounds(geometry.NewRect(0, 0, 800, 600))
	bottomContainer.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(bottomContainer)

	// Top overlay at (80,80)-(260,230) — overlaps bottom, slightly narrower.
	topItem := newOverlayMenuWidget(90, 90, 160, 130)
	topContent := newOverlayMenuContainer(80, 80, 180, 150, topItem)
	topContainer := overlay.NewContainer(topContent, geometry.Sz(800, 600))
	topContainer.SetBounds(geometry.NewRect(0, 0, 800, 600))
	topContainer.SetScreenOrigin(geometry.Pt(0, 0))
	win.Overlays().Push(topContainer)

	// Mouse at (120, 120) — inside both overlays.
	moveEvt := event.NewMouseEvent(
		event.MouseMove, event.ButtonNone, 0,
		geometry.Pt(120, 120), geometry.Pt(120, 120), event.ModNone,
	)
	win.HandleEvent(moveEvt)

	// Top overlay should get hover.
	if topItem.enterCount == 0 {
		t.Error("top overlay item should receive MouseEnter")
	}

	// Bottom overlay should NOT get hover (top overlay takes priority).
	if bottomItem.enterCount > 0 {
		t.Error("bottom overlay item should NOT receive MouseEnter when top overlay handles it")
	}
}
