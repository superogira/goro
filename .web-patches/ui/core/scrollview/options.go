package scrollview

import (
	"github.com/gogpu/ui/state"
)

// Option configures a scroll view during construction.
type Option func(*config)

// DirectionOpt sets the scroll direction.
// Default is [Vertical].
func DirectionOpt(d ScrollDirection) Option {
	return func(c *config) {
		c.direction = d
	}
}

// ScrollbarOpt sets the scrollbar visibility mode.
// Default is [ScrollbarAuto].
func ScrollbarOpt(v ScrollbarVisibility) Option {
	return func(c *config) {
		c.scrollbar = v
	}
}

// ScrollX sets the initial horizontal scroll offset.
func ScrollX(x float32) Option {
	return func(c *config) {
		c.scrollX = x
	}
}

// ScrollY sets the initial vertical scroll offset.
func ScrollY(y float32) Option {
	return func(c *config) {
		c.scrollY = y
	}
}

// ScrollStep sets the number of pixels scrolled per wheel tick.
// Default is 40 pixels.
func ScrollStep(step float32) Option {
	return func(c *config) {
		c.scrollStep = step
	}
}

// ScrollXSignal binds the horizontal scroll offset to a reactive signal.
// This is a TWO-WAY binding: the widget reads the offset from the signal,
// and when the user scrolls, the new offset is written back to the signal.
func ScrollXSignal(sig state.Signal[float32]) Option {
	return func(c *config) {
		c.scrollXSignal = sig
	}
}

// ScrollXReadonlySignal binds the horizontal scroll offset to a read-only signal.
// When set, this takes highest precedence over all other scrollX sources.
func ScrollXReadonlySignal(sig state.ReadonlySignal[float32]) Option {
	return func(c *config) {
		c.readonlyScrollXSignal = sig
	}
}

// ScrollYSignal binds the vertical scroll offset to a reactive signal.
// This is a TWO-WAY binding: the widget reads the offset from the signal,
// and when the user scrolls, the new offset is written back to the signal.
func ScrollYSignal(sig state.Signal[float32]) Option {
	return func(c *config) {
		c.scrollYSignal = sig
	}
}

// ScrollYReadonlySignal binds the vertical scroll offset to a read-only signal.
// When set, this takes highest precedence over all other scrollY sources.
func ScrollYReadonlySignal(sig state.ReadonlySignal[float32]) Option {
	return func(c *config) {
		c.readonlyScrollYSignal = sig
	}
}

// OnScroll sets the callback invoked when the scroll position changes.
// The callback receives the new (scrollX, scrollY) offsets.
func OnScroll(fn func(x, y float32)) Option {
	return func(c *config) {
		c.onScroll = fn
	}
}

// PainterOpt sets the painter used to render the scrollbar.
// Each design system provides its own painter. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}
