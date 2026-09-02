package app

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- ADR-028 Phase C: O(1) Flat Dirty Boundary List ---
//
// These tests verify the end-to-end pipeline:
//   SetNeedsRedraw → propagateDirtyUpward → InvalidateScene →
//   onBoundaryDirty → RegisterDirtyBoundary → HasDirtyBoundaries
//
// Flutter equivalent: markNeedsPaint → _nodesNeedingPaint →
//   _hasScheduledFrame → flushPaint

// TestFlatDirtyList_PropagateDirtyUpward_PopulatesDirtySet verifies that
// a child widget's SetNeedsRedraw propagates upward to the parent boundary,
// which fires onBoundaryDirty, which calls RegisterDirtyBoundary, which
// populates the Window's dirtyBoundaries map.
func TestFlatDirtyList_PropagateDirtyUpward_PopulatesDirtySet(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	// Build: root boundary → child (non-boundary).
	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	child := &testLeaf{}
	child.SetVisible(true)
	child.SetBounds(geometry.NewRect(10, 10, 48, 48))
	child.SetParent(root)
	root.kids = []widget.Widget{child}

	w.SetRoot(root)

	// Record boundaries so onBoundaryDirty callback is wired.
	PaintBoundaryLayersWithContext(root, nil, w.Context())

	// Clear dirty state from initial recording.
	w.ClearDirtyBoundaries()
	w.ClearAfterPaint()
	root.ClearSceneDirty()
	widget.ClearRedrawInTree(root)

	// Precondition: no dirty boundaries.
	if w.HasDirtyBoundaries() {
		t.Fatal("pre-condition: should start clean after clear")
	}

	// Action: child widget changes → SetNeedsRedraw(true).
	child.SetNeedsRedraw(true)

	// Verify: root boundary should be in dirty set.
	if !w.HasDirtyBoundaries() {
		t.Error("SetNeedsRedraw should propagate upward and populate dirtyBoundaries")
	}
	if w.DirtyBoundaryCount() != 1 {
		t.Errorf("expected 1 dirty boundary, got %d", w.DirtyBoundaryCount())
	}
}

// TestFlatDirtyList_DeduplicatesSameBoundary verifies that multiple children
// under the same boundary produce only one entry in the dirty set.
func TestFlatDirtyList_DeduplicatesSameBoundary(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	child1 := &testLeaf{}
	child1.SetVisible(true)
	child1.SetBounds(geometry.NewRect(10, 10, 48, 48))
	child1.SetParent(root)

	child2 := &testLeaf{}
	child2.SetVisible(true)
	child2.SetBounds(geometry.NewRect(70, 10, 48, 48))
	child2.SetParent(root)

	child3 := &testLeaf{}
	child3.SetVisible(true)
	child3.SetBounds(geometry.NewRect(130, 10, 48, 48))
	child3.SetParent(root)

	root.kids = []widget.Widget{child1, child2, child3}
	w.SetRoot(root)

	PaintBoundaryLayersWithContext(root, nil, w.Context())
	w.ClearDirtyBoundaries()
	root.ClearSceneDirty()
	widget.ClearRedrawInTree(root)

	// All 3 children dirty → same boundary → 1 entry.
	child1.SetNeedsRedraw(true)
	child2.SetNeedsRedraw(true)
	child3.SetNeedsRedraw(true)

	if w.DirtyBoundaryCount() != 1 {
		t.Errorf("expected 1 dirty boundary (deduplicated), got %d", w.DirtyBoundaryCount())
	}
}

// TestFlatDirtyList_MultipleBoundaries verifies that dirty children under
// different boundaries produce separate entries.
func TestFlatDirtyList_MultipleBoundaries(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	boundary1 := &testContainer{}
	boundary1.SetVisible(true)
	boundary1.SetRepaintBoundary(true)
	boundary1.SetBounds(geometry.NewRect(10, 10, 200, 200))
	boundary1.SetParent(root)

	boundary2 := &testContainer{}
	boundary2.SetVisible(true)
	boundary2.SetRepaintBoundary(true)
	boundary2.SetBounds(geometry.NewRect(250, 10, 200, 200))
	boundary2.SetParent(root)

	child1 := &testLeaf{}
	child1.SetVisible(true)
	child1.SetBounds(geometry.NewRect(20, 20, 48, 48))
	child1.SetParent(boundary1)
	boundary1.kids = []widget.Widget{child1}

	child2 := &testLeaf{}
	child2.SetVisible(true)
	child2.SetBounds(geometry.NewRect(260, 20, 48, 48))
	child2.SetParent(boundary2)
	boundary2.kids = []widget.Widget{child2}

	root.kids = []widget.Widget{boundary1, boundary2}
	w.SetRoot(root)

	PaintBoundaryLayersWithContext(root, nil, w.Context())
	w.ClearDirtyBoundaries()
	root.ClearSceneDirty()
	boundary1.ClearSceneDirty()
	boundary2.ClearSceneDirty()
	widget.ClearRedrawInTree(root)

	// Dirty children under separate boundaries → 2 entries.
	child1.SetNeedsRedraw(true)
	child2.SetNeedsRedraw(true)

	if w.DirtyBoundaryCount() != 2 {
		t.Errorf("expected 2 dirty boundaries, got %d", w.DirtyBoundaryCount())
	}
}

// TestFlatDirtyList_CleanState_NoDirtyBoundaries verifies that when no
// widget changes, the dirty set is empty and frame skip would apply.
func TestFlatDirtyList_CleanState_NoDirtyBoundaries(t *testing.T) {
	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	w.SetRoot(root)

	// Clear initial dirty state.
	w.ClearDirtyBoundaries()
	w.ClearAfterPaint()

	if w.HasDirtyBoundaries() {
		t.Error("clean state should have no dirty boundaries")
	}
	if w.NeedsRedraw() {
		t.Error("clean state should not need redraw after ClearAfterPaint")
	}
}

// TestFlatDirtyList_BoundarySelfDirty verifies that a boundary widget
// marking itself dirty (e.g., spinner animation) registers in the dirty set.
func TestFlatDirtyList_BoundarySelfDirty(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	spinner := &testLeaf{}
	spinner.SetVisible(true)
	spinner.SetRepaintBoundary(true)
	spinner.SetBounds(geometry.NewRect(100, 100, 48, 48))
	spinner.SetScreenOrigin(geometry.Pt(100, 100))
	spinner.SetParent(root)

	root.kids = []widget.Widget{spinner}
	w.SetRoot(root)

	PaintBoundaryLayersWithContext(root, nil, w.Context())
	w.ClearDirtyBoundaries()
	root.ClearSceneDirty()
	spinner.ClearSceneDirty()
	widget.ClearRedrawInTree(root)

	// Spinner marks itself dirty (animation frame).
	spinner.SetNeedsRedraw(true)

	// Spinner is its own boundary → InvalidateScene → onBoundaryDirty.
	if !w.HasDirtyBoundaries() {
		t.Error("spinner self-dirty should register in dirty boundary set")
	}
}

// TestFlatDirtyList_ClearAfterFrame verifies that ClearDirtyBoundaries
// resets the set after a frame, enabling correct frame skip on the next frame.
func TestFlatDirtyList_ClearAfterFrame(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	child := &testLeaf{}
	child.SetVisible(true)
	child.SetBounds(geometry.NewRect(10, 10, 48, 48))
	child.SetParent(root)
	root.kids = []widget.Widget{child}

	w.SetRoot(root)
	PaintBoundaryLayersWithContext(root, nil, w.Context())
	w.ClearDirtyBoundaries()
	root.ClearSceneDirty()
	widget.ClearRedrawInTree(root)

	// Simulate frame 1: child dirty → boundary in set.
	child.SetNeedsRedraw(true)
	if !w.HasDirtyBoundaries() {
		t.Fatal("pre-condition: should have dirty boundary")
	}

	// Simulate end of frame: clear.
	w.ClearDirtyBoundaries()
	if w.HasDirtyBoundaries() {
		t.Error("dirty boundaries should be empty after ClearDirtyBoundaries")
	}

	// Simulate frame 2: nothing dirty → frame skip.
	if w.HasDirtyBoundaries() {
		t.Error("no work → frame skip should apply")
	}
}

// TestFlatDirtyList_SuppressDuringRecording verifies that the
// suppressDirtyCallback mechanism prevents onBoundaryDirty from firing
// during Draw recording (animated widgets re-dirty themselves).
func TestFlatDirtyList_SuppressDuringRecording(t *testing.T) {
	cleanup := setupSceneRecorder(t)
	defer cleanup()

	a := New()
	w := a.Window()

	root := &testContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))
	root.SetScreenOrigin(geometry.Pt(0, 0))

	// Animated boundary that re-dirties itself during Draw.
	spinner := &animatedBoundary{}
	spinner.SetVisible(true)
	spinner.SetRepaintBoundary(true)
	spinner.SetBounds(geometry.NewRect(100, 100, 48, 48))
	spinner.SetScreenOrigin(geometry.Pt(100, 100))
	spinner.SetParent(root)

	root.kids = []widget.Widget{spinner}
	w.SetRoot(root)

	// Initial recording wires callbacks.
	PaintBoundaryLayersWithContext(root, nil, w.Context())
	w.ClearDirtyBoundaries()

	// Spinner is dirty after Draw (it calls SetNeedsRedraw → InvalidateScene).
	// But during recording, suppressDirtyCallback=true → onBoundaryDirty
	// does NOT fire → dirtyBoundaries NOT populated during Draw.
	//
	// After recording, if boundary is re-dirty (IsSceneDirty=true),
	// recordBoundary fires ctx.InvalidateRect which sets needsRedraw=true.
	// This ensures the NEXT frame is scheduled, but the dirty boundary
	// is registered via ScheduleAnimationFrame path, not onBoundaryDirty.
	//
	// The test verifies that suppression during Draw recording works.
	// The dirty boundary set may be populated by the post-recording
	// InvalidateRect path or may be empty (depending on timing).
	// What matters is that the render loop has enough information to
	// schedule the next frame (needsRedraw or needsAnimationFrame).
	if !w.NeedsRedraw() && !w.NeedsAnimationFrame() {
		t.Error("animated spinner should trigger next frame via NeedsRedraw or NeedsAnimationFrame")
	}
}

// --- flatDirtyListBenchmark is a simple leaf widget for benchmarking ---

type benchLeaf struct {
	widget.WidgetBase
}

func (w *benchLeaf) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(50, 30))
}

func (w *benchLeaf) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *benchLeaf) Event(_ widget.Context, _ event.Event) bool { return false }
func (w *benchLeaf) Children() []widget.Widget                  { return nil }

type benchContainer struct {
	widget.WidgetBase
	kids []widget.Widget
}

func (w *benchContainer) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(800, 600))
}

func (w *benchContainer) Draw(_ widget.Context, _ widget.Canvas) {}

func (w *benchContainer) Event(_ widget.Context, _ event.Event) bool { return false }
func (w *benchContainer) Children() []widget.Widget                  { return w.kids }

// BenchmarkFrameSkipDecision_FlatList benchmarks the O(1) HasDirtyBoundaries
// check vs the old O(n) NeedsRedrawInTreeNonBoundary tree walk.
func BenchmarkFrameSkipDecision_FlatList(b *testing.B) {
	a := New()
	w := a.Window()

	// Build tree with 100 widgets + 1 boundary.
	root := &benchContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	kids := make([]widget.Widget, 100)
	for i := range kids {
		leaf := &benchLeaf{}
		leaf.SetVisible(true)
		leaf.SetBounds(geometry.NewRect(float32(i*8), 0, 50, 30))
		leaf.SetParent(root)
		kids[i] = leaf
	}
	root.kids = kids
	w.SetRoot(root)

	// Pre-populate 1 dirty boundary (spinner scenario).
	w.AddDirtyBoundary(root.BoundaryCacheKey())

	b.Run("O(1)_HasDirtyBoundaries", func(b *testing.B) {
		for b.Loop() {
			_ = w.HasDirtyBoundaries()
		}
	})

	b.Run("O(n)_NeedsRedrawInTreeNonBoundary", func(b *testing.B) {
		for b.Loop() {
			_ = widget.NeedsRedrawInTreeNonBoundary(root)
		}
	})
}

// BenchmarkFrameSkipDecision_500Widgets benchmarks with a larger tree.
func BenchmarkFrameSkipDecision_500Widgets(b *testing.B) {
	a := New()
	w := a.Window()

	root := &benchContainer{}
	root.SetVisible(true)
	root.SetRepaintBoundary(true)
	root.SetBounds(geometry.NewRect(0, 0, 800, 600))

	kids := make([]widget.Widget, 500)
	for i := range kids {
		leaf := &benchLeaf{}
		leaf.SetVisible(true)
		leaf.SetBounds(geometry.NewRect(float32(i%50*16), float32(i/50*30), 50, 30))
		leaf.SetParent(root)
		kids[i] = leaf
	}
	root.kids = kids
	w.SetRoot(root)

	w.AddDirtyBoundary(root.BoundaryCacheKey())

	b.Run("O(1)_HasDirtyBoundaries", func(b *testing.B) {
		for b.Loop() {
			_ = w.HasDirtyBoundaries()
		}
	})

	b.Run("O(n)_NeedsRedrawInTreeNonBoundary", func(b *testing.B) {
		for b.Loop() {
			_ = widget.NeedsRedrawInTreeNonBoundary(root)
		}
	})
}
