package animation

import "time"

// Startable is the interface for anything that can be built into an animatable.
//
// Both AnimationBuilder and SpringBuilder satisfy this via their Build methods.
// This interface is used by Sequence and Parallel to accept both types.
type Startable interface {
	buildAnimatable() animatable
}

// buildAnimatable satisfies Startable for AnimationBuilder.
func (b *AnimationBuilder) buildAnimatable() animatable {
	return b.Build()
}

// buildAnimatable satisfies Startable for SpringBuilder.
func (b *SpringBuilder) buildAnimatable() animatable {
	return b.Build()
}

// sequence plays animations one after another.
type sequence struct {
	items   []animatable
	current int
	done    bool
	onDone  func()
}

// NewSequence creates a composition that plays animations one after another.
//
// Each animation starts only after the previous one completes.
// The sequence is complete when all animations have finished.
//
// Example:
//
//	animation.NewSequence(
//	    animation.To(opacity, 1.0).Duration(200*time.Millisecond),
//	    animation.To(scale, 1.0).Duration(300*time.Millisecond),
//	).Start(ctrl)
func NewSequence(items ...Startable) *SequenceBuilder {
	anims := make([]animatable, len(items))
	for i, item := range items {
		anims[i] = item.buildAnimatable()
	}
	return &SequenceBuilder{
		seq: &sequence{items: anims},
	}
}

// step advances the sequence by dt. Returns true when all items are done.
func (s *sequence) step(dt time.Duration) bool {
	if s.done {
		return true
	}

	for s.current < len(s.items) {
		finished := s.items[s.current].step(dt)
		if !finished {
			return false
		}
		s.current++
		// Remaining dt after completion is consumed by the next item,
		// but for simplicity we start the next item on the next frame.
		// This avoids complexity of tracking exact completion time.
		dt = 0
	}

	s.done = true
	if s.onDone != nil {
		s.onDone()
	}
	return true
}

// isDone reports whether the sequence is complete.
func (s *sequence) isDone() bool {
	return s.done
}

// signalKey returns nil because sequences don't target a single signal.
func (s *sequence) signalKey() any {
	return nil
}

// SequenceBuilder constructs a sequence composition.
type SequenceBuilder struct {
	seq *sequence
}

// OnDone sets a callback invoked when the sequence completes.
func (b *SequenceBuilder) OnDone(fn func()) *SequenceBuilder {
	b.seq.onDone = fn
	return b
}

// Start registers the sequence with the controller.
func (b *SequenceBuilder) Start(ctrl *Controller) {
	ctrl.addComposition(b.seq)
}

// parallel plays animations simultaneously.
type parallel struct {
	items  []animatable
	done   bool
	onDone func()
}

// NewParallel creates a composition that plays animations simultaneously.
//
// All animations start at the same time. The parallel composition is complete
// when the longest animation finishes.
//
// Example:
//
//	animation.NewParallel(
//	    animation.To(opacity, 1.0).Duration(200*time.Millisecond),
//	    animation.To(translateY, 0).Duration(300*time.Millisecond),
//	).Start(ctrl)
func NewParallel(items ...Startable) *ParallelBuilder {
	anims := make([]animatable, len(items))
	for i, item := range items {
		anims[i] = item.buildAnimatable()
	}
	return &ParallelBuilder{
		par: &parallel{items: anims},
	}
}

// step advances all items by dt. Returns true when all items are done.
func (p *parallel) step(dt time.Duration) bool {
	if p.done {
		return true
	}

	allDone := true
	for _, item := range p.items {
		if !item.isDone() {
			finished := item.step(dt)
			if !finished {
				allDone = false
			}
		}
	}

	if allDone {
		p.done = true
		if p.onDone != nil {
			p.onDone()
		}
	}

	return p.done
}

// isDone reports whether all parallel items are complete.
func (p *parallel) isDone() bool {
	return p.done
}

// signalKey returns nil because parallel compositions don't target a single signal.
func (p *parallel) signalKey() any {
	return nil
}

// ParallelBuilder constructs a parallel composition.
type ParallelBuilder struct {
	par *parallel
}

// OnDone sets a callback invoked when all parallel animations complete.
func (b *ParallelBuilder) OnDone(fn func()) *ParallelBuilder {
	b.par.onDone = fn
	return b
}

// Start registers the parallel composition with the controller.
func (b *ParallelBuilder) Start(ctrl *Controller) {
	ctrl.addComposition(b.par)
}

// Verify implementations.
var (
	_ animatable = (*sequence)(nil)
	_ animatable = (*parallel)(nil)
	_ Startable  = (*AnimationBuilder)(nil)
	_ Startable  = (*SpringBuilder)(nil)
)
