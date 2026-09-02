package app

import (
	"testing"

	"github.com/gogpu/ui/compositor"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/overlay"
	"github.com/gogpu/ui/widget"
)

// --- Overlay Layer Tree Integration Tests (ADR-029 Phase E) ---

// TestOverlayInLayerTree verifies that when an overlay is pushed, its content
// boundary widget appears in the Layer Tree after AppendOverlaysToLayerTree.
func TestOverlayInLayerTree(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	// Push an overlay with content that has a RepaintBoundary.
	content := &testLeaf{}
	content.SetVisible(true)
	content.SetRepaintBoundary(true)
	content.SetBounds(geometry.NewRect(100, 200, 300, 350))
	content.SetScreenOrigin(geometry.Pt(100, 200))

	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Build layer tree from root only.
	layerTree := BuildLayerTree(root)

	// Before appending overlays: only root boundary.
	var picsBefore []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(layerTree, &picsBefore)
	if len(picsBefore) != 1 {
		t.Fatalf("before AppendOverlays: expected 1 PictureLayer (root), got %d", len(picsBefore))
	}

	// Append overlay content to tree.
	overlayWidgets := win.OverlayContentWidgets()
	if len(overlayWidgets) != 1 {
		t.Fatalf("OverlayContentWidgets count = %d, want 1", len(overlayWidgets))
	}
	AppendOverlaysToLayerTree(layerTree, overlayWidgets, nil)

	// After appending: root + overlay content = 2 PictureLayers.
	var picsAfter []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(layerTree, &picsAfter)
	if len(picsAfter) != 2 {
		t.Fatalf("after AppendOverlays: expected 2 PictureLayers, got %d", len(picsAfter))
	}

	// Verify overlay content's cache key is in the tree.
	contentKey := content.BoundaryCacheKey()
	found := false
	for _, pic := range picsAfter {
		if pic.BoundaryCacheKey() == contentKey {
			found = true
			break
		}
	}
	if !found {
		t.Error("overlay content boundary not found in Layer Tree after AppendOverlaysToLayerTree")
	}
}

// TestOverlayContentWidgets_ReturnsContentNotContainer verifies that
// OverlayContentWidgets returns the inner content widget, not the Container.
func TestOverlayContentWidgets_ReturnsContentNotContainer(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	content := newOverlayContent(100, 200, 200, 150)
	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	widgets := win.OverlayContentWidgets()
	if len(widgets) != 1 {
		t.Fatalf("OverlayContentWidgets count = %d, want 1", len(widgets))
	}

	// Should be the content widget, not the Container.
	if widgets[0] != content {
		t.Error("OverlayContentWidgets should return content widget, not Container")
	}
}

// TestOverlayContentWidgets_Empty verifies empty result when no overlays.
func TestOverlayContentWidgets_Empty(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	widgets := win.OverlayContentWidgets()
	if len(widgets) != 0 {
		t.Errorf("OverlayContentWidgets with no overlays = %d, want 0", len(widgets))
	}
}

// TestOverlayContentWidgets_MultipleOverlays verifies correct widget extraction
// from multiple stacked overlays.
func TestOverlayContentWidgets_MultipleOverlays(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	content1 := newOverlayContent(50, 100, 180, 200)
	container1 := overlay.NewContainer(content1, geometry.Sz(800, 600))
	win.Overlays().Push(container1)

	content2 := newOverlayContent(230, 120, 160, 180)
	container2 := overlay.NewContainer(content2, geometry.Sz(800, 600))
	win.Overlays().Push(container2)

	widgets := win.OverlayContentWidgets()
	if len(widgets) != 2 {
		t.Fatalf("OverlayContentWidgets count = %d, want 2", len(widgets))
	}

	if widgets[0] != content1 {
		t.Error("first overlay content should be content1")
	}
	if widgets[1] != content2 {
		t.Error("second overlay content should be content2")
	}
}

// TestOverlayDismiss_RemovedFromLayerTree verifies that after removing an overlay,
// its boundary no longer appears in the Layer Tree.
func TestOverlayDismiss_RemovedFromLayerTree(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	root.SetScreenOrigin(geometry.Pt(0, 0))
	win.SetRoot(root)

	// Push overlay.
	content := &testLeaf{}
	content.SetVisible(true)
	content.SetRepaintBoundary(true)
	content.SetBounds(geometry.NewRect(100, 200, 300, 350))
	content.SetScreenOrigin(geometry.Pt(100, 200))
	contentKey := content.BoundaryCacheKey()

	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Build tree with overlay.
	layerTree := BuildLayerTree(root)
	overlayWidgets := win.OverlayContentWidgets()
	AppendOverlaysToLayerTree(layerTree, overlayWidgets, nil)

	var picsWith []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(layerTree, &picsWith)
	if len(picsWith) != 2 {
		t.Fatalf("with overlay: expected 2 PictureLayers, got %d", len(picsWith))
	}

	// Dismiss overlay.
	win.Overlays().Pop()

	// Rebuild tree without overlay.
	layerTree2 := BuildLayerTree(root)
	overlayWidgets2 := win.OverlayContentWidgets()
	AppendOverlaysToLayerTree(layerTree2, overlayWidgets2, nil)

	var picsWithout []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(layerTree2, &picsWithout)
	if len(picsWithout) != 1 {
		t.Fatalf("without overlay: expected 1 PictureLayer (root only), got %d", len(picsWithout))
	}

	// Verify overlay content key is NOT present.
	for _, pic := range picsWithout {
		if pic.BoundaryCacheKey() == contentKey {
			t.Error("dismissed overlay's boundary should not appear in Layer Tree")
		}
	}
}

// TestOverlayOnTopOfContent_ZOrder verifies that overlay PictureLayers
// appear AFTER main tree PictureLayers in the Layer Tree children order,
// ensuring correct Z-order (main content → overlays on top).
func TestOverlayOnTopOfContent_ZOrder(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	root.SetScreenOrigin(geometry.Pt(0, 0))

	// Main tree child boundary.
	mainChild := &testLeaf{}
	mainChild.SetVisible(true)
	mainChild.SetRepaintBoundary(true)
	mainChild.SetBounds(geometry.NewRect(10, 10, 58, 58))
	mainChild.SetScreenOrigin(geometry.Pt(10, 10))
	mainChild.SetParent(root)
	root.kids = []widget.Widget{mainChild}
	win.SetRoot(root)

	// Overlay content boundary.
	overlayContent := &testLeaf{}
	overlayContent.SetVisible(true)
	overlayContent.SetRepaintBoundary(true)
	overlayContent.SetBounds(geometry.NewRect(100, 200, 300, 400))
	overlayContent.SetScreenOrigin(geometry.Pt(100, 200))

	container := overlay.NewContainer(overlayContent, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// Build tree and append overlays.
	layerTree := BuildLayerTree(root)
	overlayWidgets := win.OverlayContentWidgets()
	AppendOverlaysToLayerTree(layerTree, overlayWidgets, nil)

	// Collect all PictureLayers in tree traversal order.
	var pics []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(layerTree, &pics)

	if len(pics) != 3 {
		t.Fatalf("expected 3 PictureLayers (root + mainChild + overlay), got %d", len(pics))
	}

	// Root should be first (or early), overlay should be last.
	overlayKey := overlayContent.BoundaryCacheKey()
	lastPic := pics[len(pics)-1]
	if lastPic.BoundaryCacheKey() != overlayKey {
		t.Error("overlay content PictureLayer should be last in tree (Z-order: on top)")
	}
}

// TestPaintOverlayBoundaries_RecordsDirty verifies that PaintOverlayBoundaries
// re-records dirty overlay content boundaries.
func TestPaintOverlayBoundaries_RecordsDirty(t *testing.T) {
	prev := widget.GetSceneRecorderFactory()
	widget.RegisterSceneRecorder(testSceneRecorder)
	defer widget.RegisterSceneRecorder(prev)

	content := &testLeaf{}
	content.SetVisible(true)
	content.SetRepaintBoundary(true)
	content.SetBounds(geometry.NewRect(100, 200, 300, 350))
	content.SetScreenOrigin(geometry.Pt(100, 200))
	content.InvalidateScene()

	ctx := widget.NewContext()
	ctx.SetOnInvalidateRect(func(_ geometry.Rect) {})

	PaintOverlayBoundaries([]widget.Widget{content}, ctx)

	if content.CachedScene() == nil {
		t.Error("dirty overlay content should have CachedScene after PaintOverlayBoundaries")
	}
	if content.drawCount == 0 {
		t.Error("dirty overlay content.Draw should be called during recording")
	}
}

// TestPaintOverlayBoundaries_SkipsClean verifies that PaintOverlayBoundaries
// does not re-record clean overlay boundaries.
func TestPaintOverlayBoundaries_SkipsClean(t *testing.T) {
	prev := widget.GetSceneRecorderFactory()
	widget.RegisterSceneRecorder(testSceneRecorder)
	defer widget.RegisterSceneRecorder(prev)

	content := &testLeaf{}
	content.SetVisible(true)
	content.SetRepaintBoundary(true)
	content.SetBounds(geometry.NewRect(100, 200, 300, 350))
	content.SetScreenOrigin(geometry.Pt(100, 200))

	ctx := widget.NewContext()
	ctx.SetOnInvalidateRect(func(_ geometry.Rect) {})

	// First paint: records dirty boundary.
	content.InvalidateScene()
	PaintOverlayBoundaries([]widget.Widget{content}, ctx)
	firstDrawCount := content.drawCount

	// Second paint: boundary is clean (ClearSceneDirty called by recordBoundary).
	PaintOverlayBoundaries([]widget.Widget{content}, ctx)

	if content.drawCount != firstDrawCount {
		t.Errorf("clean overlay content should NOT be re-recorded: drawCount %d -> %d",
			firstDrawCount, content.drawCount)
	}
}

// TestAppendOverlaysToLayerTree_NilTree verifies safety with nil tree.
func TestAppendOverlaysToLayerTree_NilTree(t *testing.T) {
	content := &testLeaf{}
	content.SetVisible(true)
	content.SetRepaintBoundary(true)
	content.SetBounds(geometry.NewRect(0, 0, 100, 100))

	// Should not panic.
	AppendOverlaysToLayerTree(nil, []widget.Widget{content}, nil)
}

// TestAppendOverlaysToLayerTree_EmptyOverlays verifies no change with empty overlay list.
func TestAppendOverlaysToLayerTree_EmptyOverlays(t *testing.T) {
	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	tree := BuildLayerTree(root)

	var picsBefore []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(tree, &picsBefore)

	AppendOverlaysToLayerTree(tree, nil, nil)

	var picsAfter []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(tree, &picsAfter)

	if len(picsAfter) != len(picsBefore) {
		t.Errorf("empty overlay list changed tree: %d -> %d PictureLayers",
			len(picsBefore), len(picsAfter))
	}
}

// TestAppendOverlaysToLayerTree_NonBoundaryOverlaySkipped verifies that
// overlay content widgets that are NOT RepaintBoundary are skipped.
func TestAppendOverlaysToLayerTree_NonBoundaryOverlaySkipped(t *testing.T) {
	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	tree := BuildLayerTree(root)

	// Non-boundary widget.
	nonBoundary := &testLeaf{}
	nonBoundary.SetVisible(true)
	nonBoundary.SetBounds(geometry.NewRect(0, 0, 100, 100))

	AppendOverlaysToLayerTree(tree, []widget.Widget{nonBoundary}, nil)

	var pics []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(tree, &pics)

	// Only root boundary should exist (non-boundary overlay skipped).
	if len(pics) != 1 {
		t.Errorf("non-boundary overlay should be skipped: expected 1 PictureLayer, got %d", len(pics))
	}
}

// TestAppendOverlaysToLayerTree_OverlayNotRoot verifies that overlay
// PictureLayers are NOT marked as root. Without this fix, overlay boundaries
// with Parent()==nil are falsely detected as root, causing DrawGPUTextureBase
// (QueueBaseLayer, last-call-wins) to overwrite the actual root texture
// with the overlay texture → black background behind dropdown menus.
func TestAppendOverlaysToLayerTree_OverlayNotRoot(t *testing.T) {
	// Build main tree with root boundary.
	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	tree := BuildLayerTree(root)

	// Create overlay content widget (standalone, Parent()==nil — like dropdown menu).
	overlayContent := &testContainer{}
	overlayContent.SetVisible(true)
	overlayContent.SetRepaintBoundary(true)
	overlayContent.SetBounds(geometry.NewRect(100, 300, 300, 450))
	overlayContent.SetScreenOrigin(geometry.Pt(100, 300))

	AppendOverlaysToLayerTree(tree, []widget.Widget{overlayContent}, nil)

	var pics []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(tree, &pics)

	if len(pics) != 2 {
		t.Fatalf("expected 2 PictureLayers (root + overlay), got %d", len(pics))
	}

	// Root boundary must be root.
	if !pics[0].IsRoot() {
		t.Error("first PictureLayer (root) should have IsRoot=true")
	}
	// Overlay boundary must NOT be root.
	if pics[1].IsRoot() {
		t.Error("overlay PictureLayer should have IsRoot=false (was incorrectly set to true because Parent()==nil)")
	}
}

// TestAppendOverlaysToLayerTree_OverlayNotRoot_Reused verifies that overlay
// PictureLayers remain non-root when the layer tree is reused across frames
// (the existing parameter is non-nil, triggering updateBoundaryLayer which
// calls syncPictureLayer → SetRoot(Parent()==nil) → true). The
// clearRootOnPictureLayers pass must fix this on every frame.
func TestAppendOverlaysToLayerTree_OverlayNotRoot_Reused(t *testing.T) {
	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	// Frame 1: build fresh tree + append overlay.
	tree1 := BuildLayerTree(root)
	overlayContent := &testContainer{}
	overlayContent.SetVisible(true)
	overlayContent.SetRepaintBoundary(true)
	overlayContent.SetBounds(geometry.NewRect(100, 300, 300, 450))
	overlayContent.SetScreenOrigin(geometry.Pt(100, 300))
	AppendOverlaysToLayerTree(tree1, []widget.Widget{overlayContent}, nil)

	// Frame 2: reuse existing tree (simulates UpdateLayerTree + append).
	tree2 := UpdateLayerTree(root, tree1)
	AppendOverlaysToLayerTree(tree2, []widget.Widget{overlayContent}, tree1)

	var pics []*compositor.PictureLayerImpl
	collectPictureLayersFromTree(tree2, &pics)

	if len(pics) != 2 {
		t.Fatalf("expected 2 PictureLayers on frame 2, got %d", len(pics))
	}
	if !pics[0].IsRoot() {
		t.Error("root PictureLayer should remain IsRoot=true on reused tree")
	}
	if pics[1].IsRoot() {
		t.Error("overlay PictureLayer should remain IsRoot=false on reused tree (syncPictureLayer resets it)")
	}
}

// TestDrawOverlayScrim_NoOverlays verifies no panic when no overlays.
func TestDrawOverlayScrim_NoOverlays(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Should not panic with nil canvas.
	win.DrawOverlayScrim(nil)
}

// TestDrawOverlayScrim_ModalOnlyBehavior verifies that DrawOverlayScrim
// checks modality correctly: non-modal overlays produce no scrim.
func TestDrawOverlayScrim_ModalOnlyBehavior(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	// Non-modal overlay (dropdown) — no scrim expected.
	content := newOverlayContent(100, 200, 200, 150)
	container := overlay.NewContainer(content, geometry.Sz(800, 600))
	win.Overlays().Push(container)

	// DrawOverlayScrim with nil canvas should not panic.
	// (No modal overlay to trigger DrawRect, so nil canvas is safe.)
	win.DrawOverlayScrim(nil)

	// Verify that the overlay is non-modal.
	if container.Modal() {
		t.Error("dropdown container should not be modal")
	}
}

// TestOverlayContentWidgets_FallbackForNonContainer verifies that non-Container
// overlays return themselves from OverlayContentWidgets.
func TestOverlayContentWidgets_FallbackForNonContainer(t *testing.T) {
	uiApp := New()
	win := uiApp.Window()

	root := newOverlayRoot(geometry.Sz(800, 600))
	win.SetRoot(root)

	raw := &rawOverlay{}
	raw.SetVisible(true)
	raw.SetEnabled(true)
	raw.SetBounds(geometry.NewRect(50, 50, 120, 80))
	win.Overlays().Push(raw)

	widgets := win.OverlayContentWidgets()
	if len(widgets) != 1 {
		t.Fatalf("OverlayContentWidgets count = %d, want 1", len(widgets))
	}

	// rawOverlay has no Content() method, so it should be returned directly.
	if widgets[0] != raw {
		t.Error("non-Container overlay should be returned as-is from OverlayContentWidgets")
	}
}
