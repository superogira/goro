package badge

import (
	"github.com/gogpu/ui/state"
)

// Option configures a badge during construction.
type Option func(*config)

// Count sets the badge's initial static count.
// Negative values are treated as zero.
func Count(n int) Option {
	return func(c *config) {
		c.count = n
	}
}

// CountFn sets a dynamic count function that is evaluated on each draw.
// When set, this takes precedence over the static count but not over
// a signal set via [CountSignal] or [CountReadonlySignal].
func CountFn(fn func() int) Option {
	return func(c *config) {
		c.countFn = fn
	}
}

// CountSignal binds the badge's count to a reactive signal.
// This is a one-way read binding: the widget reads the value from the signal.
// When set, the signal value takes precedence over both [CountFn] and [Count]
// but not over [CountReadonlySignal].
func CountSignal(sig state.Signal[int]) Option {
	return func(c *config) {
		c.countSignal = sig
	}
}

// CountReadonlySignal binds the badge's count to a read-only signal.
// This is useful for computed signals created via [state.NewComputed].
// When set, this takes highest precedence over all other count sources.
func CountReadonlySignal(sig state.ReadonlySignal[int]) Option {
	return func(c *config) {
		c.readonlyCountSignal = sig
	}
}

// Dot enables dot mode, rendering a small filled circle with no text.
// In dot mode the count is ignored and the badge is always visible.
func Dot(enabled bool) Option {
	return func(c *config) {
		c.dot = enabled
	}
}

// Max sets the count threshold above which the badge renders "N+".
// For example, with Max(99) a count of 100 displays as "99+".
// The default is 99. Values <= 0 are ignored.
func Max(n int) Option {
	return func(c *config) {
		c.max = n
		c.maxSet = true
	}
}

// ShowZero forces the badge to remain visible when the count is zero.
// By default a count badge with a non-positive count is hidden.
// This option has no effect in dot mode.
func ShowZero(show bool) Option {
	return func(c *config) {
		c.showZero = show
	}
}

// ColorSchemeOpt sets the color scheme for painting.
// This overrides the painter's built-in defaults.
func ColorSchemeOpt(cs BadgeColorScheme) Option {
	return func(c *config) {
		c.colorScheme = cs
	}
}

// Disabled sets the badge's disabled state, which dims its colors.
func Disabled(d bool) Option {
	return func(c *config) {
		c.disabled = d
	}
}

// DisabledFn sets a dynamic function for the disabled state.
// When set, this takes precedence over the static value but not
// over a signal set via [DisabledSignal].
func DisabledFn(fn func() bool) Option {
	return func(c *config) {
		c.disabledFn = fn
	}
}

// DisabledSignal binds the disabled state to a reactive signal.
// When set, the signal value takes precedence over both [DisabledFn]
// and [Disabled] but not over [DisabledReadonlySignal].
func DisabledSignal(sig state.Signal[bool]) Option {
	return func(c *config) {
		c.disabledSignal = sig
	}
}

// DisabledReadonlySignal binds the disabled state to a read-only signal.
// When set, this takes highest precedence over all other disabled sources.
func DisabledReadonlySignal(sig state.ReadonlySignal[bool]) Option {
	return func(c *config) {
		c.readonlyDisabledSignal = sig
	}
}

// PainterOpt sets the painter used to render the badge.
// Each design system provides its own painter. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}
