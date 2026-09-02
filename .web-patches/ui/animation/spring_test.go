package animation

import (
	"math"
	"testing"
	"time"
)

func TestSpringConvergence(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	// Tick for 2 seconds — should converge.
	for range 120 {
		ctrl.Tick(16 * time.Millisecond)
	}

	if math.Abs(float64(sig.Get()-100)) > 1.0 {
		t.Errorf("spring did not converge: got %v, want ~100", sig.Get())
	}
	if ctrl.HasActive() {
		t.Error("spring should be settled")
	}
}

func TestSpringBouncy(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingHighBouncy).
		Start(ctrl)

	// Track if spring overshoots.
	maxVal := float32(0)
	for range 200 {
		ctrl.Tick(16 * time.Millisecond)
		v := sig.Get()
		if v > maxVal {
			maxVal = v
		}
	}

	// High bouncy should overshoot target.
	if maxVal <= 100 {
		t.Errorf("bouncy spring should overshoot: max = %v", maxVal)
	}

	// But should eventually converge.
	if math.Abs(float64(sig.Get()-100)) > 1.0 {
		t.Errorf("bouncy spring did not converge: got %v, want ~100", sig.Get())
	}
}

func TestSpringNoBounce(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy). // critically damped
		Start(ctrl)

	// Critically damped should not overshoot (much).
	maxVal := float32(0)
	for range 200 {
		ctrl.Tick(16 * time.Millisecond)
		v := sig.Get()
		if v > maxVal {
			maxVal = v
		}
	}

	// Allow tiny overshoot from Euler integration.
	if maxVal > 102 {
		t.Errorf("critically damped spring should not overshoot significantly: max = %v", maxVal)
	}
}

func TestSpringVelocityPreservation(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	// Start first spring.
	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	// Let it build up velocity.
	for range 5 {
		ctrl.Tick(16 * time.Millisecond)
	}

	// Re-target to a new value — velocity should be preserved.
	s2 := SpringTo(sig, 200.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	// The new spring should have inherited velocity from the old one.
	if s2.velocity == 0 {
		t.Error("new spring should have inherited velocity from canceled spring")
	}
}

func TestSpringInitialVelocity(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		InitialVelocity(500).
		Start(ctrl)

	ctrl.Tick(16 * time.Millisecond)
	// With initial velocity, should move faster initially.
	if sig.Get() < 5 {
		t.Errorf("spring with initial velocity should move quickly: got %v", sig.Get())
	}
}

func TestSpringOnDone(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	done := false
	SpringTo(sig, 100.0).
		Stiffness(StiffnessHigh).
		DampingRatio(DampingNoBouncy).
		OnDone(func() { done = true }).
		Start(ctrl)

	// Tick until settled.
	for range 200 {
		ctrl.Tick(16 * time.Millisecond)
	}

	if !done {
		t.Error("OnDone was not called after spring settled")
	}
}

func TestSpringCancel(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	s := SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	ctrl.Tick(16 * time.Millisecond)
	s.Cancel()
	ctrl.Tick(16 * time.Millisecond)

	if ctrl.HasActive() {
		t.Error("controller should have no active after cancel")
	}
}

func TestSpringDtCap(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	// Large dt should be capped, not cause instability.
	ctrl.Tick(500 * time.Millisecond)
	val := sig.Get()
	if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
		t.Error("large dt caused instability")
	}
}

func TestSpringIsDone(t *testing.T) {
	sig := newMockSignal(0)
	s := SpringTo(sig, 0.0). // target == current -> should settle immediately
					Stiffness(StiffnessMedium).
					DampingRatio(DampingNoBouncy).
					Build()

	// Position equals target with zero velocity -> should settle on first step.
	s.step(16 * time.Millisecond)
	if !s.isDone() {
		t.Error("spring should be done when already at target")
	}
}

func TestSpringVelocity(t *testing.T) {
	sig := newMockSignal(0)
	ctrl := NewController()

	s := SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Start(ctrl)

	ctrl.Tick(16 * time.Millisecond)
	if s.Velocity() == 0 {
		t.Error("spring should have non-zero velocity after first tick")
	}
}

func TestSpringBuildDampingConversion(t *testing.T) {
	sig := newMockSignal(0)
	s := SpringTo(sig, 100.0).
		Stiffness(1500).
		DampingRatio(1.0).
		Mass(1.0).
		Build()

	// d = 2 * zeta * sqrt(k * m) = 2 * 1.0 * sqrt(1500) ≈ 77.46
	expected := float32(2 * math.Sqrt(1500))
	if math.Abs(float64(s.damping-expected)) > 0.1 {
		t.Errorf("damping = %v, want ~%v", s.damping, expected)
	}
}

func TestSpringZeroDt(t *testing.T) {
	sig := newMockSignal(0)
	s := SpringTo(sig, 100.0).
		Stiffness(StiffnessMedium).
		DampingRatio(DampingNoBouncy).
		Build()

	// Zero dt should not advance.
	done := s.step(0)
	if done {
		t.Error("zero dt should not finish spring")
	}
	if sig.Get() != 0 {
		t.Errorf("zero dt should not change position: got %v", sig.Get())
	}
}
