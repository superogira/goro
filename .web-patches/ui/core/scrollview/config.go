package scrollview

import (
	"github.com/gogpu/ui/state"
)

// config holds the scroll view's configuration, set at construction time via options.
type config struct {
	direction  ScrollDirection
	scrollbar  ScrollbarVisibility
	scrollX    float32
	scrollY    float32
	scrollStep float32 // pixels per wheel tick; 0 = defaultScrollStep

	scrollXSignal         state.Signal[float32]
	readonlyScrollXSignal state.ReadonlySignal[float32]
	scrollYSignal         state.Signal[float32]
	readonlyScrollYSignal state.ReadonlySignal[float32]

	onScroll func(x, y float32)
	painter  Painter
}

// ResolvedScrollX returns the current horizontal scroll offset.
// Priority: ReadonlySignal > Signal > Static.
func (c *config) ResolvedScrollX() float32 {
	if c.readonlyScrollXSignal != nil {
		return c.readonlyScrollXSignal.Get()
	}
	if c.scrollXSignal != nil {
		return c.scrollXSignal.Get()
	}
	return c.scrollX
}

// ResolvedScrollY returns the current vertical scroll offset.
// Priority: ReadonlySignal > Signal > Static.
func (c *config) ResolvedScrollY() float32 {
	if c.readonlyScrollYSignal != nil {
		return c.readonlyScrollYSignal.Get()
	}
	if c.scrollYSignal != nil {
		return c.scrollYSignal.Get()
	}
	return c.scrollY
}

// resolvedScrollStep returns the configured scroll step or the default.
func (c *config) resolvedScrollStep() float32 {
	if c.scrollStep > 0 {
		return c.scrollStep
	}
	return defaultScrollStep
}
