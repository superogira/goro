package tabview

import "github.com/gogpu/ui/state"

// config holds the tabview's configuration, set at construction time via options.
type config struct {
	tabs                   []Tab
	position               TabPosition
	selected               int
	selectedSignal         state.Signal[int]
	readonlySelectedSignal state.ReadonlySignal[int]
	closeable              bool
	onSelect               func(index int)
	onClose                func(index int)
	painter                Painter
}

// ResolvedSelected returns the current selected tab index.
// Priority: ReadonlySignal > Signal > Static.
func (c *config) ResolvedSelected() int {
	if c.readonlySelectedSignal != nil {
		return c.readonlySelectedSignal.Get()
	}
	if c.selectedSignal != nil {
		return c.selectedSignal.Get()
	}
	return c.selected
}

// setSelected updates the selected index in all tiers.
// It sets the signal value if bound, and the static value otherwise.
func (c *config) setSelected(idx int) {
	c.selected = idx
	if c.selectedSignal != nil {
		c.selectedSignal.Set(idx)
	}
}
