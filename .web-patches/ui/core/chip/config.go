package chip

import (
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// config holds the chip's configuration, set at construction time via options.
type config struct {
	label                  string
	labelFn                func() string
	labelSignal            state.Signal[string]
	readonlyLabelSignal    state.ReadonlySignal[string]
	selectable             bool
	selected               bool
	selectedFn             func() bool
	selectedSignal         state.Signal[bool]
	readonlySelectedSignal state.ReadonlySignal[bool]
	onClick                func()
	onSelectedChanged      func(bool)
	disabled               bool
	disabledFn             func() bool
	disabledSignal         state.Signal[bool]
	readonlyDisabledSignal state.ReadonlySignal[bool]
	colorScheme            ChipColorScheme
	painter                Painter
}

// ResolvedLabel returns the current display label.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedLabel() string {
	if c.readonlyLabelSignal != nil {
		return c.readonlyLabelSignal.Get()
	}
	if c.labelSignal != nil {
		return c.labelSignal.Get()
	}
	if c.labelFn != nil {
		return c.labelFn()
	}
	return c.label
}

// ResolvedSelected returns the current selected state.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedSelected() bool {
	if c.readonlySelectedSignal != nil {
		return c.readonlySelectedSignal.Get()
	}
	if c.selectedSignal != nil {
		return c.selectedSignal.Get()
	}
	if c.selectedFn != nil {
		return c.selectedFn()
	}
	return c.selected
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

// ChipColorScheme provides theme-derived colors for chip painting.
// Zero value means the painter should use its built-in defaults.
type ChipColorScheme struct {
	Background         widget.Color // unselected fill color
	Border             widget.Color // unselected outline color
	Label              widget.Color // unselected label color
	SelectedBackground widget.Color // selected fill color
	SelectedLabel      widget.Color // selected label color
	DisabledBackground widget.Color // fill color when disabled
	DisabledLabel      widget.Color // label color when disabled
}
