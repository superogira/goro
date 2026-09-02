package badge

import (
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// config holds the badge's configuration, set at construction time via options.
type config struct {
	count                  int
	countFn                func() int
	countSignal            state.Signal[int]
	readonlyCountSignal    state.ReadonlySignal[int]
	dot                    bool
	max                    int
	maxSet                 bool // true if max was explicitly set via option
	showZero               bool
	disabled               bool
	disabledFn             func() bool
	disabledSignal         state.Signal[bool]
	readonlyDisabledSignal state.ReadonlySignal[bool]
	colorScheme            BadgeColorScheme
	painter                Painter
}

// ResolvedCount returns the current count, clamped to be non-negative.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedCount() int {
	var v int
	switch {
	case c.readonlyCountSignal != nil:
		v = c.readonlyCountSignal.Get()
	case c.countSignal != nil:
		v = c.countSignal.Get()
	case c.countFn != nil:
		v = c.countFn()
	default:
		v = c.count
	}
	if v < 0 {
		return 0
	}
	return v
}

// ResolvedMax returns the count threshold above which the badge renders "N+".
// When max was not explicitly set, the default is used.
func (c *config) ResolvedMax() int {
	if !c.maxSet || c.max <= 0 {
		return defaultMax
	}
	return c.max
}

// ResolvedDisabled returns the current disabled state.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedDisabled() bool {
	if c.readonlyDisabledSignal != nil {
		return c.readonlyDisabledSignal.Get()
	}
	if c.disabledSignal != nil {
		return c.disabledSignal.Get()
	}
	if c.disabledFn != nil {
		return c.disabledFn()
	}
	return c.disabled
}

// BadgeColorScheme provides theme-derived colors for badge painting.
// Zero value means the painter should use its built-in defaults.
type BadgeColorScheme struct {
	Background         widget.Color // pill/dot fill color
	Label              widget.Color // count text color
	DisabledBackground widget.Color // fill color when disabled
	DisabledLabel      widget.Color // text color when disabled
}
