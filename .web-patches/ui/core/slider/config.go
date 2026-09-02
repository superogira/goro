package slider

import (
	"github.com/gogpu/ui/state"
)

// config holds the slider's configuration, set at construction time via options.
type config struct {
	value                  float32
	valueFn                func() float32
	valueSignal            state.Signal[float32]
	readonlyValueSignal    state.ReadonlySignal[float32]
	minVal                 float32
	maxVal                 float32
	step                   float32 // 0 = continuous
	onChange               func(float32)
	disabled               bool
	disabledFn             func() bool
	disabledSignal         state.Signal[bool]
	readonlyDisabledSignal state.ReadonlySignal[bool]
	orientation            Orientation
	marks                  []Mark
	a11yHint               string
	painter                Painter
}

// ResolvedValue returns the current slider value.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedValue() float32 {
	if c.readonlyValueSignal != nil {
		return c.readonlyValueSignal.Get()
	}
	if c.valueSignal != nil {
		return c.valueSignal.Get()
	}
	if c.valueFn != nil {
		return c.valueFn()
	}
	return c.value
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
