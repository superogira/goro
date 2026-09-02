package animation

import (
	"math"
	"testing"
	"time"
)

// mockSignal implements signalFloat32 for testing.
type mockSignal struct {
	value float32
}

func (s *mockSignal) Get() float32  { return s.value }
func (s *mockSignal) Set(v float32) { s.value = v }

func newMockSignal(v float32) *mockSignal {
	return &mockSignal{value: v}
}

func TestAnimationBasic(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	// At t=50ms, should be ~0.5.
	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig.Get()-0.5)) > 0.02 {
		t.Errorf("at 50ms: got %v, want ~0.5", sig.Get())
	}

	// At t=100ms, should complete at 1.0.
	ctrl.Tick(50 * time.Millisecond)
	if sig.Get() != 1.0 {
		t.Errorf("at 100ms: got %v, want 1.0", sig.Get())
	}

	// Should be done.
	if ctrl.HasActive() {
		t.Error("controller should have no active animations")
	}
}

func TestAnimationFromExplicit(t *testing.T) {
	sig := newMockSignal(5)
	ctrl := NewController()

	To(sig, 10.0).
		From(0.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig.Get()-5.0)) > 0.1 {
		t.Errorf("from=0, to=10, at 50ms: got %v, want ~5.0", sig.Get())
	}
}

func TestAnimationFromImplicit(t *testing.T) {
	sig := newMockSignal(5)
	ctrl := NewController()

	// No From() -> should start from current signal value (5).
	To(sig, 10.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig.Get()-7.5)) > 0.1 {
		t.Errorf("from=implicit(5), to=10, at 50ms: got %v, want ~7.5", sig.Get())
	}
}

func TestAnimationDelay(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		Duration(100 * time.Millisecond).
		Delay(50 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	// During delay, value should not change.
	ctrl.Tick(30 * time.Millisecond)
	if sig.Get() != 0 {
		t.Errorf("during delay: got %v, want 0", sig.Get())
	}

	// After delay starts (50ms delay + 50ms into animation = 100ms total).
	ctrl.Tick(70 * time.Millisecond)
	if math.Abs(float64(sig.Get()-0.5)) > 0.05 {
		t.Errorf("50ms into animation after delay: got %v, want ~0.5", sig.Get())
	}
}

func TestAnimationRepeat(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Repeat(1). // play twice total
		Start(ctrl)

	// Complete first iteration.
	ctrl.Tick(100 * time.Millisecond)
	if !ctrl.HasActive() {
		t.Error("should still be active after first iteration")
	}

	// Complete second iteration.
	ctrl.Tick(100 * time.Millisecond)
	if sig.Get() != 1.0 {
		t.Errorf("after 2 iterations: got %v, want 1.0", sig.Get())
	}
	if ctrl.HasActive() {
		t.Error("should be done after 2 iterations")
	}
}

func TestAnimationAutoReverse(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		From(0.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Repeat(1).
		AutoReverse().
		Start(ctrl)

	// First iteration: 0 -> 1
	ctrl.Tick(100 * time.Millisecond)
	// Second iteration should reverse: 1 -> 0
	ctrl.Tick(50 * time.Millisecond)
	// At midpoint of reverse, should be ~0.5.
	if math.Abs(float64(sig.Get()-0.5)) > 0.1 {
		t.Errorf("reverse midpoint: got %v, want ~0.5", sig.Get())
	}
}

func TestAnimationOnDone(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	done := false
	To(sig, 1.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		OnDone(func() { done = true }).
		Start(ctrl)

	ctrl.Tick(100 * time.Millisecond)
	if !done {
		t.Error("OnDone was not called")
	}
}

func TestAnimationOnDoneNotCalledOnCancel(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	done := false
	anim := To(sig, 1.0).
		Duration(100 * time.Millisecond).
		OnDone(func() { done = true }).
		Start(ctrl)

	anim.Cancel()
	ctrl.Tick(100 * time.Millisecond)
	if done {
		t.Error("OnDone should not be called when canceled")
	}
}

func TestAnimationZeroDuration(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		Duration(0).
		Ease(Linear).
		Start(ctrl)

	ctrl.Tick(time.Millisecond)
	if sig.Get() != 1.0 {
		t.Errorf("zero duration: got %v, want 1.0", sig.Get())
	}
}

func TestAnimationInfiniteRepeat(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		From(0.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Repeat(-1). // infinite
		Start(ctrl)

	// Should still be active after many iterations.
	for range 10 {
		ctrl.Tick(100 * time.Millisecond)
	}
	if !ctrl.HasActive() {
		t.Error("infinite repeat should still be active")
	}
}

func TestAnimationBuild(t *testing.T) {
	sig := newMockSignal(5)
	anim := To(sig, 10.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Build()

	// Build should set from to current signal value.
	if anim.from != 5 {
		t.Errorf("Build() from = %v, want 5", anim.from)
	}
	if anim.to != 10 {
		t.Errorf("Build() to = %v, want 10", anim.to)
	}
}

func TestAnimationEasing(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	To(sig, 1.0).
		From(0.0).
		Duration(100 * time.Millisecond).
		Ease(EaseInQuad).
		Start(ctrl)

	// At t=50ms with EaseInQuad, progress should be 0.25 (0.5^2).
	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig.Get()-0.25)) > 0.05 {
		t.Errorf("EaseInQuad at midpoint: got %v, want ~0.25", sig.Get())
	}
}

func TestAnimationIsDone(t *testing.T) {
	sig := newMockSignal(0)
	anim := To(sig, 1.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Build()

	if anim.isDone() {
		t.Error("animation should not be done before stepping")
	}

	anim.step(100 * time.Millisecond)
	if !anim.isDone() {
		t.Error("animation should be done after full duration")
	}
}

func TestAnimationSignalKey(t *testing.T) {
	sig := newMockSignal(0)
	anim := To(sig, 1.0).Build()
	key := anim.signalKey()
	if key == nil {
		t.Error("signalKey should not be nil")
	}
	// Key should be the signal itself.
	if key != any(sig) {
		t.Error("signalKey should be the signal")
	}
}
