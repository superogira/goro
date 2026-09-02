package app

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/overlay"
	"github.com/gogpu/ui/widget"
)

// --- Test helpers for overlay damage tracking (ADR-029) ---

// overlayContent is a minimal widget for overlay content that tracks Draw calls
// and has configurable intrinsic size. It never fills the full window, so its
// bounds are tighter than the Container backdrop (full-window).
type overlayContent struct {
	widget.WidgetBase
	drawCount int
	width     float32
	height    float32
}

func newOverlayContent(x, y, w, h float32) *overlayContent {
	oc := &overlayContent{width: w, height: h}
	oc.SetVisible(true)
	oc.SetEnabled(true)
	oc.SetBounds(geometry.NewRect(x, y, w, h))
	// Mark dirty to simulate production state: overlay content widgets are
	// always dirty when first pushed (either from MountTree or widget init).
	oc.SetNeedsRedraw(true)
	return oc
}

func (o *overlayContent) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(o.width, o.height))
}

func (o *overlayContent) Draw(_ widget.Context, canvas widget.Canvas) {
	o.drawCount++
	if canvas != nil {
		canvas.DrawRect(o.Bounds(), widget.RGBA8(100, 100, 255, 255))
	}
}

func (o *overlayContent) Event(_ widget.Context, _ event.Event) bool { return false }
func (o *overlayContent) Children() []widget.Widget                  { return nil }

// overlayRoot is a minimal root widget for overlay damage tests.
type overlayRoot struct {
	widget.WidgetBase
}

func newOverlayRoot(size geometry.Size) *overlayRoot {
	root := &overlayRoot{}
	root.SetVisible(true)
	root.SetEnabled(true)
	root.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	return root
}

func (r *overlayRoot) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(r.Bounds().Width(), r.Bounds().Height()))
}

func (r *overlayRoot) Draw(_ widget.Context, canvas widget.Canvas) {
	if canvas != nil {
		canvas.DrawRect(r.Bounds(), widget.RGBA8(255, 255, 255, 255))
	}
}

func (r *overlayRoot) Event(_ widget.Context, _ event.Event) bool { return false }
func (r *overlayRoot) Children() []widget.Widget                  { return nil }

// --- Tests ---

// TestNoOverlay_ZeroDamage verifies that when no overlays are present,
// HasDirtyOverlays returns false and DirtyOverlayContentRects returns nil.
func TestNoOverlay_ZeroDamage(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	if win.HasOverlays() {
		t.Error("HasOverlays should be false with no overlays")
	}
	if win.OverlayCount() != 0 {
		t.Errorf("OverlayCount = %d, want 0", win.OverlayCount())
	}
	if win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be false with no overlays")
	}
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 0 {
		t.Errorf("DirtyOverlayContentRects = %v, want empty", rects)
	}
}

// TestHasDirtyOverlays_ReturnsCorrectState verifies HasDirtyOverlays tracks
// NeedsRedraw state on overlay content widgets correctly.
func TestHasDirtyOverlays_ReturnsCorrectState(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	content := newOverlayContent(200, 150, 200, 100)
	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Initially, content has NeedsRedraw from construction.
	if !win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be true after pushing overlay with dirty content")
	}

	// Clear overlay redraw flags.
	win.ClearOverlayRedraw()

	if win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be false after ClearOverlayRedraw")
	}

	// Mark content dirty again (simulating hover event).
	content.SetNeedsRedraw(true)

	if !win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be true after marking content dirty")
	}
}

// TestClearOverlayRedraw_ClearsAllOverlays verifies that ClearOverlayRedraw
// clears NeedsRedraw on all overlay widgets in the stack.
func TestClearOverlayRedraw_ClearsAllOverlays(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Push two overlays.
	content1 := newOverlayContent(100, 100, 200, 100)
	content2 := newOverlayContent(300, 200, 150, 80)
	container1 := overlay.NewContainer(content1, geometry.Sz(800, 600))
	container2 := overlay.NewContainer(content2, geometry.Sz(800, 600))
	win.Overlays().Push(container1)
	win.Overlays().Push(container2)

	// Both should be dirty initially.
	if !win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be true with dirty overlays")
	}

	// Clear all.
	win.ClearOverlayRedraw()

	if win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be false after ClearOverlayRedraw")
	}

	// Verify individual content widgets are clean.
	if content1.NeedsRedraw() {
		t.Error("content1 NeedsRedraw should be false after clear")
	}
	if content2.NeedsRedraw() {
		t.Error("content2 NeedsRedraw should be false after clear")
	}
}

// TestDropdownOverlay_DamageRectIsMenuArea verifies that for non-modal
// overlays (dropdowns), DirtyOverlayContentRects returns the CONTENT
// widget's bounds (menu area), NOT the full-window Container bounds.
//
// This is the core ADR-029 test: Container.Draw draws a full-window
// backdrop, but the damage rect should only cover the dropdown menu.
func TestDropdownOverlay_DamageRectIsMenuArea(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Simulate a dropdown menu at position (100, 200), size 200x150.
	// In a real scenario, the dropdown widget would be pushed as overlay content.
	menuContent := newOverlayContent(100, 200, 200, 150)
	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Container bounds = full window (0,0, 800,600).
	ctx := win.Context()
	container.Layout(ctx, geometry.Tight(geometry.Sz(800, 600)))

	containerBounds := container.Bounds()
	if containerBounds.Width() != 800 || containerBounds.Height() != 600 {
		t.Fatalf("Container bounds = %v, want full window (800x600)", containerBounds)
	}

	// Content bounds = menu area only (100, 200, 200, 150).
	contentBounds := menuContent.Bounds()
	if contentBounds.Width() != 200 || contentBounds.Height() != 150 {
		t.Fatalf("Content bounds = %v, want (200x150)", contentBounds)
	}

	// DirtyOverlayContentRects should return CONTENT bounds, not Container bounds.
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 1 {
		t.Fatalf("DirtyOverlayContentRects count = %d, want 1", len(rects))
	}

	r := rects[0]
	// Damage rect should match content bounds (menu area), not full window.
	if r.Width() > 250 || r.Height() > 200 {
		t.Errorf("damage rect %v too large — should be menu area (~200x150), not full window", r)
	}
	if r.Width() < 150 || r.Height() < 100 {
		t.Errorf("damage rect %v too small — should cover menu content area", r)
	}

	t.Logf("Container bounds: %v (full window)", containerBounds)
	t.Logf("Content bounds: %v (menu area)", contentBounds)
	t.Logf("Damage rect: %v (should match content)", r)
}

// TestModalOverlay_ScrimSeparateFromContent verifies that for modal overlays
// (dialogs), the damage rect covers only the dialog content area, not the
// full-window scrim backdrop.
//
// Flutter equivalent: ModalBarrier is event-only (no draw contribution to
// damage), dialog content is in its own RepaintBoundary.
func TestModalOverlay_ScrimSeparateFromContent(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Modal dialog at center: (250, 150) with size 300x300.
	dialogContent := newOverlayContent(250, 150, 300, 300)
	container := overlay.NewContainer(dialogContent, geometry.Sz(800, 600),
		overlay.WithModal(true),
	)
	win.Overlays().Push(container)

	ctx := win.Context()
	container.Layout(ctx, geometry.Tight(geometry.Sz(800, 600)))

	// Verify modal scrim covers full window.
	if !container.Modal() {
		t.Fatal("container should be modal")
	}
	containerBounds := container.Bounds()
	if containerBounds.Width() != 800 || containerBounds.Height() != 600 {
		t.Fatalf("modal Container bounds = %v, want full window", containerBounds)
	}

	// DirtyOverlayContentRects returns CONTENT bounds (dialog area).
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 1 {
		t.Fatalf("DirtyOverlayContentRects count = %d, want 1", len(rects))
	}

	r := rects[0]
	// Damage rect should be dialog content area (~300x300), not full window.
	if r.Width() > 400 {
		t.Errorf("modal damage rect width = %.0f, want ~300 (dialog content), not 800 (full window)", r.Width())
	}
	if r.Height() > 400 {
		t.Errorf("modal damage rect height = %.0f, want ~300 (dialog content), not 600 (full window)", r.Height())
	}

	t.Logf("Modal Container bounds: %v (full window scrim)", containerBounds)
	t.Logf("Dialog content bounds: %v", dialogContent.Bounds())
	t.Logf("Damage rect: %v (should match dialog content)", r)
}

// TestOverlayHover_DamageOnHoveredItemOnly verifies that when overlay content
// is marked dirty (e.g., from a hover event on a menu item), the damage rect
// covers only the content area.
func TestOverlayHover_DamageOnHoveredItemOnly(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	menuContent := newOverlayContent(100, 200, 200, 150)
	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Clear initial dirty state.
	win.ClearOverlayRedraw()

	// Simulate hover: mark content dirty.
	menuContent.SetNeedsRedraw(true)

	if !win.HasDirtyOverlays() {
		t.Fatal("HasDirtyOverlays should be true after hover")
	}

	rects := win.DirtyOverlayContentRects()
	if len(rects) != 1 {
		t.Fatalf("DirtyOverlayContentRects count = %d, want 1", len(rects))
	}

	r := rects[0]
	// Should be menu content area, not full window.
	if r.Width() > 250 {
		t.Errorf("hover damage width = %.0f, too wide — should be menu area (~200)", r.Width())
	}

	// After clearing, should be clean.
	win.ClearOverlayRedraw()
	rects = win.DirtyOverlayContentRects()
	if len(rects) != 0 {
		t.Errorf("after clear, DirtyOverlayContentRects = %v, want empty", rects)
	}
}

// TestOverlayClose_ContainerRemoved verifies that after removing an overlay
// (simulating dropdown close), the overlay is no longer in the stack and
// overlay damage tracking reports zero rects.
func TestOverlayClose_ContainerRemoved(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	menuContent := newOverlayContent(100, 200, 200, 150)
	container := overlay.NewContainer(menuContent, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	if win.OverlayCount() != 1 {
		t.Fatalf("OverlayCount = %d, want 1", win.OverlayCount())
	}

	// Remove the overlay (simulating dismiss).
	win.Overlays().Pop()

	if win.OverlayCount() != 0 {
		t.Errorf("OverlayCount = %d after pop, want 0", win.OverlayCount())
	}
	if win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be false after removing all overlays")
	}
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 0 {
		t.Errorf("DirtyOverlayContentRects = %v after pop, want empty", rects)
	}
}

// TestMultipleOverlays_IndependentDamage verifies that with multiple overlays
// stacked (e.g., nested dropdown), damage tracking returns rects for each
// dirty overlay's content independently.
func TestMultipleOverlays_IndependentDamage(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// First overlay: primary dropdown menu.
	menu1 := newOverlayContent(50, 100, 180, 200)
	container1 := overlay.NewContainer(menu1, geometry.Sz(800, 600))
	win.Overlays().Push(container1)

	// Second overlay: submenu.
	menu2 := newOverlayContent(230, 120, 160, 180)
	container2 := overlay.NewContainer(menu2, geometry.Sz(800, 600))
	win.Overlays().Push(container2)

	if win.OverlayCount() != 2 {
		t.Fatalf("OverlayCount = %d, want 2", win.OverlayCount())
	}

	// Both dirty initially.
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 2 {
		t.Fatalf("DirtyOverlayContentRects count = %d, want 2", len(rects))
	}

	// No rect should be full window size.
	for i, r := range rects {
		if r.Width() > 400 || r.Height() > 400 {
			t.Errorf("rect[%d] = %v too large — should be menu area, not full window", i, r)
		}
	}

	// Clear all, then mark only submenu dirty.
	win.ClearOverlayRedraw()
	menu2.SetNeedsRedraw(true)

	rects = win.DirtyOverlayContentRects()
	if len(rects) != 1 {
		t.Fatalf("DirtyOverlayContentRects count = %d after marking only menu2, want 1", len(rects))
	}

	// The rect should match menu2's bounds.
	r := rects[0]
	if r.Width() < 120 || r.Width() > 200 {
		t.Errorf("submenu damage width = %.0f, want ~160", r.Width())
	}
}

// TestOverlayContentRects_FallbackForNonContainer verifies that when an
// overlay does not implement the ContentProvider interface (non-Container
// overlay), DirtyOverlayContentRects falls back to the overlay's own bounds.
func TestOverlayContentRects_FallbackForNonContainer(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Push a raw overlay (not wrapped in Container).
	raw := &rawOverlay{}
	raw.SetVisible(true)
	raw.SetEnabled(true)
	raw.SetBounds(geometry.NewRect(50, 50, 120, 80))
	raw.SetNeedsRedraw(true)
	win.Overlays().Push(raw)

	rects := win.DirtyOverlayContentRects()
	if len(rects) != 1 {
		t.Fatalf("DirtyOverlayContentRects count = %d, want 1 (fallback)", len(rects))
	}

	r := rects[0]
	if r.Width() != 120 || r.Height() != 80 {
		t.Errorf("fallback rect = %v, want (120x80)", r)
	}
}

// TestCleanOverlay_NoDamageRects verifies that clean overlays (content
// not dirty) do not produce damage rects.
func TestCleanOverlay_NoDamageRects(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	content := newOverlayContent(100, 100, 200, 150)
	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Clear all dirty flags.
	win.ClearOverlayRedraw()

	// Verify no damage rects for clean overlay.
	if win.HasDirtyOverlays() {
		t.Error("HasDirtyOverlays should be false after clear")
	}
	rects := win.DirtyOverlayContentRects()
	if len(rects) != 0 {
		t.Errorf("clean overlay DirtyOverlayContentRects = %v, want empty", rects)
	}
}

// TestPushOverlay_SetsRepaintBoundary verifies that the windowOverlayManager
// wraps overlay content in a RepaintBoundary for GPU texture caching (ADR-029).
func TestPushOverlay_SetsRepaintBoundary(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Use overlayManager (via context) to push overlay the same way
	// dropdown/dialog widgets do.
	content := newOverlayContent(100, 100, 200, 150)

	// Direct overlay manager push.
	mgr := &windowOverlayManager{window: win}
	mgr.PushOverlay(content, nil)

	if win.OverlayCount() != 1 {
		t.Fatalf("OverlayCount = %d, want 1", win.OverlayCount())
	}

	// Check that content widget was marked as RepaintBoundary.
	if !content.IsRepaintBoundary() {
		t.Error("overlay content should be marked as RepaintBoundary (ADR-029)")
	}
}

// TestOverlayDamageRect_MatchesContentBoundsExactly verifies that the
// damage rect matches the content widget's Bounds() exactly.
func TestOverlayDamageRect_MatchesContentBoundsExactly(t *testing.T) {
	tests := []struct {
		name  string
		x, y  float32
		w, h  float32
		modal bool
	}{
		{"small_menu", 50, 100, 120, 80, false},
		{"large_dialog", 200, 100, 400, 400, true},
		{"edge_dropdown", 0, 0, 150, 200, false},
		{"bottom_right", 600, 400, 200, 200, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uiApp := New()
			win := uiApp.Window()
			root := newOverlayRoot(geometry.Sz(800, 600))
			win.SetRoot(root)

			content := newOverlayContent(tc.x, tc.y, tc.w, tc.h)
			opts := []overlay.ContainerOption{}
			if tc.modal {
				opts = append(opts, overlay.WithModal(true))
			}
			container := overlay.NewContainer(content, geometry.Sz(800, 600), opts...)
			win.Overlays().Push(container)

			rects := win.DirtyOverlayContentRects()
			if len(rects) != 1 {
				t.Fatalf("DirtyOverlayContentRects count = %d, want 1", len(rects))
			}

			r := rects[0]
			cb := content.Bounds()
			if r.Min.X != cb.Min.X || r.Min.Y != cb.Min.Y ||
				r.Max.X != cb.Max.X || r.Max.Y != cb.Max.Y {
				t.Errorf("damage rect = %v, want content bounds %v", r, cb)
			}
		})
	}
}

// --- Test helpers ---

// rawOverlay is a minimal Overlay implementation for testing fallback behavior
// in DirtyOverlayContentRects (when overlay does not implement ContentProvider).
type rawOverlay struct {
	widget.WidgetBase
}

func (r *rawOverlay) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(r.Bounds().Width(), r.Bounds().Height()))
}
func (r *rawOverlay) Draw(_ widget.Context, _ widget.Canvas) {}
func (r *rawOverlay) Event(_ widget.Context, _ event.Event) bool {
	return false
}
func (r *rawOverlay) Children() []widget.Widget { return nil }
func (r *rawOverlay) Dismiss()                  {}
func (r *rawOverlay) Modal() bool               { return false }
