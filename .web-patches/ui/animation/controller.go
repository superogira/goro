package animation

import "time"

// Controller manages active animations and provides the Tick entry point.
//
// The Controller owns a map of signal -> active animation for auto-cancel.
// When a new animation targets a signal that already has an active animation,
// the old animation is automatically canceled.
//
// Controller is designed to be owned by a window and accessed via widget.Context.
// It is NOT thread-safe; all calls must happen on the UI thread.
type Controller struct {
	// active animations indexed by signal identity for O(1) auto-cancel lookup.
	active map[any]animatable

	// compositions (Sequence/Parallel) that don't target a single signal.
	compositions []animatable
}

// NewController creates a new animation controller.
func NewController() *Controller {
	return &Controller{
		active: make(map[any]animatable),
	}
}

// Tick advances all active animations by dt.
//
// Returns true if any animations are still running. The caller should request
// a new frame when this returns true:
//
//	active := ctrl.Tick(dt)
//	if active { window.RequestRedraw() }
func (c *Controller) Tick(dt time.Duration) bool {
	// Tick signal-keyed animations.
	for key, anim := range c.active {
		if anim.step(dt) {
			delete(c.active, key)
		}
	}

	// Tick compositions.
	n := 0
	for _, comp := range c.compositions {
		if !comp.step(dt) {
			c.compositions[n] = comp
			n++
		}
	}
	// Clear references to allow GC.
	for i := n; i < len(c.compositions); i++ {
		c.compositions[i] = nil
	}
	c.compositions = c.compositions[:n]

	return len(c.active) > 0 || len(c.compositions) > 0
}

// HasActive reports whether any animations are running.
func (c *Controller) HasActive() bool {
	return len(c.active) > 0 || len(c.compositions) > 0
}

// CancelAll stops all animations immediately without calling OnDone callbacks.
func (c *Controller) CancelAll() {
	for key := range c.active {
		delete(c.active, key)
	}
	c.compositions = c.compositions[:0]
}

// Cancel stops the animation targeting the given signal.
//
// The signal parameter should be the same signal passed to To() or SpringTo().
func (c *Controller) Cancel(signal signalFloat32) {
	delete(c.active, any(signal))
}

// add registers a tween animation with auto-cancel.
func (c *Controller) add(a *Animation) {
	key := a.signalKey()
	if key == nil {
		return
	}
	// Auto-cancel existing animation on the same signal.
	if existing, ok := c.active[key]; ok {
		cancelAnimatable(existing)
	}
	c.active[key] = a
}

// addSpring registers a spring animation with auto-cancel and velocity transfer.
func (c *Controller) addSpring(s *Spring) {
	key := s.signalKey()
	if key == nil {
		return
	}
	// Auto-cancel existing animation and transfer velocity if it was a spring.
	if existing, ok := c.active[key]; ok {
		switch old := existing.(type) {
		case *Spring:
			// Velocity preservation: transfer velocity from the canceled spring.
			s.velocity = old.Velocity()
			old.Cancel()
		case *Animation:
			old.Cancel()
		}
	}
	c.active[key] = s
}

// cancelAnimatable cancels an animatable without calling its OnDone callback.
func cancelAnimatable(a animatable) {
	switch v := a.(type) {
	case *Animation:
		v.Cancel()
	case *Spring:
		v.Cancel()
	}
}

// addComposition registers a composition (Sequence/Parallel) that does not
// target a single signal.
func (c *Controller) addComposition(comp animatable) {
	c.compositions = append(c.compositions, comp)
}
