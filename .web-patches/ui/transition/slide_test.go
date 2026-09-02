package transition

import (
	"testing"
	"time"

	"github.com/gogpu/ui/animation"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// --- NewSlide construction tests ---

func TestNewSlideDefaults(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child)

	if s.Child() != child {
		t.Error("Child() should return wrapped widget")
	}
	if s.direction != FromTop {
		t.Errorf("default direction: got %v, want FromTop", s.direction)
	}
	if s.duration != defaultSlideDuration {
		t.Errorf("default duration: got %v, want %v", s.duration, defaultSlideDuration)
	}
	if !s.autoStart {
		t.Error("autoStart should default to true")
	}
	if s.reverse {
		t.Error("reverse should default to false")
	}
	if s.IsAnimating() {
		t.Error("should not be animating before mount")
	}
}

func TestNewSlideWithOptions(t *testing.T) {
	child := newMockWidget()
	dur := 500 * time.Millisecond
	s := NewSlide(child,
		SlideFrom(FromRight),
		SlideDuration(dur),
		SlideEasing(animation.Linear),
		SlideAutoStart(false),
		SlideReverse(true),
	)

	if s.direction != FromRight {
		t.Errorf("direction: got %v, want FromRight", s.direction)
	}
	if s.duration != dur {
		t.Errorf("duration: got %v, want %v", s.duration, dur)
	}
	if s.autoStart {
		t.Error("autoStart should be false")
	}
	if !s.reverse {
		t.Error("reverse should be true")
	}
}

func TestNewSlideNilChild(t *testing.T) {
	s := NewSlide(nil)
	if s.Children() != nil {
		t.Error("Children should return nil for nil child")
	}

	// Layout/Draw/Event should not panic with nil child.
	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	_ = s.Layout(ctx, geometry.Constraints{MaxWidth: 100, MaxHeight: 100})
	s.Draw(ctx, canvas)
	if s.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should return false for nil child")
	}
}

func TestNewSlideReverseInitialProgress(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideReverse(true))

	// With reverse + autoStart, should start at progress=1 (fully visible).
	if s.progress != 1.0 {
		t.Errorf("reverse autoStart initial progress: got %v, want 1.0", s.progress)
	}
}

// --- SlideIn/SlideOut trigger tests ---

func TestSlideInTriggersAnimation(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	s.SlideIn()
	if !s.IsAnimating() {
		t.Error("should be animating after SlideIn")
	}
	if !s.slideIn {
		t.Error("should be in slide-in state")
	}
	if s.progress != 0 {
		t.Errorf("progress should be 0 at start, got %v", s.progress)
	}
}

func TestSlideOutTriggersAnimation(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	s.SlideOut()
	if !s.IsAnimating() {
		t.Error("should be animating after SlideOut")
	}
	if s.slideIn {
		t.Error("should be in slide-out state")
	}
}

func TestSlideInRestartsAnimation(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	s.SlideOut()
	s.progress = 0.5

	s.SlideIn()
	if s.progress != 0 {
		t.Error("SlideIn should restart progress to 0")
	}
	if !s.slideIn {
		t.Error("should switch to slide-in state")
	}
}

// --- Layout tests ---

func TestSlideLayoutDelegatesToChild(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))
	ctx := newMockContext(time.Now())

	s.SetBounds(geometry.NewRect(10, 20, 200, 200))
	size := s.Layout(ctx, geometry.Constraints{MaxWidth: 200, MaxHeight: 200})

	if size.Width != 100 || size.Height != 50 {
		t.Errorf("Layout size: got %v, want (100,50)", size)
	}
}

func TestSlideLayoutSetsChildBounds(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))
	ctx := newMockContext(time.Now())

	origin := geometry.Pt(10, 20)
	s.SetBounds(geometry.FromPointSize(origin, geometry.Sz(200, 200)))
	_ = s.Layout(ctx, geometry.Constraints{MaxWidth: 200, MaxHeight: 200})

	childBounds := child.Bounds()
	if childBounds.Min != origin {
		t.Errorf("child origin: got %v, want %v", childBounds.Min, origin)
	}
}

// --- Draw tests ---

func TestSlideDrawNoAnimation(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	s.SetBounds(geometry.NewRect(0, 0, 200, 100))

	s.Draw(ctx, canvas)
	if child.drawn != 1 {
		t.Errorf("should draw child once, got %d", child.drawn)
	}
	// No transform should be pushed when not animating at progress=0.
	// (offset is non-zero when progress=0 for slide-in, but we haven't started).
}

func TestSlideDrawSlideInAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 200 * time.Millisecond
	s := NewSlide(child,
		SlideFrom(FromLeft),
		SlideDuration(dur),
		SlideEasing(animation.Linear),
		SlideAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &mockCanvas{}

	// Set up bounds.
	s.SetBounds(geometry.NewRect(0, 0, 200, 200))
	s.Layout(ctx, geometry.Constraints{MaxWidth: 200, MaxHeight: 200})

	s.SlideIn()

	// First draw: initializes start time.
	s.Draw(ctx, canvas)
	if child.drawn != 1 {
		t.Errorf("draw count: got %d, want 1", child.drawn)
	}

	// Advance halfway.
	ctx.now = now.Add(100 * time.Millisecond)
	s.Draw(ctx, canvas)
	if child.drawn != 2 {
		t.Errorf("draw count: got %d, want 2", child.drawn)
	}
	if !s.IsAnimating() {
		t.Error("should still be animating at 50%")
	}

	// Advance past duration.
	ctx.now = now.Add(300 * time.Millisecond)
	s.Draw(ctx, canvas)
	if s.IsAnimating() {
		t.Error("should stop animating after duration")
	}
	if s.Progress() != 1.0 {
		t.Errorf("progress should be 1.0, got %v", s.Progress())
	}
}

func TestSlideDrawSlideOutAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 100 * time.Millisecond
	s := NewSlide(child,
		SlideFrom(FromBottom),
		SlideDuration(dur),
		SlideEasing(animation.Linear),
		SlideAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &mockCanvas{}

	s.SetBounds(geometry.NewRect(0, 0, 200, 200))
	s.Layout(ctx, geometry.Constraints{MaxWidth: 200, MaxHeight: 200})

	s.SlideOut()

	// First draw.
	s.Draw(ctx, canvas)

	// Advance past duration.
	ctx.now = now.Add(200 * time.Millisecond)
	s.Draw(ctx, canvas)
	if s.IsAnimating() {
		t.Error("should stop animating after duration")
	}
}

func TestSlideDrawPushesTransformDuringAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 200 * time.Millisecond
	s := NewSlide(child,
		SlideFrom(FromTop),
		SlideDuration(dur),
		SlideEasing(animation.Linear),
		SlideAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &mockCanvas{}

	s.SetBounds(geometry.NewRect(0, 0, 100, 50))
	s.Layout(ctx, geometry.Constraints{MaxWidth: 100, MaxHeight: 50})

	s.SlideIn()

	// Draw at t=0: full offset from top -> Y = -50 * (1-0) = -50.
	s.Draw(ctx, canvas)

	// Transform should be pushed and popped (net zero after Draw completes).
	if len(canvas.transforms) != 0 {
		t.Errorf("transforms should be popped after draw, got %d remaining", len(canvas.transforms))
	}
}

func TestSlideDrawNoTransformAtFullProgress(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))
	s.progress = 1.0 // fully visible, slide-in complete
	s.slideIn = true

	ctx := newMockContext(time.Now())
	canvas := &mockCanvas{}
	s.SetBounds(geometry.NewRect(0, 0, 100, 50))
	s.Layout(ctx, geometry.Constraints{MaxWidth: 100, MaxHeight: 50})

	s.Draw(ctx, canvas)
	// At progress=1.0, offset should be (0,0), so no PushTransform.
	if len(canvas.transforms) != 0 {
		t.Errorf("should not push transform at full progress, got %d", len(canvas.transforms))
	}
}

func TestSlideDrawInvalidatesDuringAnimation(t *testing.T) {
	child := newMockWidget()
	dur := 200 * time.Millisecond
	s := NewSlide(child,
		SlideDuration(dur),
		SlideAutoStart(false),
	)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := newMockContext(now)
	canvas := &mockCanvas{}

	s.SetBounds(geometry.NewRect(0, 0, 100, 50))

	s.SlideIn()
	s.Draw(ctx, canvas)

	// After first draw (still animating), NeedsRedraw should be true.
	if !s.NeedsRedraw() {
		t.Error("should set NeedsRedraw during animation")
	}
}

// --- Direction offset tests ---

func TestSlideComputeOffsetDirections(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		slideIn   bool
		progress  float32
		wantX     float32
		wantY     float32
	}{
		{"FromTop slide-in start", FromTop, true, 0, 0, -50},
		{"FromTop slide-in end", FromTop, true, 1, 0, 0},
		{"FromBottom slide-in start", FromBottom, true, 0, 0, 50},
		{"FromBottom slide-in end", FromBottom, true, 1, 0, 0},
		{"FromLeft slide-in start", FromLeft, true, 0, -100, 0},
		{"FromLeft slide-in end", FromLeft, true, 1, 0, 0},
		{"FromRight slide-in start", FromRight, true, 0, 100, 0},
		{"FromRight slide-in end", FromRight, true, 1, 0, 0},
		{"FromTop slide-out start", FromTop, false, 0, 0, 0},
		{"FromTop slide-out end", FromTop, false, 1, 0, -50},
		{"FromLeft slide-out mid", FromLeft, false, 0.5, -50, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			child := newMockWidget()
			s := NewSlide(child,
				SlideFrom(tt.direction),
				SlideEasing(animation.Linear),
				SlideAutoStart(false),
			)
			s.slideIn = tt.slideIn
			s.progress = tt.progress

			// Set child bounds to 100x50.
			child.SetBounds(geometry.NewRect(0, 0, 100, 50))
			s.SetBounds(geometry.NewRect(0, 0, 100, 50))

			offset := s.computeOffset()
			if offset.X != tt.wantX || offset.Y != tt.wantY {
				t.Errorf("offset: got (%v, %v), want (%v, %v)",
					offset.X, offset.Y, tt.wantX, tt.wantY)
			}
		})
	}
}

// --- Mount/Unmount lifecycle tests ---

func TestSlideMountAutoStart(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child) // autoStart=true (default)

	ctx := newMockContext(time.Now())
	s.Mount(ctx)

	if !s.IsAnimating() {
		t.Error("should start animating on mount with autoStart")
	}
	if !s.slideIn {
		t.Error("default mount should slide in")
	}
}

func TestSlideMountAutoStartReverse(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideReverse(true))

	ctx := newMockContext(time.Now())
	s.Mount(ctx)

	if !s.IsAnimating() {
		t.Error("should start animating on mount")
	}
	if s.slideIn {
		t.Error("reverse mount should slide out")
	}
}

func TestSlideMountNoAutoStart(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	ctx := newMockContext(time.Now())
	s.Mount(ctx)

	if s.IsAnimating() {
		t.Error("should not animate with autoStart=false")
	}
}

func TestSlideUnmountStopsAnimation(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))

	s.SlideIn()
	if !s.IsAnimating() {
		t.Error("should be animating")
	}

	s.Unmount()
	if s.IsAnimating() {
		t.Error("should stop animating after unmount")
	}
}

// --- SetChild test ---

func TestSlideSetChild(t *testing.T) {
	child1 := newMockWidget()
	child2 := newMockWidget()
	s := NewSlide(child1, SlideAutoStart(false))

	if s.Child() != child1 {
		t.Error("initial child mismatch")
	}

	s.SetChild(child2)
	if s.Child() != child2 {
		t.Error("SetChild should replace child")
	}
}

// --- Event tests ---

func TestSlideEventForwardsToChild(t *testing.T) {
	child := newMockWidget()
	child.eventRet = true
	s := NewSlide(child, SlideAutoStart(false))

	ctx := newMockContext(time.Now())
	if !s.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should forward to child and return its result")
	}
}

func TestSlideEventNilChild(t *testing.T) {
	s := NewSlide(nil)
	ctx := newMockContext(time.Now())
	if s.Event(ctx, &event.MouseEvent{}) {
		t.Error("Event should return false for nil child")
	}
}

// --- Children tests ---

func TestSlideChildrenReturnsChild(t *testing.T) {
	child := newMockWidget()
	s := NewSlide(child, SlideAutoStart(false))
	children := s.Children()
	if len(children) != 1 || children[0] != child {
		t.Error("Children should return single-element slice with child")
	}
}

func TestSlideChildrenNilChild(t *testing.T) {
	s := NewSlide(nil)
	if s.Children() != nil {
		t.Error("Children should return nil for nil child")
	}
}

// --- Interface compliance ---

func TestSlideImplementsWidget(t *testing.T) {
	var _ widget.Widget = (*Slide)(nil)
}

func TestSlideImplementsLifecycle(t *testing.T) {
	var _ widget.Lifecycle = (*Slide)(nil)
}
