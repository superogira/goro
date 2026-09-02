package primitives_test

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// --- Scene Display List Tests (ADR-007 Phase 2) ---

func TestRepaintBoundary_Scene_SmallWidget(t *testing.T) {
	// Even small widgets now use scene.Scene (ADR-007 unifies the path).
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)

	rb.Layout(nil, geometry.Tight(geometry.Sz(64, 64)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	if child.drawCount != 1 {
		t.Errorf("expected child Draw called once, got %d", child.drawCount)
	}
	if !rb.CacheValid() {
		t.Error("cache should be valid after draw")
	}
	if canvas.replayCount != 1 {
		t.Errorf("expected 1 ReplayScene call, got %d", canvas.replayCount)
	}
}

func TestRepaintBoundary_Scene_LargeWidget(t *testing.T) {
	// Large widgets also use scene.Scene display list.
	child := &colorFillWidget{color: widget.ColorRed}
	child.SetVisible(true)
	child.SetEnabled(true)
	child.SetNeedsRedraw(true)

	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(256, 256)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	if !rb.CacheValid() {
		t.Error("cache should be valid after scene recording")
	}
	if canvas.replayCount != 1 {
		t.Errorf("expected 1 ReplayScene call, got %d", canvas.replayCount)
	}
	if canvas.replayScenes[0] == nil {
		t.Error("ReplayScene received nil scene")
	}
}

func TestRepaintBoundary_Scene_Reuse(t *testing.T) {
	// Scene should be reused across frames (Reset, not recreated).
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)

	rb.Layout(nil, geometry.Tight(geometry.Sz(200, 200)))

	// First draw: creates scene.
	canvas1 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas1)

	if child.drawCount != 1 {
		t.Errorf("expected 1 draw after first frame, got %d", child.drawCount)
	}

	// Mark boundary dirty for a second render.
	rb.MarkBoundaryDirty()

	canvas2 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas2)

	if child.drawCount != 2 {
		t.Errorf("expected 2 draws after dirty second frame, got %d", child.drawCount)
	}

	// Same scene object should be reused (Reset, not new allocation).
	if canvas1.replayScenes[0] != canvas2.replayScenes[0] {
		t.Error("scene should be reused across frames (Reset, not recreated)")
	}
}

func TestRepaintBoundary_Scene_Resize(t *testing.T) {
	// Scene cache should be invalidated when size changes.
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)

	rb.Layout(nil, geometry.Tight(geometry.Sz(200, 200)))
	canvas1 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas1)

	if !rb.CacheValid() {
		t.Error("cache should be valid after first draw")
	}

	// Resize: should invalidate cache.
	rb.Layout(nil, geometry.Tight(geometry.Sz(300, 300)))
	if rb.CacheValid() {
		t.Error("cache should be invalid after size change")
	}

	canvas2 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas2)

	if !rb.CacheValid() {
		t.Error("cache should be valid after re-record")
	}
	if child.drawCount != 2 {
		t.Errorf("child drawn %d times, want 2 (re-record after resize)", child.drawCount)
	}
}

func TestRepaintBoundary_Scene_Unmount(t *testing.T) {
	// All scene resources should be freed on Unmount.
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)

	rb.Layout(nil, geometry.Tight(geometry.Sz(256, 256)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	if !rb.CacheValid() {
		t.Error("cache should be valid before unmount")
	}

	rb.Unmount()

	if rb.CacheValid() {
		t.Error("cache should be invalid after Unmount")
	}
	if rb.CacheHits() != 0 {
		t.Error("cache hits should be reset after Unmount")
	}
}

func TestRepaintBoundary_Scene_CacheHit(t *testing.T) {
	// When cache is valid and boundary is clean, should replay cached scene.
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)

	rb.Layout(nil, geometry.Tight(geometry.Sz(200, 200)))

	canvas1 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas1) // Cache miss.

	canvas2 := &replayRecordingCanvas{}
	rb.Draw(nil, canvas2) // Cache hit.

	if child.drawCount != 1 {
		t.Errorf("expected child Draw called once (cached on second), got %d", child.drawCount)
	}
	if rb.CacheHits() != 1 {
		t.Errorf("expected 1 cache hit, got %d", rb.CacheHits())
	}
}

func TestRepaintBoundary_Scene_NonEmpty(t *testing.T) {
	// Verify that the replayed scene is not empty.
	child := &colorFillWidget{color: widget.ColorBlue}
	child.SetVisible(true)
	child.SetEnabled(true)
	child.SetNeedsRedraw(true)

	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	if canvas.replayCount != 1 {
		t.Fatalf("expected 1 ReplayScene call, got %d", canvas.replayCount)
	}
	s := canvas.replayScenes[0]
	if s == nil || s.IsEmpty() {
		t.Error("replayed scene should not be nil or empty after recording child drawing")
	}
}

// replayRecordingCanvas for this file is defined in repaint_boundary_cache_test.go.

// --- Helper test widget that fills with a color ---

// colorFillWidget draws a solid color fill across its bounds.
type colorFillWidget struct {
	widget.WidgetBase
	color widget.Color
}

func (w *colorFillWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(256, 256))
}

func (w *colorFillWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	canvas.DrawRect(geometry.NewRect(0, 0, 256, 256), w.color)
}

func (w *colorFillWidget) Event(_ widget.Context, _ event.Event) bool {
	return false
}

var _ widget.Widget = (*colorFillWidget)(nil)

// Verify replayRecordingCanvas satisfies widget.Canvas.
var _ widget.Canvas = (*replayRecordingCanvas)(nil)
