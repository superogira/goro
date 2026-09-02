package desktop

import (
	"testing"

	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/compositor"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- Compositor Clip Tests (Layer Tree) ---
//
// These tests verify that the Layer Tree pipeline respects CompositorClip —
// skipping boundary textures outside the viewport. ADR-007 Phase D replaced
// the widget tree walkBoundaries with Layer Tree walk.

// TestCompositorClip_SkipsItemsOutsideClip verifies that BuildLayerTree
// produces PictureLayers with correct clip data, and renderSingleBoundaryFromLayer
// skips items whose screen rect doesn't intersect their CompositorClip.
func TestCompositorClip_SkipsItemsOutsideClip(t *testing.T) {
	viewportClip := geometry.NewRect(0, 200, 800, 300)
	t.Logf("viewport clip: %v (Min=%v Max=%v)", viewportClip, viewportClip.Min, viewportClip.Max)

	root := &ccTestContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	type itemSpec struct {
		screenY float32
		wantVis bool
	}
	specs := []itemSpec{
		{screenY: 100, wantVis: false}, // fully above clip
		{screenY: 190, wantVis: true},  // partially above
		{screenY: 300, wantVis: true},  // fully inside
		{screenY: 480, wantVis: true},  // partially below
		{screenY: 510, wantVis: false}, // fully below
	}

	items := make([]*ccTestItem, len(specs))
	for i, s := range specs {
		items[i] = &ccTestItem{index: i}
		items[i].SetVisible(true)
		items[i].SetRepaintBoundary(true)
		items[i].SetBounds(geometry.NewRect(0, 0, 200, 40))
		items[i].SetScreenOrigin(geometry.Pt(10, s.screenY))
		items[i].SetCompositorClip(viewportClip)
		items[i].SetParent(root)
	}

	children := make([]widget.Widget, len(items))
	for i, item := range items {
		children[i] = item
	}
	root.children = children

	// Build Layer Tree — PictureLayers carry clip data from widgets.
	layerTree := app.BuildLayerTree(root)

	// Collect PictureLayers from the Layer Tree (excluding root).
	var pictureLayers []*compositor.PictureLayerImpl
	collectPictureLayers(layerTree, &pictureLayers, false)

	// Verify correct number of child boundaries in tree.
	if len(pictureLayers) != len(items) {
		t.Fatalf("Layer Tree has %d child PictureLayers, want %d", len(pictureLayers), len(items))
	}

	// Verify each PictureLayer's clip filtering matches expected visibility.
	for i, pic := range pictureLayers {
		visible := isPictureLayerVisible(pic)
		if visible != specs[i].wantVis {
			t.Errorf("item[%d] visible=%v, want %v (screenY=%v clip=%v)",
				i, visible, specs[i].wantVis, specs[i].screenY, viewportClip)
		}
	}
}

// TestCompositorClip_NoClipShowsAll verifies backward compatibility:
// boundaries without CompositorClip are always composited.
func TestCompositorClip_NoClipShowsAll(t *testing.T) {
	root := &ccTestContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	item0 := &ccTestItem{index: 0}
	item0.SetVisible(true)
	item0.SetRepaintBoundary(true)
	item0.SetBounds(geometry.NewRect(0, 0, 200, 40))
	item0.SetScreenOrigin(geometry.Pt(10, 100))
	item0.SetParent(root)

	item1 := &ccTestItem{index: 1}
	item1.SetVisible(true)
	item1.SetRepaintBoundary(true)
	item1.SetBounds(geometry.NewRect(0, 0, 200, 40))
	item1.SetScreenOrigin(geometry.Pt(10, 700))
	item1.SetParent(root)

	root.children = []widget.Widget{item0, item1}

	layerTree := app.BuildLayerTree(root)

	var pictureLayers []*compositor.PictureLayerImpl
	collectPictureLayers(layerTree, &pictureLayers, false)

	visibleCount := 0
	for _, pic := range pictureLayers {
		if isPictureLayerVisible(pic) {
			visibleCount++
		}
	}

	if visibleCount != 2 {
		t.Errorf("without CompositorClip, all items should be visible: got %d, want 2", visibleCount)
	}
}

// TestCompositorClip_RootNeverClipped verifies that the root PictureLayer
// (IsRoot=true) is never affected by compositor clip.
func TestCompositorClip_RootNeverClipped(t *testing.T) {
	root := &ccTestContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	root.SetCompositorClip(geometry.NewRect(0, 0, 1, 1)) // tiny clip

	layerTree := app.BuildLayerTree(root)

	// Find root PictureLayer.
	var rootPic *compositor.PictureLayerImpl
	collectPictureLayers(layerTree, nil, false) // just to verify structure
	walkLayerTree(layerTree, func(layer compositor.Layer) {
		if pic, ok := layer.(*compositor.PictureLayerImpl); ok && pic.IsRoot() {
			rootPic = pic
		}
	})

	if rootPic == nil {
		t.Fatal("root PictureLayer not found in Layer Tree")
	}

	// Root is always visible regardless of clip.
	if !isPictureLayerVisible(rootPic) {
		t.Error("root boundary should never be clipped (IsRoot=true)")
	}
}

// --- Layer Tree test helpers ---

// collectPictureLayers walks the Layer Tree and collects non-root PictureLayers.
// If out is nil, it only walks (useful for testing walkability).
func collectPictureLayers(layer compositor.Layer, out *[]*compositor.PictureLayerImpl, includeRoot bool) {
	if layer == nil {
		return
	}
	if pic, ok := layer.(*compositor.PictureLayerImpl); ok {
		if out != nil && (includeRoot || !pic.IsRoot()) {
			*out = append(*out, pic)
		}
		return
	}
	if cl, ok := layer.(compositor.ContainerLayer); ok {
		for _, child := range cl.Children() {
			collectPictureLayers(child, out, includeRoot)
		}
	}
}

// walkLayerTree calls fn for every layer in the tree.
func walkLayerTree(layer compositor.Layer, fn func(compositor.Layer)) {
	if layer == nil {
		return
	}
	fn(layer)
	if cl, ok := layer.(compositor.ContainerLayer); ok {
		for _, child := range cl.Children() {
			walkLayerTree(child, fn)
		}
	}
}

// isPictureLayerVisible applies the same visibility rules as
// renderSingleBoundaryFromLayer: root is always visible, non-root
// checks ScreenOrigin validity and CompositorClip intersection.
func isPictureLayerVisible(pic *compositor.PictureLayerImpl) bool {
	if pic.IsRoot() {
		return true
	}
	if !pic.IsScreenOriginValid() {
		return false
	}
	if !pic.HasPictureClip() {
		return true
	}
	clip := pic.PictureClipRect()
	origin := pic.ScreenOrigin()
	bw, bh := pic.Size()
	screenRect := geometry.Rect{
		Min: origin,
		Max: geometry.Pt(origin.X+float32(bw), origin.Y+float32(bh)),
	}
	return screenRect.Intersects(clip)
}

// --- test widgets ---

type ccTestItem struct {
	widget.WidgetBase
	index int
}

func (w *ccTestItem) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(200, 40))
}

func (w *ccTestItem) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *ccTestItem) Event(_ widget.Context, _ event.Event) bool { return false }

func (w *ccTestItem) Children() []widget.Widget { return nil }

type ccTestContainer struct {
	widget.WidgetBase
	children []widget.Widget
}

func (w *ccTestContainer) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(800, 600))
}

func (w *ccTestContainer) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *ccTestContainer) Event(_ widget.Context, _ event.Event) bool { return false }

func (w *ccTestContainer) Children() []widget.Widget { return w.children }
