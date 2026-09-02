package chip

import (
	"github.com/gogpu/ui/state"
)

// Option configures a chip during construction.
type Option func(*config)

// Label sets the chip's static label text.
func Label(s string) Option {
	return func(c *config) {
		c.label = s
	}
}

// LabelFn sets a dynamic label function evaluated on each draw.
// When set, this takes precedence over the static label but not over
// a signal set via [LabelSignal] or [LabelReadonlySignal].
func LabelFn(fn func() string) Option {
	return func(c *config) {
		c.labelFn = fn
	}
}

// LabelSignal binds the chip's label to a reactive signal.
// When set, the signal value takes precedence over both [LabelFn] and [Label]
// but not over [LabelReadonlySignal].
func LabelSignal(sig state.Signal[string]) Option {
	return func(c *config) {
		c.labelSignal = sig
	}
}

// LabelReadonlySignal binds the chip's label to a read-only signal.
// When set, this takes highest precedence over all other label sources.
func LabelReadonlySignal(sig state.ReadonlySignal[string]) Option {
	return func(c *config) {
		c.readonlyLabelSignal = sig
	}
}

// Selectable makes the chip toggle a selected state on activation
// (a filter chip). When false (the default), the chip is an action chip
// that only invokes its [OnClick] handler.
func Selectable(enabled bool) Option {
	return func(c *config) {
		c.selectable = enabled
	}
}

// Selected sets the chip's initial selected state. Only meaningful when
// [Selectable] is enabled.
func Selected(sel bool) Option {
	return func(c *config) {
		c.selected = sel
	}
}

// SelectedFn sets a dynamic selected-state function evaluated on each draw.
// Using a function (or signal) places the chip in controlled mode: the owner
// must update the source in response to [OnSelectedChanged].
func SelectedFn(fn func() bool) Option {
	return func(c *config) {
		c.selectedFn = fn
	}
}

// SelectedSignal binds the selected state to a reactive signal (controlled mode).
func SelectedSignal(sig state.Signal[bool]) Option {
	return func(c *config) {
		c.selectedSignal = sig
	}
}

// SelectedReadonlySignal binds the selected state to a read-only signal
// (controlled mode). Takes highest precedence over all other selected sources.
func SelectedReadonlySignal(sig state.ReadonlySignal[bool]) Option {
	return func(c *config) {
		c.readonlySelectedSignal = sig
	}
}

// OnClick sets the handler invoked when the chip is activated by click or
// keyboard. It fires for both action and filter chips.
func OnClick(fn func()) Option {
	return func(c *config) {
		c.onClick = fn
	}
}

// OnSelectedChanged sets the handler invoked when a selectable chip toggles.
// The argument is the new selected state. Only fires when [Selectable] is set.
func OnSelectedChanged(fn func(bool)) Option {
	return func(c *config) {
		c.onSelectedChanged = fn
	}
}

// ColorSchemeOpt sets the color scheme for painting, overriding defaults.
func ColorSchemeOpt(cs ChipColorScheme) Option {
	return func(c *config) {
		c.colorScheme = cs
	}
}

// Disabled sets the chip's disabled state.
func Disabled(d bool) Option {
	return func(c *config) {
		c.disabled = d
	}
}

// DisabledFn sets a dynamic function for the disabled state.
// When set, this takes precedence over the static value but not over
// a signal set via [DisabledSignal].
func DisabledFn(fn func() bool) Option {
	return func(c *config) {
		c.disabledFn = fn
	}
}

// DisabledSignal binds the disabled state to a reactive signal.
func DisabledSignal(sig state.Signal[bool]) Option {
	return func(c *config) {
		c.disabledSignal = sig
	}
}

// DisabledReadonlySignal binds the disabled state to a read-only signal.
// Takes highest precedence over all other disabled sources.
func DisabledReadonlySignal(sig state.ReadonlySignal[bool]) Option {
	return func(c *config) {
		c.readonlyDisabledSignal = sig
	}
}

// PainterOpt sets the painter used to render the chip. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}
