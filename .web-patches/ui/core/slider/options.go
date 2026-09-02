package slider

import (
	"github.com/gogpu/ui/state"
)

// Option configures a slider during construction.
type Option func(*config)

// Value sets the slider's initial static value.
func Value(v float32) Option {
	return func(c *config) {
		c.value = v
	}
}

// ValueFn sets a dynamic value function that is evaluated on each draw.
// When set, this takes precedence over the static value but not over
// a signal set via [ValueSignal].
func ValueFn(fn func() float32) Option {
	return func(c *config) {
		c.valueFn = fn
	}
}

// ValueSignal binds the slider's value to a reactive signal.
// This is a TWO-WAY binding: the widget reads the value from the signal,
// and when the user drags the thumb, the new value is written back to the signal.
// When set, the signal value takes precedence over both [ValueFn] and [Value]
// but not over [ValueReadonlySignal].
func ValueSignal(sig state.Signal[float32]) Option {
	return func(c *config) {
		c.valueSignal = sig
	}
}

// ValueReadonlySignal binds the slider's value to a read-only signal.
// This is useful for computed signals created via [state.NewComputed].
// When set, this takes highest precedence over all other value sources.
func ValueReadonlySignal(sig state.ReadonlySignal[float32]) Option {
	return func(c *config) {
		c.readonlyValueSignal = sig
	}
}

// Min sets the slider's minimum value. Default is 0.
func Min(v float32) Option {
	return func(c *config) {
		c.minVal = v
	}
}

// Max sets the slider's maximum value. Default is 100.
func Max(v float32) Option {
	return func(c *config) {
		c.maxVal = v
	}
}

// Step sets the step increment. When step > 0, the value snaps to the
// nearest multiple of step within [min, max]. When step is 0 (default),
// the slider is continuous.
func Step(v float32) Option {
	return func(c *config) {
		c.step = v
	}
}

// OnChange sets the callback invoked when the slider value changes.
// The callback receives the new value.
func OnChange(fn func(float32)) Option {
	return func(c *config) {
		c.onChange = fn
	}
}

// Disabled sets the slider's disabled state. A disabled slider does not
// respond to user input and is drawn with a dimmed appearance.
func Disabled(d bool) Option {
	return func(c *config) {
		c.disabled = d
	}
}

// DisabledFn sets a dynamic function that is evaluated to determine whether
// the slider is disabled. When set, this takes precedence over the static value
// but not over a signal set via [DisabledSignal].
func DisabledFn(fn func() bool) Option {
	return func(c *config) {
		c.disabledFn = fn
	}
}

// DisabledSignal binds the slider's disabled state to a reactive signal.
// When set, the signal value takes precedence over both [DisabledFn] and [Disabled]
// but not over [DisabledReadonlySignal].
func DisabledSignal(sig state.Signal[bool]) Option {
	return func(c *config) {
		c.disabledSignal = sig
	}
}

// DisabledReadonlySignal binds the slider's disabled state to a read-only signal.
// This is useful for computed signals created via [state.NewComputed].
// When set, this takes highest precedence over all other disabled sources.
func DisabledReadonlySignal(sig state.ReadonlySignal[bool]) Option {
	return func(c *config) {
		c.readonlyDisabledSignal = sig
	}
}

// OrientationOpt sets the slider's orientation.
func OrientationOpt(o Orientation) Option {
	return func(c *config) {
		c.orientation = o
	}
}

// Marks sets the slider's tick marks.
func Marks(m []Mark) Option {
	return func(c *config) {
		c.marks = m
	}
}

// A11yHint sets the accessibility hint text for the slider.
func A11yHint(hint string) Option {
	return func(c *config) {
		c.a11yHint = hint
	}
}

// PainterOpt sets the painter used to render the slider.
// Each design system provides its own painter. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}
