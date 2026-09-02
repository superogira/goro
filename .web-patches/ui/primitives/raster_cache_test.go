package primitives_test

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// --- RasterCacheConfig Tests ---

func TestDefaultRasterCacheConfig(t *testing.T) {
	cfg := primitives.DefaultRasterCacheConfig()

	if cfg.PromotionThreshold != 3 {
		t.Errorf("PromotionThreshold = %d, want 3", cfg.PromotionThreshold)
	}
	if cfg.MinArea != 4096 {
		t.Errorf("MinArea = %d, want 4096", cfg.MinArea)
	}
	if cfg.MinComplexity != 20 {
		t.Errorf("MinComplexity = %d, want 20", cfg.MinComplexity)
	}
}

func TestWithRasterCacheConfig(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 5,
		MinArea:            1024,
		MinComplexity:      10,
	}
	rb := primitives.NewRepaintBoundary(nil, primitives.WithRasterCacheConfig(cfg))
	stats := rb.RasterCacheStats()

	// Initial state: not stable, zero hits.
	if stats.IsStable {
		t.Error("should not be stable initially")
	}
	if stats.ConsecutiveHits != 0 {
		t.Errorf("ConsecutiveHits = %d, want 0", stats.ConsecutiveHits)
	}
}

// --- RasterCache Promotion Tests ---

func TestRasterCache_PromotionAfterThreshold(t *testing.T) {
	// Use a low-threshold config and a complex child to trigger promotion.
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 2,
		MinArea:            100, // 10x10
		MinComplexity:      1,   // very low for test
	}

	child := newComplexDrawWidget(30) // 30 draw calls = 30+ tags
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}

	// Draw 1: cache miss — records scene.
	rb.Draw(nil, canvas)
	if rb.IsStable() {
		t.Error("should not be stable after first draw (cache miss)")
	}

	// Draw 2: cache hit #1 — not yet at threshold.
	rb.Draw(nil, canvas)
	if rb.ConsecutiveHits() != 1 {
		t.Errorf("ConsecutiveHits = %d, want 1", rb.ConsecutiveHits())
	}
	if rb.IsStable() {
		t.Error("should not be stable after 1 consecutive hit")
	}

	// Draw 3: cache hit #2 — meets threshold.
	rb.Draw(nil, canvas)
	if rb.ConsecutiveHits() != 2 {
		t.Errorf("ConsecutiveHits = %d, want 2", rb.ConsecutiveHits())
	}
	if !rb.IsStable() {
		t.Error("should be stable after 2 consecutive hits (threshold=2)")
	}

	stats := rb.RasterCacheStats()
	if stats.TotalPromotions != 1 {
		t.Errorf("TotalPromotions = %d, want 1", stats.TotalPromotions)
	}
}

func TestRasterCache_DemotionOnDirty(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            100,
		MinComplexity:      1,
	}

	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}

	// Draw 1: miss. Draw 2: hit → promote.
	rb.Draw(nil, canvas)
	rb.Draw(nil, canvas)
	if !rb.IsStable() {
		t.Fatal("pre-condition: should be stable")
	}

	// Mark dirty → should reset consecutive hits.
	rb.MarkBoundaryDirty()
	if rb.ConsecutiveHits() != 0 {
		t.Errorf("ConsecutiveHits = %d after dirty, want 0", rb.ConsecutiveHits())
	}

	// Draw 3: cache miss → demotes.
	rb.Draw(nil, canvas)
	if rb.IsStable() {
		t.Error("should be demoted after cache miss")
	}

	stats := rb.RasterCacheStats()
	if stats.TotalDemotions != 1 {
		t.Errorf("TotalDemotions = %d, want 1", stats.TotalDemotions)
	}
}

func TestRasterCache_NoPromotionBelowMinArea(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            10000, // 100x100 required
		MinComplexity:      1,
	}

	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	// 50x50 = 2500 pixels < 10000 MinArea.
	rb.Layout(nil, geometry.Tight(geometry.Sz(50, 50)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas) // miss
	rb.Draw(nil, canvas) // hit #1

	if rb.IsStable() {
		t.Error("should not be stable when area < MinArea")
	}
}

func TestRasterCache_NoPromotionBelowMinComplexity(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            100,
		MinComplexity:      50, // requires 50+ tags
	}

	// Simple child: only 1 draw call.
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas) // miss
	rb.Draw(nil, canvas) // hit

	if rb.IsStable() {
		t.Error("should not be stable when tag count < MinComplexity")
	}
}

func TestRasterCache_RepromotionAfterDemotion(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            100,
		MinComplexity:      1,
	}

	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}

	// Promote.
	rb.Draw(nil, canvas)
	rb.Draw(nil, canvas)
	if !rb.IsStable() {
		t.Fatal("pre-condition: should be stable")
	}

	// Demote.
	rb.MarkBoundaryDirty()
	rb.Draw(nil, canvas) // miss

	// Re-promote.
	rb.Draw(nil, canvas) // hit → threshold met again
	if !rb.IsStable() {
		t.Error("should be stable again after re-promotion")
	}

	stats := rb.RasterCacheStats()
	if stats.TotalPromotions != 2 {
		t.Errorf("TotalPromotions = %d, want 2", stats.TotalPromotions)
	}
	if stats.TotalDemotions != 1 {
		t.Errorf("TotalDemotions = %d, want 1", stats.TotalDemotions)
	}
}

func TestRasterCache_UnmountResetsState(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            100,
		MinComplexity:      1,
	}

	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)
	rb.Draw(nil, canvas) // promote

	if !rb.IsStable() {
		t.Fatal("pre-condition: should be stable")
	}

	rb.Unmount()

	if rb.IsStable() {
		t.Error("should not be stable after Unmount")
	}
	if rb.ConsecutiveHits() != 0 {
		t.Errorf("ConsecutiveHits = %d after Unmount, want 0", rb.ConsecutiveHits())
	}
}

// --- RasterCacheStats Tests ---

func TestRasterCacheStats_Initial(t *testing.T) {
	rb := primitives.NewRepaintBoundary(nil)
	stats := rb.RasterCacheStats()

	if stats.ConsecutiveHits != 0 {
		t.Errorf("ConsecutiveHits = %d, want 0", stats.ConsecutiveHits)
	}
	if stats.IsStable {
		t.Error("should not be stable initially")
	}
	if stats.TagCount != 0 {
		t.Errorf("TagCount = %d, want 0", stats.TagCount)
	}
	if stats.Area != 0 {
		t.Errorf("Area = %d, want 0", stats.Area)
	}
	if stats.TotalPromotions != 0 {
		t.Errorf("TotalPromotions = %d, want 0", stats.TotalPromotions)
	}
	if stats.TotalDemotions != 0 {
		t.Errorf("TotalDemotions = %d, want 0", stats.TotalDemotions)
	}
}

func TestRasterCacheStats_AfterDraw(t *testing.T) {
	child := newComplexDrawWidget(10)
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(200, 150)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	stats := rb.RasterCacheStats()

	if stats.Area != 200*150 {
		t.Errorf("Area = %d, want %d", stats.Area, 200*150)
	}
	if stats.TagCount == 0 {
		t.Error("TagCount should be > 0 after drawing complex child")
	}
}

// --- SceneVersion / SceneChanged Tests (Task 3d) ---

func TestSceneVersion_InitialZero(t *testing.T) {
	rb := primitives.NewRepaintBoundary(nil)
	if rb.SceneVersion() != 0 {
		t.Errorf("SceneVersion = %d, want 0", rb.SceneVersion())
	}
}

func TestSceneVersion_IncrementsOnCacheMiss(t *testing.T) {
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 50)))

	canvas := &replayRecordingCanvas{}

	// First draw: cache miss.
	rb.Draw(nil, canvas)
	v1 := rb.SceneVersion()
	if v1 == 0 {
		t.Error("SceneVersion should be > 0 after first draw")
	}

	// Second draw: cache hit — version should NOT change.
	rb.Draw(nil, canvas)
	v2 := rb.SceneVersion()
	if v2 != v1 {
		t.Errorf("SceneVersion changed on cache hit: %d → %d", v1, v2)
	}

	// Third draw: dirty → cache miss — version should change.
	rb.MarkBoundaryDirty()
	rb.Draw(nil, canvas)
	v3 := rb.SceneVersion()
	if v3 == v2 {
		t.Error("SceneVersion should change on cache miss after dirty")
	}
}

func TestSceneChanged(t *testing.T) {
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 50)))

	canvas := &replayRecordingCanvas{}

	// Before any draw: version 0, SceneChanged(0) should be false.
	if rb.SceneChanged(0) {
		t.Error("SceneChanged(0) should be false before any draw")
	}

	// First draw: version changes.
	rb.Draw(nil, canvas)
	v1 := rb.SceneVersion()
	if !rb.SceneChanged(0) {
		t.Error("SceneChanged(0) should be true after first draw")
	}
	if rb.SceneChanged(v1) {
		t.Error("SceneChanged(v1) should be false (current version)")
	}

	// Cache hit: no change.
	rb.Draw(nil, canvas)
	if rb.SceneChanged(v1) {
		t.Error("SceneChanged(v1) should be false after cache hit")
	}

	// Dirty + redraw: version changes.
	rb.MarkBoundaryDirty()
	rb.Draw(nil, canvas)
	if !rb.SceneChanged(v1) {
		t.Error("SceneChanged(v1) should be true after dirty redraw")
	}
}

func TestSceneVersion_ResetOnUnmount(t *testing.T) {
	child := newDrawCountingWidget()
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 50)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas)

	if rb.SceneVersion() == 0 {
		t.Fatal("pre-condition: SceneVersion should be > 0")
	}

	rb.Unmount()

	if rb.SceneVersion() != 0 {
		t.Errorf("SceneVersion = %d after Unmount, want 0", rb.SceneVersion())
	}
}

// --- Stability with Default Config Tests ---

func TestRasterCache_DefaultConfig_RequiresThreeHits(t *testing.T) {
	// With default config (threshold=3), need 3 consecutive hits.
	child := newComplexDrawWidget(30) // 30 draw calls
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}

	rb.Draw(nil, canvas) // miss
	rb.Draw(nil, canvas) // hit 1
	rb.Draw(nil, canvas) // hit 2

	if rb.IsStable() {
		t.Error("should not be stable after 2 hits (threshold=3)")
	}

	rb.Draw(nil, canvas) // hit 3

	if !rb.IsStable() {
		t.Error("should be stable after 3 hits (threshold=3)")
	}
}

func TestRasterCache_StayStableOnContinuedHits(t *testing.T) {
	cfg := primitives.RasterCacheConfig{
		PromotionThreshold: 1,
		MinArea:            100,
		MinComplexity:      1,
	}

	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child, primitives.WithRasterCacheConfig(cfg))
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas) // miss
	rb.Draw(nil, canvas) // hit → promote

	// 10 more hits — should stay stable, no extra promotions.
	for range 10 {
		rb.Draw(nil, canvas)
	}

	stats := rb.RasterCacheStats()
	if !stats.IsStable {
		t.Error("should remain stable on continued hits")
	}
	if stats.TotalPromotions != 1 {
		t.Errorf("TotalPromotions = %d, want 1 (no double-promotion)", stats.TotalPromotions)
	}
	if stats.ConsecutiveHits != 11 {
		t.Errorf("ConsecutiveHits = %d, want 11", stats.ConsecutiveHits)
	}
}

func TestRasterCache_InvalidateCacheResetsHits(t *testing.T) {
	child := newComplexDrawWidget(30)
	rb := primitives.NewRepaintBoundary(child)
	rb.Layout(nil, geometry.Tight(geometry.Sz(100, 100)))

	canvas := &replayRecordingCanvas{}
	rb.Draw(nil, canvas) // miss
	rb.Draw(nil, canvas) // hit 1

	rb.InvalidateCache()
	rb.Draw(nil, canvas) // miss (invalidated)

	// Consecutive hits should be reset.
	if rb.ConsecutiveHits() != 0 {
		t.Errorf("ConsecutiveHits = %d after InvalidateCache, want 0", rb.ConsecutiveHits())
	}
}

// --- Helper: complex drawing widget ---

// complexDrawWidget performs multiple draw calls to generate a scene with
// many encoding tags. Used to test the MinComplexity criterion.
type complexDrawWidget struct {
	widget.WidgetBase
	drawCalls int
}

func newComplexDrawWidget(calls int) *complexDrawWidget {
	w := &complexDrawWidget{drawCalls: calls}
	w.SetVisible(true)
	w.SetEnabled(true)
	w.SetNeedsRedraw(true)
	return w
}

func (w *complexDrawWidget) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(100, 100))
}

func (w *complexDrawWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	for i := range w.drawCalls {
		// Each DrawRect generates at least one encoding tag in SceneCanvas.
		canvas.DrawRect(
			geometry.NewRect(float32(i), float32(i), 10, 10),
			widget.ColorRed,
		)
	}
}

func (w *complexDrawWidget) Event(_ widget.Context, _ event.Event) bool { return false }

var _ widget.Widget = (*complexDrawWidget)(nil)
