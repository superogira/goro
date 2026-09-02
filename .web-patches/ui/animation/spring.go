package animation

import (
	"math"
	"time"
)

// Spring default convergence thresholds (pixel-based).
const (
	defaultRestDelta float32 = 0.5  // Sub-pixel position threshold
	defaultRestSpeed float32 = 10.0 // Pixels per second velocity threshold
	defaultMass      float32 = 1.0
	maxDT            float32 = 0.033 // 33ms cap to prevent instability
	subStepDT        float32 = 0.001 // 1ms fixed sub-step for Euler stability
)

// Spring represents a running spring animation that drives a Signal[float32].
//
// It uses the damped harmonic oscillator model:
//
//	F = -k*x - d*v
//	a = F / m
//
// where k is stiffness, d is damping coefficient, m is mass, x is displacement
// from target, and v is velocity.
//
// Convergence is detected using the dual-threshold approach: the spring is
// settled when both |position - target| < restDelta AND |velocity| < restSpeed.
type Spring struct {
	signal signalFloat32
	target float32

	mass      float32
	stiffness float32
	damping   float32

	// Convergence thresholds
	restDelta float32
	restSpeed float32

	// State
	position float32
	velocity float32
	done     bool
	onDone   func()
}

// step advances the spring simulation by dt. Returns true when settled.
func (s *Spring) step(dt time.Duration) bool {
	if s.done {
		return true
	}

	dtSec := float32(dt.Seconds())

	// Cap dt to prevent instability from pauses/debugger.
	if dtSec > maxDT {
		dtSec = maxDT
	}
	if dtSec <= 0 {
		return false
	}

	// Sub-stepped Euler integration for stability with high stiffness.
	// Using fixed sub-steps of 1ms prevents instability that occurs when
	// dt * sqrt(stiffness/mass) > 2 (Euler stability limit).
	remaining := dtSec
	for remaining > 0 {
		step := subStepDT
		if step > remaining {
			step = remaining
		}
		remaining -= step

		displacement := s.position - s.target
		springForce := -s.stiffness * displacement
		dampingForce := -s.damping * s.velocity
		acceleration := (springForce + dampingForce) / s.mass

		s.velocity += acceleration * step
		s.position += s.velocity * step
	}

	// Update signal.
	s.signal.Set(s.position)

	// Dual-threshold convergence check.
	positionDelta := s.position - s.target
	if positionDelta < 0 {
		positionDelta = -positionDelta
	}
	velAbs := s.velocity
	if velAbs < 0 {
		velAbs = -velAbs
	}

	if positionDelta < s.restDelta && velAbs < s.restSpeed {
		// Snap to target.
		s.position = s.target
		s.velocity = 0
		s.signal.Set(s.target)
		s.done = true
		if s.onDone != nil {
			s.onDone()
		}
		return true
	}

	return false
}

// isDone reports whether the spring has settled.
func (s *Spring) isDone() bool {
	return s.done
}

// signalKey returns the signal identity for auto-cancel.
func (s *Spring) signalKey() any {
	return s.signal
}

// Velocity returns the current velocity of the spring.
//
// This is used for velocity preservation when re-targeting a spring.
func (s *Spring) Velocity() float32 {
	return s.velocity
}

// Cancel stops the spring immediately without calling OnDone.
func (s *Spring) Cancel() {
	s.done = true
}

// SpringBuilder constructs a Spring using the builder pattern.
//
// Example:
//
//	animation.SpringTo(position, 200.0).
//	    Stiffness(animation.StiffnessMedium).
//	    DampingRatio(animation.DampingNoBouncy).
//	    Start(ctrl)
type SpringBuilder struct {
	spring       *Spring
	dampingRatio float32
	dampingSet   bool
}

// SpringTo creates a new SpringBuilder that animates the signal to the target.
//
// The spring starts from the signal's current value with zero initial velocity.
// If a previous spring on the same signal is canceled via auto-cancel,
// velocity is automatically transferred.
func SpringTo(signal signalFloat32, target float32) *SpringBuilder {
	return &SpringBuilder{
		spring: &Spring{
			signal:    signal,
			target:    target,
			mass:      defaultMass,
			stiffness: StiffnessMedium,
			restDelta: defaultRestDelta,
			restSpeed: defaultRestSpeed,
			position:  signal.Get(),
		},
		dampingRatio: DampingNoBouncy,
		dampingSet:   true,
	}
}

// Stiffness sets the spring constant k. Default is StiffnessMedium (1500).
func (b *SpringBuilder) Stiffness(k float32) *SpringBuilder {
	b.spring.stiffness = k
	return b
}

// DampingRatio sets the damping ratio. Default is DampingNoBouncy (1.0).
//
// Presets: DampingHighBouncy (0.2), DampingMediumBouncy (0.5),
// DampingLowBouncy (0.75), DampingNoBouncy (1.0).
func (b *SpringBuilder) DampingRatio(ratio float32) *SpringBuilder {
	b.dampingRatio = ratio
	b.dampingSet = true
	return b
}

// Mass sets the mass. Default is 1.0. Higher mass = slower response.
func (b *SpringBuilder) Mass(m float32) *SpringBuilder {
	b.spring.mass = m
	return b
}

// InitialVelocity sets the initial velocity. Default is 0.
func (b *SpringBuilder) InitialVelocity(v float32) *SpringBuilder {
	b.spring.velocity = v
	return b
}

// RestDelta sets the position convergence threshold. Default is 0.5 (sub-pixel).
func (b *SpringBuilder) RestDelta(d float32) *SpringBuilder {
	b.spring.restDelta = d
	return b
}

// RestSpeed sets the velocity convergence threshold. Default is 10.0 (px/s).
func (b *SpringBuilder) RestSpeed(s float32) *SpringBuilder {
	b.spring.restSpeed = s
	return b
}

// OnDone sets a callback invoked when the spring settles.
func (b *SpringBuilder) OnDone(fn func()) *SpringBuilder {
	b.spring.onDone = fn
	return b
}

// Build returns the configured Spring without starting it.
func (b *SpringBuilder) Build() *Spring {
	s := b.spring
	// Convert damping ratio to damping coefficient: d = 2 * zeta * sqrt(k * m)
	if b.dampingSet {
		s.damping = 2 * b.dampingRatio * float32(math.Sqrt(float64(s.stiffness*s.mass)))
	}
	return s
}

// Start builds the spring and registers it with the controller.
//
// If another animation is running on the same signal, it is auto-canceled.
// If the canceled animation was a Spring, its velocity is transferred.
func (b *SpringBuilder) Start(ctrl *Controller) *Spring {
	s := b.Build()
	ctrl.addSpring(s)
	return s
}

// Verify Spring implements animatable.
var _ animatable = (*Spring)(nil)
