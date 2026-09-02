package dirty

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// testWidget is a mock widget that embeds WidgetBase for testing the collector.
type testWidget struct {
	widget.WidgetBase
}

func (w *testWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(100, 50))
}

func (w *testWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *testWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func newTestWidget(x, y, w, h float32) *testWidget {
	tw := &testWidget{}
	tw.SetVisible(true)
	tw.SetEnabled(true)
	tw.SetBounds(geometry.NewRect(x, y, w, h))
	tw.SetScreenOrigin(geometry.Pt(x, y))
	return tw
}

// customTestWidget does not embed WidgetBase — no NeedsRedraw method.
type customTestWidget struct {
	bounds   geometry.Rect
	children []widget.Widget
}

func (w *customTestWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(w.bounds.Size())
}

func (w *customTestWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *customTestWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func (w *customTestWidget) Children() []widget.Widget { return w.children }

func (w *customTestWidget) Bounds() geometry.Rect { return w.bounds }

// invisibleWidget is visible=false.
type invisibleWidget struct {
	widget.WidgetBase
}

func (w *invisibleWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(50, 50))
}

func (w *invisibleWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *invisibleWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func TestCollector_NilRoot(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(nil) // should not panic
	if !tr.IsEmpty() {
		t.Error("collecting nil root should produce no dirty regions")
	}
}

func TestCollector_SingleDirtyWidget(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	w := newTestWidget(10, 20, 100, 50)
	w.SetNeedsRedraw(true)

	c.Collect(w)

	if tr.IsEmpty() {
		t.Error("dirty widget should produce a dirty region")
	}
	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1", tr.RegionCount())
	}
	expected := geometry.NewRect(10, 20, 100, 50)
	if tr.DirtyRegions()[0].Bounds != expected {
		t.Errorf("region = %v, want %v", tr.DirtyRegions()[0].Bounds, expected)
	}
}

func TestCollector_SingleCleanWidget(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	w := newTestWidget(10, 20, 100, 50)
	w.ClearRedraw()

	c.Collect(w)

	if !tr.IsEmpty() {
		t.Error("clean widget should produce no dirty regions")
	}
}

func TestCollector_DirtyChild(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	parent := newTestWidget(0, 0, 300, 300)
	parent.ClearRedraw()

	child := newTestWidget(50, 50, 100, 50)
	child.SetNeedsRedraw(true)
	parent.AddChild(child)

	c.Collect(parent)

	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1", tr.RegionCount())
	}
	expected := geometry.NewRect(50, 50, 100, 50)
	if tr.DirtyRegions()[0].Bounds != expected {
		t.Errorf("region = %v, want %v", tr.DirtyRegions()[0].Bounds, expected)
	}
}

func TestCollector_MultipleDirtyWidgets(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	parent := newTestWidget(0, 0, 500, 500)
	parent.ClearRedraw()

	child1 := newTestWidget(10, 10, 50, 50)
	child1.SetNeedsRedraw(true)

	child2 := newTestWidget(200, 200, 50, 50)
	child2.ClearRedraw()

	child3 := newTestWidget(400, 400, 50, 50)
	child3.SetNeedsRedraw(true)

	parent.AddChild(child1)
	parent.AddChild(child2)
	parent.AddChild(child3)

	c.Collect(parent)

	if tr.RegionCount() != 2 {
		t.Errorf("region count = %d, want 2 (two dirty children)", tr.RegionCount())
	}
}

func TestCollector_InvisibleWidgetSkipped(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	parent := newTestWidget(0, 0, 300, 300)
	parent.ClearRedraw()

	inv := &invisibleWidget{}
	inv.SetVisible(false)
	inv.SetBounds(geometry.NewRect(50, 50, 100, 100))
	inv.SetNeedsRedraw(true)
	parent.AddChild(inv)

	c.Collect(parent)

	if !tr.IsEmpty() {
		t.Error("invisible dirty widget should be skipped")
	}
}

func TestCollector_InvisibleSubtreeSkipped(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	root := newTestWidget(0, 0, 500, 500)
	root.ClearRedraw()

	invisParent := &invisibleWidget{}
	invisParent.SetVisible(false)
	invisParent.SetBounds(geometry.NewRect(0, 0, 200, 200))
	invisParent.ClearRedraw()

	dirtyChild := newTestWidget(10, 10, 50, 50)
	dirtyChild.SetNeedsRedraw(true)
	invisParent.AddChild(dirtyChild)

	root.AddChild(invisParent)

	c.Collect(root)

	if !tr.IsEmpty() {
		t.Error("children of invisible widget should be skipped")
	}
}

func TestCollector_CustomWidgetTreatedAsDirty(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	cw := &customTestWidget{
		bounds: geometry.NewRect(10, 10, 80, 80),
	}
	c.Collect(cw)

	if tr.IsEmpty() {
		t.Error("custom widget without NeedsRedraw should be treated as dirty")
	}
	expected := geometry.NewRect(10, 10, 80, 80)
	if tr.DirtyRegions()[0].Bounds != expected {
		t.Errorf("region = %v, want %v", tr.DirtyRegions()[0].Bounds, expected)
	}
}

func TestCollector_CustomWidgetWithChildren(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	child := newTestWidget(50, 50, 30, 30)
	child.SetNeedsRedraw(true)

	cw := &customTestWidget{
		bounds:   geometry.NewRect(0, 0, 200, 200),
		children: []widget.Widget{child},
	}
	c.Collect(cw)

	// Leaf-dirty pattern: custom widget has dirty child → skip self,
	// report only child. Result: 1 region (child only).
	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1 (leaf child only)", tr.RegionCount())
	}
}

func TestCollector_DeeplyNested(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	root := newTestWidget(0, 0, 500, 500)
	root.ClearRedraw()

	mid := newTestWidget(100, 100, 300, 300)
	mid.ClearRedraw()
	root.AddChild(mid)

	leaf := newTestWidget(150, 150, 50, 50)
	leaf.SetNeedsRedraw(true)
	mid.AddChild(leaf)

	c.Collect(root)

	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1", tr.RegionCount())
	}
	expected := geometry.NewRect(150, 150, 50, 50)
	if tr.DirtyRegions()[0].Bounds != expected {
		t.Errorf("region = %v, want %v", tr.DirtyRegions()[0].Bounds, expected)
	}
}

func TestCollector_AllClean(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	root := newTestWidget(0, 0, 500, 500)
	root.ClearRedraw()

	child1 := newTestWidget(10, 10, 50, 50)
	child1.ClearRedraw()
	child2 := newTestWidget(100, 100, 50, 50)
	child2.ClearRedraw()
	root.AddChild(child1)
	root.AddChild(child2)

	c.Collect(root)

	if !tr.IsEmpty() {
		t.Error("all-clean tree should produce no dirty regions")
	}
}

func TestCollector_ParentDirtyChildrenClean(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	parent := newTestWidget(0, 0, 300, 300)
	parent.SetNeedsRedraw(true)

	child := newTestWidget(50, 50, 50, 50)
	child.ClearRedraw()
	parent.AddChild(child)

	c.Collect(parent)

	// Only parent is dirty, child is clean.
	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1", tr.RegionCount())
	}
	expected := geometry.NewRect(0, 0, 300, 300)
	if tr.DirtyRegions()[0].Bounds != expected {
		t.Errorf("region = %v, want %v", tr.DirtyRegions()[0].Bounds, expected)
	}
}

func TestCollector_BothParentAndChildDirty(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	parent := newTestWidget(0, 0, 300, 300)
	parent.SetNeedsRedraw(true)

	child := newTestWidget(50, 50, 50, 50)
	child.SetNeedsRedraw(true)
	parent.AddChild(child)

	c.Collect(parent)

	// Leaf-dirty pattern: parent has dirty child → skip parent, report child only.
	if tr.RegionCount() != 1 {
		t.Errorf("region count = %d, want 1 (leaf child only)", tr.RegionCount())
	}
}

// noBoundsWidget has no Bounds() method and no NeedsRedraw — tests the markWidgetDirty fallback.
type noBoundsWidget struct {
	children []widget.Widget
}

func (w *noBoundsWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(50, 50))
}

func (w *noBoundsWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *noBoundsWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func (w *noBoundsWidget) Children() []widget.Widget { return w.children }

func TestCollector_NoBoundsWidget(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	// Widget without Bounds method — isWidgetDirty returns true (no NeedsRedraw),
	// but markWidgetDirty can't get bounds, so no region added.
	w := &noBoundsWidget{}
	c.Collect(w)

	if !tr.IsEmpty() {
		t.Error("widget without Bounds should not add a region")
	}
}

func TestNewCollector(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}
}

// Integration test: collect + optimize + intersect.
func TestCollectOptimizeIntersect(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	root := newTestWidget(0, 0, 800, 600)
	root.ClearRedraw()

	// Two nearby dirty widgets should merge.
	w1 := newTestWidget(10, 10, 40, 40)
	w1.SetNeedsRedraw(true)
	w2 := newTestWidget(55, 10, 40, 40)
	w2.SetNeedsRedraw(true)
	// One far-away dirty widget.
	w3 := newTestWidget(500, 500, 40, 40)
	w3.SetNeedsRedraw(true)

	root.AddChild(w1)
	root.AddChild(w2)
	root.AddChild(w3)

	c.Collect(root)
	tr.Optimize()

	// With mergeGap=0, only overlapping regions merge. Adjacent (non-overlapping)
	// regions stay separate for precise dirty tracking.
	if tr.RegionCount() != 3 {
		t.Errorf("after optimize: region count = %d, want 3 (no gap merge)", tr.RegionCount())
	}

	// Widget in merged region should intersect.
	if !tr.Intersects(geometry.NewRect(30, 20, 10, 10)) {
		t.Error("should intersect merged region")
	}
	// Widget far away should not intersect.
	if tr.Intersects(geometry.NewRect(300, 300, 10, 10)) {
		t.Error("should not intersect between regions")
	}
}

// --- Viewport clip regression tests (2026-05-07) ---

// viewportWidget implements IsViewportClip() to act as a ScrollView-like container.
type viewportWidget struct {
	widget.WidgetBase
	kids []widget.Widget
}

func newViewportWidget(w, h float32, children ...widget.Widget) *viewportWidget {
	vp := &viewportWidget{kids: children}
	vp.SetVisible(true)
	vp.SetEnabled(true)
	vp.SetBounds(geometry.NewRect(0, 0, w, h))
	vp.SetScreenOrigin(geometry.Pt(0, 0))
	return vp
}

func (w *viewportWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(w.Bounds().Size())
}

func (w *viewportWidget) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *viewportWidget) Event(_ widget.Context, _ event.Event) bool { return false }

func (w *viewportWidget) Children() []widget.Widget { return w.kids }

func (w *viewportWidget) IsViewportClip() bool { return true }

func (w *viewportWidget) ScreenBounds() geometry.Rect {
	return geometry.NewRect(
		w.Bounds().Min.X, w.Bounds().Min.Y,
		w.Bounds().Width(), w.Bounds().Height(),
	)
}

// TestCollectorViewportClipsDirtyRegion verifies that a dirty widget inside
// a viewport container (IsViewportClip=true) has its dirty region clipped to
// the viewport bounds. Before the fix, a widget with 36000px height inside
// a 300px ScrollView would produce a 36000px dirty region, causing the
// entire window to be repainted.
// Regression: widget with bounds 36000px inside ScrollView -> huge dirty region (2026-05-07)
func TestCollectorViewportClipsDirtyRegion(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	// Large dirty content inside a small viewport.
	content := newTestWidget(0, 0, 300, 36000)
	content.SetNeedsRedraw(true)

	viewport := newViewportWidget(300, 300, content)
	viewport.ClearRedraw()

	c.Collect(viewport)

	if tr.IsEmpty() {
		t.Fatal("dirty content inside viewport should produce a dirty region")
	}

	// The dirty region must be clipped to the viewport bounds (300px),
	// not the full content bounds (36000px).
	regions := tr.DirtyRegions()
	for _, r := range regions {
		if r.Bounds.Height() > 300 {
			t.Errorf("dirty region height = %v, want <= 300 (viewport clip); "+
				"Collector must clip dirty regions to viewport bounds",
				r.Bounds.Height())
		}
	}
}

// TestCollectorSkipsCleanViewportChildren verifies that when a viewport
// container and all its children are clean, the Collector reports zero
// dirty regions. This ensures the viewport-specific path does not
// spuriously generate dirty regions.
// Regression: ensures viewport clean path produces 0 regions (2026-05-07)
func TestCollectorSkipsCleanViewportChildren(t *testing.T) {
	tr := NewTracker()
	c := NewCollector(tr)

	// Clean content inside clean viewport.
	content := newTestWidget(0, 0, 300, 1000)
	content.ClearRedraw()

	viewport := newViewportWidget(300, 300, content)
	viewport.ClearRedraw()

	c.Collect(viewport)

	if !tr.IsEmpty() {
		t.Errorf("all-clean viewport should produce 0 dirty regions, got %d",
			tr.RegionCount())
	}
}

// --- Leaf Dirty Region Tests (ADR-024 RepaintBoundary integration) ---
//
// When a child widget is dirty (e.g., checkbox hover), propagateDirtyUpward
// marks all ancestors dirty too. The Collector must report LEAF dirty widget
// bounds (small rect), NOT parent container bounds (full card).
//
// Without this, cyan overlay shows full-window dirty on every hover →
// appears as "always full repaint" when only a small widget changed.

// TestCollector_LeafDirtyNotParent verifies that when a child is dirty
// AND its parent is dirty (via propagation), only the CHILD's bounds
// are reported — not the parent's large bounds.
func TestCollector_LeafDirtyNotParent(t *testing.T) {
	// Parent: large card (0,0 → 400,300).
	parent := newTestWidget(0, 0, 400, 300)

	// Child: small checkbox (10,10 → 200,36).
	child := newTestWidget(10, 10, 200, 36)
	child.SetParent(parent)
	parent.AddChild(child)

	// Simulate propagateDirtyUpward: child dirty → parent dirty.
	child.SetNeedsRedraw(true)
	parent.SetNeedsRedraw(true) // marked by propagation

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(parent)

	regions := tr.DirtyRegions()

	// We should get child bounds (small), NOT parent bounds (large).
	// If parent bounds are reported, it means the overlay will show
	// full card cyan on every checkbox hover.
	for _, r := range regions {
		if r.Bounds.Width() > 250 || r.Bounds.Height() > 100 {
			t.Errorf("dirty region too large: %v — should be child bounds (~200x36), "+
				"not parent bounds (~400x300). Collector reports parent container "+
				"instead of leaf dirty widget.", r.Bounds)
		}
	}

	// Must have at least one region (the child).
	if len(regions) == 0 {
		t.Error("expected at least 1 dirty region for the dirty child")
	}
}

// TestCollector_OnlyLeafDirtyReported verifies that when parent is dirty
// ONLY because of propagation (has dirty children), the parent's own bounds
// are NOT added — only leaf dirty children are reported.
func TestCollector_OnlyLeafDirtyReported(t *testing.T) {
	parent := newTestWidget(0, 0, 800, 600)

	child1 := newTestWidget(10, 10, 100, 30) // dirty
	child1.SetParent(parent)
	parent.AddChild(child1)

	child2 := newTestWidget(10, 50, 100, 30) // clean
	child2.SetParent(parent)
	parent.AddChild(child2)

	child1.SetNeedsRedraw(true)
	parent.SetNeedsRedraw(true) // propagation artifact

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(parent)

	regions := tr.DirtyRegions()

	// Should have exactly 1 region: child1 bounds.
	// Parent should NOT be reported (it has dirty children → skip self).
	// child2 should NOT be reported (it's clean).
	foundChild1 := false
	foundParent := false
	for _, r := range regions {
		if r.Bounds.Width() >= 700 {
			foundParent = true
		}
		if r.Bounds.Width() <= 150 && r.Bounds.Height() <= 50 {
			foundChild1 = true
		}
	}

	if foundParent {
		t.Error("parent bounds (800x600) should NOT be reported when it has dirty children; " +
			"Collector should skip parent and report only leaf dirty widgets")
	}
	if !foundChild1 {
		t.Error("child1 bounds should be reported as dirty region")
	}
}

// TestCollector_DeepNestingLeafDirty verifies that leaf-dirty pattern
// works through deeply nested containers (taskmanager/gallery pattern:
// chart inside collapsible inside card inside ScrollView).
func TestCollector_DeepNestingLeafDirty(t *testing.T) {
	// Simulate: root → card → section → chart (dirty)
	root := newTestWidget(0, 0, 800, 600)
	card := newTestWidget(24, 24, 736, 500)
	card.SetParent(root)
	root.AddChild(card)

	section := newTestWidget(32, 100, 672, 200)
	section.SetParent(card)
	card.AddChild(section)

	chart := newTestWidget(32, 120, 640, 160)
	chart.SetParent(section)
	section.AddChild(chart)

	// Only chart dirty (PushValue → SetNeedsRedraw → propagation)
	chart.SetNeedsRedraw(true)
	// propagation marks ancestors: section, card, root

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(root)

	regions := tr.DirtyRegions()

	// Should find chart bounds (~640x160), NOT root/card/section bounds
	foundLeaf := false
	foundLarge := false
	for _, r := range regions {
		if r.Bounds.Width() <= 650 && r.Bounds.Height() <= 170 {
			foundLeaf = true
		}
		if r.Bounds.Width() > 700 {
			foundLarge = true
		}
	}

	if !foundLeaf {
		t.Error("chart leaf bounds NOT found in dirty regions; " +
			"Collector doesn't recurse deep enough through leaf-dirty pattern")
	}
	if foundLarge {
		t.Error("parent container bounds found — leaf-dirty not working for deep nesting")
	}
}

// TestCollector_GalleryPattern_ScrollViewWithSections verifies leaf-dirty
// for gallery pattern: ScrollView(viewport) → VBox → sections → leaf widget.
func TestCollector_GalleryPattern_ScrollViewWithSections(t *testing.T) {
	chart := newTestWidget(32, 340, 640, 160)

	section1 := newTestWidget(24, 0, 720, 300)
	section2 := newTestWidget(24, 320, 720, 200)
	section2.AddChild(chart)
	chart.SetParent(section2)

	vbox := newTestWidget(0, 0, 760, 2000)
	vbox.AddChild(section1)
	vbox.AddChild(section2)
	section1.SetParent(vbox)
	section2.SetParent(vbox)

	scrollView := newViewportWidget(800, 600, vbox)
	vbox.SetParent(scrollView)

	chart.SetNeedsRedraw(true)

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(scrollView)

	regions := tr.DirtyRegions()

	foundChart := false
	foundLarge := false
	for _, r := range regions {
		w := r.Bounds.Width()
		h := r.Bounds.Height()
		if w <= 650 && h <= 170 {
			foundChart = true
		}
		if w > 700 && h > 300 {
			foundLarge = true
		}
	}

	if !foundChart {
		t.Errorf("chart leaf bounds NOT found; regions=%v", regions)
	}
	if foundLarge {
		t.Errorf("large container bounds found — leaf-dirty not working through viewport; regions=%v", regions)
	}
}

// TestCollector_TaskmanagerPattern_ChartInCollapsible verifies leaf-dirty
// for taskmanager pattern: ScrollView → VBox → Collapsible → chart.
// Chart updates via PushValue → SetNeedsRedraw → only chart bounds reported.
func TestCollector_TaskmanagerPattern_ChartInCollapsible(t *testing.T) {
	cpuChart := newTestWidget(12, 40, 660, 200)

	collapsibleCPU := newTestWidget(0, 0, 700, 350)
	collapsibleCPU.AddChild(cpuChart)
	cpuChart.SetParent(collapsibleCPU)

	collapsibleMem := newTestWidget(0, 370, 700, 250)

	vbox := newTestWidget(0, 0, 700, 1200)
	vbox.AddChild(collapsibleCPU)
	vbox.AddChild(collapsibleMem)
	collapsibleCPU.SetParent(vbox)
	collapsibleMem.SetParent(vbox)

	scrollView := newViewportWidget(700, 800, vbox)
	vbox.SetParent(scrollView)

	cpuChart.SetNeedsRedraw(true)

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(scrollView)

	regions := tr.DirtyRegions()

	foundChart := false
	foundLarge := false
	for _, r := range regions {
		w := r.Bounds.Width()
		h := r.Bounds.Height()
		if w <= 670 && h <= 210 {
			foundChart = true
		}
		if w > 690 || h > 400 {
			foundLarge = true
		}
	}

	if !foundChart {
		t.Errorf("chart bounds NOT found; regions=%v", regions)
	}
	if foundLarge {
		t.Errorf("large container bounds found; regions=%v", regions)
	}
}

// TestCollector_NoDirtyChildren_ReportSelf verifies that when a widget is
// dirty but has NO dirty children, it reports its own bounds.
func TestCollector_NoDirtyChildren_ReportSelf(t *testing.T) {
	parent := newTestWidget(0, 0, 800, 600)

	child1 := newTestWidget(10, 10, 100, 30)
	child1.SetParent(parent)
	parent.AddChild(child1)

	// Only parent dirty, children clean (e.g., theme change).
	parent.SetNeedsRedraw(true)
	child1.ClearRedraw()

	tr := NewTracker()
	c := NewCollector(tr)
	c.Collect(parent)

	regions := tr.DirtyRegions()
	if len(regions) == 0 {
		t.Error("expected parent bounds as dirty region (no dirty children)")
	}
}
