package transition

import (
	"testing"
	"time"

	"github.com/gogpu/ui/animation"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- NewFade construction tests ---

func TestNewFadeDefaults(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child)

	if f.Child() != child {
		t.Error("Child() should return wrapped widget")
	}
	if f.duration != defaultFadeDuration {
		t.Errorf("default duration: got %v, want %v", f.duration, defaultFadeDuration)
	}
	if !f.autoStart {
		t.Error("autoStart should default to true")
	}
	if f.opacity != 0 {
		t.Errorf("initial opacity: got %v, want 0", f.opacity)
	}
	if f.IsAnimating() {
		t.Error("should not be animating before mount")
	}
	if f.bgColor != widget.ColorWhite {
		t.Error("default background color should be white")
	}
}

func TestNewFadeWithOptions(t *testing.T) {
	child := newMockWidget()
	dur := 500 * time.Millisecond
	bg := widget.ColorBlack
	f := NewFade(child,
		FadeDuration(dur),
		FadeEasing(animation.Linear),
		FadeAutoStart(false),
		FadeBackground(bg),
	)

	if f.duration != dur {
		t.Errorf("duration: got %v, want %v", f.duration, dur)
	}
	if f.autoStart {
		t.Error("autoStart should be false")
	}
	if f.bgColor != bg {
		t.Error("background color should be black")
	}
}

func TestNewFadeNilChild(t *testing.T) {
	f := NewFade(nil)
	if f.Children() != nil {
		t.Error("Children should return nil for nil child")
	}

	// Layout/Draw/Event should not panic with nil child.
	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	_ = f.Layout(ctx, geometry.Constraints{MaxWidth: 100, MaxHeight: 100})
	f.Draw(ctx, canvas)
	if f.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should return false for nil child")
	}
}

// --- FadeIn/FadeOut trigger tests ---

func TestFadeInTriggersAnimation(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.FadeIn()
	if !f.IsAnimating() {
		t.Error("should be animating after FadeIn")
	}
	if !f.fadingIn {
		t.Error("should be in fading-in state")
	}
}

func TestFadeOutTriggersAnimation(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.FadeOut()
	if !f.IsAnimating() {
		t.Error("should be animating after FadeOut")
	}
	if f.fadingIn {
		t.Error("should be in fading-out state")
	}
	if f.opacity != 1.0 {
		t.Errorf("FadeOut should set opacity to 1.0, got %v", f.opacity)
	}
}

func TestFadeInRestartsAnimation(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.FadeOut()
	f.progress = 0.5

	f.FadeIn()
	if f.progress != 0 {
		t.Error("FadeIn should restart progress to 0")
	}
	if !f.fadingIn {
		t.Error("should switch to fading-in state")
	}
}

// --- SetOpacity tests ---

func TestSetOpacityClampsValues(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.SetOpacity(1.5)
	if f.Opacity() != 1.0 {
		t.Errorf("SetOpacity(1.5) should clamp to 1.0, got %v", f.Opacity())
	}

	f.SetOpacity(-0.5)
	if f.Opacity() != 0 {
		t.Errorf("SetOpacity(-0.5) should clamp to 0, got %v", f.Opacity())
	}

	f.SetOpacity(0.7)
	if f.Opacity() != 0.7 {
		t.Errorf("SetOpacity(0.7): got %v", f.Opacity())
	}
}

func TestSetOpacityStopsAnimation(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.FadeIn()
	if !f.IsAnimating() {
		t.Error("should be animating")
	}

	f.SetOpacity(0.5)
	if f.IsAnimating() {
		t.Error("SetOpacity should stop animation")
	}
}

// --- Layout tests ---

func TestFadeLayoutDelegatesToChild(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))
	ctx := newMockContext(time.Now())

	f.SetBounds(geometry.NewRect(10, 20, 200, 200))
	size := f.Layout(ctx, geometry.Constraints{MaxWidth: 200, MaxHeight: 200})

	if size.Width != 100 || size.Height != 50 {
		t.Errorf("Layout size: got %v, want (100,50)", size)
	}
}

// --- Draw tests ---

func TestFadeDrawAtZeroOpacity(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))
	f.opacity = 0

	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	f.Draw(ctx, canvas)

	if child.drawn != 0 {
		t.Error("should not draw child at zero opacity")
	}
}

func TestFadeDrawAtFullOpacity(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))
	f.opacity = 1.0

	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	f.Draw(ctx, canvas)

	if child.drawn != 1 {
		t.Errorf("should draw child at full opacity, got %d draws", child.drawn)
	}
}

func TestFadeDrawWithOpacityPusher(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))
	f.opacity = 0.5

	ctx := newMockContext(time.Now())
	canvas := &opacityCanvas{}
	f.Draw(ctx, canvas)

	if child.drawn != 1 {
		t.Errorf("should draw child with opacity pusher, got %d draws", child.drawn)
	}
	// OpacityPusher should have been called and popped.
	if len(canvas.opacities) != 0 {
		t.Errorf("opacity stack should be empty after draw, got %d", len(canvas.opacities))
	}
}

func TestFadeDrawWithOverlayFallback(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false), FadeBackground(widget.ColorBlack))
	f.opacity = 0.5

	ctx := newMockContext(time.Now())

	// Use a recording canvas that captures draw calls.
	canvas := &fadeTestCanvas{}

	// Set up child bounds.
	child.SetBounds(geometry.NewRect(0, 0, 100, 50))
	f.SetBounds(geometry.NewRect(0, 0, 100, 50))

	f.Draw(ctx, canvas)

	if child.drawn != 1 {
		t.Errorf("should draw child, got %d draws", child.drawn)
	}
	// Should have drawn an overlay rect.
	if len(canvas.rects) != 1 {
		t.Errorf("should draw 1 overlay rect, got %d", len(canvas.rects))
	}
	// Overlay alpha should be 1-opacity = 0.5.
	overlay := canvas.rects[0]
	if overlay.color.A < 0.49 || overlay.color.A > 0.51 {
		t.Errorf("overlay alpha: got %v, want ~0.5", overlay.color.A)
	}
}

func TestFadeDrawFadeInAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 200 * time.Millisecond
	f := NewFade(child,
		FadeDuration(dur),
		FadeEasing(animation.Linear),
		FadeAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &opacityCanvas{}

	f.FadeIn()

	// First draw: initializes start time.
	f.Draw(ctx, canvas)
	// Opacity at t=0 should be 0 (invisible), so child not drawn.
	if child.drawn != 0 {
		t.Errorf("draw count at t=0: got %d, want 0 (opacity=0)", child.drawn)
	}

	// Advance halfway.
	ctx.now = now.Add(100 * time.Millisecond)
	f.Draw(ctx, canvas)
	if !f.IsAnimating() {
		t.Error("should still be animating at 50%")
	}
	// Opacity should be ~0.5.
	if f.Opacity() < 0.4 || f.Opacity() > 0.6 {
		t.Errorf("opacity at 50%%: got %v, want ~0.5", f.Opacity())
	}

	// Advance past duration.
	ctx.now = now.Add(300 * time.Millisecond)
	f.Draw(ctx, canvas)
	if f.IsAnimating() {
		t.Error("should stop animating after duration")
	}
	if f.Opacity() != 1.0 {
		t.Errorf("opacity after complete: got %v, want 1.0", f.Opacity())
	}
}

func TestFadeDrawFadeOutAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 100 * time.Millisecond
	f := NewFade(child,
		FadeDuration(dur),
		FadeEasing(animation.Linear),
		FadeAutoStart(false),
	)
	f.opacity = 1.0

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &opacityCanvas{}

	f.FadeOut()

	// First draw.
	f.Draw(ctx, canvas)

	// Advance past duration.
	ctx.now = now.Add(200 * time.Millisecond)
	f.Draw(ctx, canvas)
	if f.IsAnimating() {
		t.Error("should stop animating after duration")
	}
	if f.Opacity() != 0 {
		t.Errorf("opacity after fadeout: got %v, want 0", f.Opacity())
	}
}

func TestFadeDrawInvalidatesDuringAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 200 * time.Millisecond
	f := NewFade(child,
		FadeDuration(dur),
		FadeAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &mockCanvas{}

	f.FadeIn()
	f.Draw(ctx, canvas)

	// After first draw (still animating), NeedsRedraw should be true.
	if !f.NeedsRedraw() {
		t.Error("should set NeedsRedraw during animation")
	}
}

// --- Mount/Unmount lifecycle tests ---

func TestFadeMountAutoStart(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child) // autoStart=true (default)

	ctx := newMockContext(time.Now())
	f.Mount(ctx)

	if !f.IsAnimating() {
		t.Error("should start animating on mount with autoStart")
	}
	if !f.fadingIn {
		t.Error("default mount should fade in")
	}
}

func TestFadeMountNoAutoStart(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	ctx := newMockContext(time.Now())
	f.Mount(ctx)

	if f.IsAnimating() {
		t.Error("should not animate with autoStart=false")
	}
}

func TestFadeUnmountStopsAnimation(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))

	f.FadeIn()
	if !f.IsAnimating() {
		t.Error("should be animating")
	}

	f.Unmount()
	if f.IsAnimating() {
		t.Error("should stop animating after unmount")
	}
}

// --- Event tests ---

func TestFadeEventForwardsToChild(t *testing.T) {
	child := newMockWidget()
	child.eventRet = true
	f := NewFade(child, FadeAutoStart(false))

	ctx := newMockContext(time.Now())
	if !f.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should forward to child and return its result")
	}
}

func TestFadeEventNilChild(t *testing.T) {
	f := NewFade(nil)
	ctx := newMockContext(time.Now())
	if f.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should return false for nil child")
	}
}

// --- Children tests ---

func TestFadeChildrenReturnsChild(t *testing.T) {
	child := newMockWidget()
	f := NewFade(child, FadeAutoStart(false))
	children := f.Children()
	if len(children) != 1 || children[0] != child {
		t.Error("Children should return single-element slice with child")
	}
}

func TestFadeChildrenNilChild(t *testing.T) {
	f := NewFade(nil)
	if f.Children() != nil {
		t.Error("Children should return nil for nil child")
	}
}

// --- clampFloat32 tests ---

func TestClampFloat32(t *testing.T) {
	tests := []struct {
		input float32
		want  float32
	}{
		{-1, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{2, 1},
	}
	for _, tt := range tests {
		got := clampFloat32(tt.input)
		if got != tt.want {
			t.Errorf("clampFloat32(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Interface compliance ---

func TestFadeImplementsWidget(t *testing.T) {
	var _ widget.Widget = (*Fade)(nil)
}

func TestFadeImplementsLifecycle(t *testing.T) {
	var _ widget.Lifecycle = (*Fade)(nil)
}

// --- Test helper: recording canvas for overlay testing ---

type rectCall struct {
	bounds geometry.Rect
	color  widget.Color
}

// fadeTestCanvas records DrawRect calls for verifying overlay approach.
type fadeTestCanvas struct {
	mockCanvas
	rects []rectCall
}

func (c *fadeTestCanvas) DrawRect(r geometry.Rect, color widget.Color) {
	c.rects = append(c.rects, rectCall{bounds: r, color: color})
}
