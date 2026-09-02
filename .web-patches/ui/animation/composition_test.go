package animation

import (
	"math"
	"testing"
	"time"
)

func TestSequenceBasic(t *testing.T) {
	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	ctrl := NewController()

	NewSequence(
		To(sig1, 1.0).From(0.0).Duration(100*time.Millisecond).Ease(Linear),
		To(sig2, 1.0).From(0.0).Duration(100*time.Millisecond).Ease(Linear),
	).Start(ctrl)

	// First animation should run.
	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig1.Get()-0.5)) > 0.05 {
		t.Errorf("seq: sig1 at 50ms = %v, want ~0.5", sig1.Get())
	}
	if sig2.Get() != 0 {
		t.Errorf("seq: sig2 should not have started: got %v", sig2.Get())
	}

	// Complete first animation.
	ctrl.Tick(50 * time.Millisecond)
	if sig1.Get() != 1.0 {
		t.Errorf("seq: sig1 at 100ms = %v, want 1.0", sig1.Get())
	}

	// Second animation should now run.
	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig2.Get()-0.5)) > 0.05 {
		t.Errorf("seq: sig2 at 150ms = %v, want ~0.5", sig2.Get())
	}

	// Complete second animation.
	ctrl.Tick(50 * time.Millisecond)
	if sig2.Get() != 1.0 {
		t.Errorf("seq: sig2 at 200ms = %v, want 1.0", sig2.Get())
	}

	if ctrl.HasActive() {
		t.Error("sequence should be done")
	}
}

func TestSequenceOnDone(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	done := false
	NewSequence(
		To(sig, 1.0).From(0.0).Duration(50 * time.Millisecond).Ease(Linear),
	).OnDone(func() { done = true }).Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)
	if !done {
		t.Error("sequence OnDone was not called")
	}
}

func TestSequenceEmpty(t *testing.T) {
	ctrl := NewController()

	done := false
	NewSequence().OnDone(func() { done = true }).Start(ctrl)

	ctrl.Tick(time.Millisecond)
	if !done {
		t.Error("empty sequence should complete immediately")
	}
}

func TestParallelBasic(t *testing.T) {
	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	ctrl := NewController()

	NewParallel(
		To(sig1, 1.0).From(0.0).Duration(100*time.Millisecond).Ease(Linear),
		To(sig2, 2.0).From(0.0).Duration(200*time.Millisecond).Ease(Linear),
	).Start(ctrl)

	// Both should run simultaneously.
	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig1.Get()-0.5)) > 0.05 {
		t.Errorf("par: sig1 at 50ms = %v, want ~0.5", sig1.Get())
	}
	if math.Abs(float64(sig2.Get()-0.5)) > 0.05 {
		t.Errorf("par: sig2 at 50ms = %v, want ~0.5", sig2.Get())
	}

	// sig1 completes at 100ms.
	ctrl.Tick(50 * time.Millisecond)
	if sig1.Get() != 1.0 {
		t.Errorf("par: sig1 at 100ms = %v, want 1.0", sig1.Get())
	}

	// sig2 still running.
	if !ctrl.HasActive() {
		t.Error("parallel should still be active (sig2 not done)")
	}

	// sig2 completes at 200ms.
	ctrl.Tick(100 * time.Millisecond)
	if sig2.Get() != 2.0 {
		t.Errorf("par: sig2 at 200ms = %v, want 2.0", sig2.Get())
	}

	if ctrl.HasActive() {
		t.Error("parallel should be done")
	}
}

func TestParallelOnDone(t *testing.T) {
	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	ctrl := NewController()

	done := false
	NewParallel(
		To(sig1, 1.0).From(0.0).Duration(50*time.Millisecond).Ease(Linear),
		To(sig2, 1.0).From(0.0).Duration(100*time.Millisecond).Ease(Linear),
	).OnDone(func() { done = true }).Start(ctrl)

	// After first animation completes.
	ctrl.Tick(50 * time.Millisecond)
	if done {
		t.Error("parallel OnDone should not fire until all complete")
	}

	// After second completes.
	ctrl.Tick(50 * time.Millisecond)
	if !done {
		t.Error("parallel OnDone was not called")
	}
}

func TestParallelEmpty(t *testing.T) {
	ctrl := NewController()

	done := false
	NewParallel().OnDone(func() { done = true }).Start(ctrl)

	ctrl.Tick(time.Millisecond)
	if !done {
		t.Error("empty parallel should complete immediately")
	}
}

func TestSequenceWithSpring(t *testing.T) {
	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	ctrl := NewController()

	NewSequence(
		To(sig1, 1.0).From(0.0).Duration(50*time.Millisecond).Ease(Linear),
		SpringTo(sig2, 100.0).Stiffness(StiffnessHigh).DampingRatio(DampingNoBouncy),
	).Start(ctrl)

	// Complete first animation.
	ctrl.Tick(50 * time.Millisecond)
	if sig1.Get() != 1.0 {
		t.Errorf("seq with spring: sig1 = %v, want 1.0", sig1.Get())
	}

	// Spring should start running.
	for range 200 {
		ctrl.Tick(16 * time.Millisecond)
	}
	if math.Abs(float64(sig2.Get()-100)) > 1.0 {
		t.Errorf("seq with spring: sig2 = %v, want ~100", sig2.Get())
	}
}

func TestCompositionWithController(t *testing.T) {
	ctrl := NewController()

	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)

	// Direct animation + composition should coexist.
	To(sig1, 1.0).From(0.0).Duration(50 * time.Millisecond).Ease(Linear).Start(ctrl)

	NewSequence(
		To(sig2, 1.0).From(0.0).Duration(50 * time.Millisecond).Ease(Linear),
	).Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)

	if sig1.Get() != 1.0 {
		t.Errorf("direct animation: sig1 = %v, want 1.0", sig1.Get())
	}
	if sig2.Get() != 1.0 {
		t.Errorf("composition: sig2 = %v, want 1.0", sig2.Get())
	}
}

func TestControllerCancelAllIncludesCompositions(t *testing.T) {
	ctrl := NewController()

	sig := newMockSignal(0)
	NewSequence(
		To(sig, 1.0).From(0.0).Duration(1000 * time.Millisecond).Ease(Linear),
	).Start(ctrl)

	if !ctrl.HasActive() {
		t.Error("should have active composition")
	}

	ctrl.CancelAll()
	if ctrl.HasActive() {
		t.Error("CancelAll should clear compositions too")
	}
}

func TestStartableSatisfied(t *testing.T) {
	// Compile-time check only - these are verified by var _ lines in composition.go.
	sig := newMockSignal(0)
	var _ Startable = To(sig, 1.0)
	var _ Startable = SpringTo(sig, 1.0)
}
