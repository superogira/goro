package animation

import (
	"math"
	"testing"
	"time"
)

func TestControllerTickReturnsFalseWhenEmpty(t *testing.T) {
	ctrl := NewController()
	active := ctrl.Tick(16 * time.Millisecond)
	if active {
		t.Error("empty controller should return false")
	}
}

func TestControllerHasActive(t *testing.T) {
	ctrl := NewController()
	if ctrl.HasActive() {
		t.Error("empty controller should have no active")
	}

	sig := newMockSignal(0)
	To(sig, 1.0).Duration(100 * time.Millisecond).Start(ctrl)
	if !ctrl.HasActive() {
		t.Error("controller should have active after adding animation")
	}
}

func TestControllerAutoCancel(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	// Start first animation.
	To(sig, 1.0).
		From(0.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)
	midVal := sig.Get()
	if math.Abs(float64(midVal-0.5)) > 0.05 {
		t.Errorf("first animation at 50ms: got %v, want ~0.5", midVal)
	}

	// Start second animation on same signal — should auto-cancel first.
	To(sig, 0.0).
		Duration(100 * time.Millisecond).
		Ease(Linear).
		Start(ctrl)

	// Second animation starts from current value (~0.5) to 0.
	ctrl.Tick(50 * time.Millisecond)
	// Should be roughly midway between ~0.5 and 0 = ~0.25.
	if math.Abs(float64(sig.Get()-0.25)) > 0.1 {
		t.Errorf("second animation at 50ms: got %v, want ~0.25", sig.Get())
	}
}

func TestControllerCancelAll(t *testing.T) {
	ctrl := NewController()

	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	To(sig1, 1.0).Duration(100 * time.Millisecond).Start(ctrl)
	To(sig2, 1.0).Duration(100 * time.Millisecond).Start(ctrl)

	if !ctrl.HasActive() {
		t.Error("should have active animations")
	}

	ctrl.CancelAll()
	if ctrl.HasActive() {
		t.Error("should have no active after CancelAll")
	}
}

func TestControllerCancelSpecific(t *testing.T) {
	ctrl := NewController()

	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)
	To(sig1, 1.0).Duration(100 * time.Millisecond).Start(ctrl)
	To(sig2, 1.0).Duration(100 * time.Millisecond).Start(ctrl)

	ctrl.Cancel(sig1)

	// sig2 should still be active.
	active := ctrl.Tick(50 * time.Millisecond)
	if !active {
		t.Error("sig2 animation should still be active")
	}
	if sig1.Get() != 0 {
		t.Errorf("sig1 should not have been updated: got %v", sig1.Get())
	}
}

func TestControllerMultipleSignals(t *testing.T) {
	ctrl := NewController()

	sig1 := newMockSignal(0)
	sig2 := newMockSignal(0)

	To(sig1, 1.0).From(0.0).Duration(100 * time.Millisecond).Ease(Linear).Start(ctrl)
	To(sig2, 2.0).From(0.0).Duration(100 * time.Millisecond).Ease(Linear).Start(ctrl)

	ctrl.Tick(50 * time.Millisecond)
	if math.Abs(float64(sig1.Get()-0.5)) > 0.05 {
		t.Errorf("sig1 at 50ms: got %v, want ~0.5", sig1.Get())
	}
	if math.Abs(float64(sig2.Get()-1.0)) > 0.05 {
		t.Errorf("sig2 at 50ms: got %v, want ~1.0", sig2.Get())
	}
}

func TestControllerSpringAutoCancel(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	// Start tween, then spring on same signal.
	To(sig, 1.0).Duration(100 * time.Millisecond).Start(ctrl)
	SpringTo(sig, 2.0).Stiffness(StiffnessHigh).DampingRatio(DampingNoBouncy).Start(ctrl)

	// Should have only one active (the spring replaced the tween).
	count := len(ctrl.active)
	if count != 1 {
		t.Errorf("expected 1 active animation, got %d", count)
	}
}

func TestControllerCleanup(t *testing.T) {
	ctrl := NewController()

	sig := newMockSignal(0)
	To(sig, 1.0).Duration(50 * time.Millisecond).Ease(Linear).Start(ctrl)

	// Complete the animation.
	ctrl.Tick(50 * time.Millisecond)

	if ctrl.HasActive() {
		t.Error("completed animation should be cleaned up")
	}
}

func TestControllerNewReturnsNonNil(t *testing.T) {
	ctrl := NewController()
	if ctrl == nil {
		t.Error("NewController should return non-nil")
	}
}
