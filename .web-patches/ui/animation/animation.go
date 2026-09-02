package animation

import "time"

// repeatInfinite signals infinite repetition.
const repeatInfinite = -1

// animatable is the internal interface for anything the Controller can tick.
type animatable interface {
	// step advances the animation by dt. Returns true when finished.
	step(dt time.Duration) bool

	// isDone reports whether the animation has completed.
	isDone() bool

	// signalKey returns the signal identity for auto-cancel, or nil.
	signalKey() any
}

// Animation represents a running tween animation that drives a Signal[float32].
//
// Animations are created with the [To] builder and started with [AnimationBuilder.Start].
// The animation updates its target signal each frame via the Controller's Tick method.
type Animation struct {
	signal signalFloat32
	from   float32
	to     float32

	duration time.Duration
	delay    time.Duration
	easing   Easing
	repeat   int  // 0 = once, repeatInfinite = infinite, N = repeat N more times
	reverse  bool // auto-reverse after each iteration
	onDone   func()

	// internal state
	elapsed time.Duration
	started bool
	done    bool
	iter    int
	forward bool // current direction (for auto-reverse)
}

// signalFloat32 is a minimal interface for a writable float32 signal.
// This avoids importing state package directly in the animation struct,
// keeping the dependency lightweight.
type signalFloat32 interface {
	Get() float32
	Set(float32)
}

// step advances the animation by dt. Returns true when finished.
func (a *Animation) step(dt time.Duration) bool {
	if a.done {
		return true
	}

	a.elapsed += dt

	// Handle delay.
	if a.elapsed < a.delay {
		return false
	}

	if !a.started {
		a.started = true
		a.forward = true
	}

	// Compute progress within current iteration.
	activeTime := a.elapsed - a.delay
	progress := float32(1)
	if a.duration > 0 {
		progress = float32(activeTime) / float32(a.duration)
	}

	// Handle iteration boundary.
	if progress >= 1 {
		done, newProgress := a.advanceIteration(activeTime)
		if done {
			return true
		}
		progress = newProgress
	}

	a.applyProgress(progress)
	return false
}

// advanceIteration handles the transition between iterations.
// Returns (true, 0) if the animation is done, or (false, newProgress) to continue.
func (a *Animation) advanceIteration(activeTime time.Duration) (bool, float32) {
	a.iter++

	// Check if we should continue repeating.
	if a.repeat != repeatInfinite && a.iter > a.repeat {
		a.applyProgress(1)
		a.finish()
		return true, 0
	}

	// Start next iteration.
	if a.reverse {
		a.forward = !a.forward
	}
	// Carry over excess time.
	excess := activeTime - a.duration
	a.elapsed = a.delay + excess
	if a.duration > 0 {
		return false, float32(excess) / float32(a.duration)
	}
	return false, 1
}

// applyProgress sets the signal value for the given linear progress [0,1].
func (a *Animation) applyProgress(progress float32) {
	// Clamp to [0,1].
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	// Apply direction for auto-reverse.
	t := progress
	if !a.forward {
		t = 1 - progress
	}

	// Apply easing.
	if a.easing != nil {
		t = a.easing(t)
	}

	// Interpolate and set.
	value := a.from + (a.to-a.from)*t
	a.signal.Set(value)
}

// finish marks the animation as done and calls onDone if set.
func (a *Animation) finish() {
	a.done = true
	if a.onDone != nil {
		a.onDone()
	}
}

// isDone reports whether the animation has completed.
func (a *Animation) isDone() bool {
	return a.done
}

// signalKey returns the signal identity for auto-cancel.
func (a *Animation) signalKey() any {
	return a.signal
}

// Cancel stops the animation immediately without calling OnDone.
func (a *Animation) Cancel() {
	a.done = true
}

// AnimationBuilder constructs an Animation using the builder pattern.
//
// Example:
//
//	animation.To(opacity, 1.0).
//	    From(0.0).
//	    Duration(300 * time.Millisecond).
//	    Ease(animation.M3Standard).
//	    Start(ctrl)
type AnimationBuilder struct {
	anim    *Animation
	fromSet bool
}

// To creates a new AnimationBuilder that animates the signal to the target value.
//
// If From is not called, the animation starts from the signal's current value
// at the time Start is called.
func To(signal signalFloat32, target float32) *AnimationBuilder {
	return &AnimationBuilder{
		anim: &Animation{
			signal:   signal,
			to:       target,
			duration: DurationMedium2, // 300ms default
			easing:   M3Standard,
		},
	}
}

// From sets the starting value. If not called, starts from signal.Get().
func (b *AnimationBuilder) From(value float32) *AnimationBuilder {
	b.anim.from = value
	b.fromSet = true
	return b
}

// Duration sets the animation duration. Default is 300ms (M3 Medium2).
func (b *AnimationBuilder) Duration(d time.Duration) *AnimationBuilder {
	b.anim.duration = d
	return b
}

// Delay sets a delay before the animation starts.
func (b *AnimationBuilder) Delay(d time.Duration) *AnimationBuilder {
	b.anim.delay = d
	return b
}

// Ease sets the easing function. Default is M3Standard.
func (b *AnimationBuilder) Ease(e Easing) *AnimationBuilder {
	b.anim.easing = e
	return b
}

// Repeat sets the number of additional repetitions.
//
// 0 means play once (default). Pass -1 for infinite repetition.
func (b *AnimationBuilder) Repeat(count int) *AnimationBuilder {
	b.anim.repeat = count
	return b
}

// AutoReverse makes the animation reverse direction after each iteration.
func (b *AnimationBuilder) AutoReverse() *AnimationBuilder {
	b.anim.reverse = true
	return b
}

// OnDone sets a callback invoked when the animation completes.
// Not called if the animation is canceled.
func (b *AnimationBuilder) OnDone(fn func()) *AnimationBuilder {
	b.anim.onDone = fn
	return b
}

// Build returns the configured Animation without starting it.
//
// If From was not called, the from value is set to the signal's current value.
func (b *AnimationBuilder) Build() *Animation {
	a := b.anim
	if !b.fromSet {
		a.from = a.signal.Get()
	}
	return a
}

// Start builds the animation and registers it with the controller.
//
// If From was not called, the animation starts from the signal's current value.
// If another animation is already running on the same signal, it is auto-canceled.
func (b *AnimationBuilder) Start(ctrl *Controller) *Animation {
	a := b.Build()
	ctrl.add(a)
	return a
}

// Verify Animation implements animatable.
var _ animatable = (*Animation)(nil)
