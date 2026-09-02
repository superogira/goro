package listview

import (
	"github.com/gogpu/ui/cdk"
	"github.com/gogpu/ui/state"
)

// config holds the list view's configuration, set at construction time via options.
type config struct {
	// Item data.
	itemCount               int
	itemCountFn             func() int
	itemCountSignal         state.Signal[int]
	readonlyItemCountSignal state.ReadonlySignal[int]

	// Item content — the Content[ItemContext] that renders each visible item.
	// Set via BuildItem (convenience) or ItemContent (direct Content[C]).
	itemContent cdk.Content[ItemContext]

	// Height modes (mutually exclusive; checked in order: fixed > fn > estimated).
	fixedItemHeight     float32
	itemHeightFn        func(index int) float32
	estimatedItemHeight float32

	// Selection.
	selectionMode               SelectionMode
	selectedIndex               int
	selectedIndexSignal         state.Signal[int]
	readonlySelectedIndexSignal state.ReadonlySignal[int]
	onItemClick                 func(index int)
	onSelectionChange           func(index int)

	// Disabled state.
	disabled               bool
	disabledFn             func() bool
	disabledSignal         state.Signal[bool]
	readonlyDisabledSignal state.ReadonlySignal[bool]

	// Scroll pass-through.
	scrollYSignal state.Signal[float32]
	onScroll      func(offset float32)

	// End-reached callback for infinite scroll.
	onEndReached        func()
	endReachedThreshold int

	// Visual options.
	divider bool
	painter Painter

	// Rendering tuning.
	overscan int

	// Accessibility.
	a11yLabel string
}

// defaultConfig returns a config with sensible defaults.
func defaultConfig() config {
	return config{
		selectedIndex:       -1,
		estimatedItemHeight: defaultEstimatedHeight,
		endReachedThreshold: defaultEndReachedThreshold,
		overscan:            defaultOverscan,
	}
}

// Default configuration values.
const (
	defaultEstimatedHeight     float32 = 48
	defaultEndReachedThreshold int     = 5
	defaultOverscan            int     = 3
)

// ResolvedItemCount returns the current item count.
// Priority: ReadonlySignal > Signal > Fn > Static.
func (c *config) ResolvedItemCount() int {
	if c.readonlyItemCountSignal != nil {
		return c.readonlyItemCountSignal.Get()
	}
	if c.itemCountSignal != nil {
		return c.itemCountSignal.Get()
	}
	if c.itemCountFn != nil {
		return c.itemCountFn()
	}
	return c.itemCount
}

// ResolvedSelectedIndex returns the current selected index.
// Priority: ReadonlySignal > Signal > Static.
// Returns -1 if no selection.
func (c *config) ResolvedSelectedIndex() int {
	if c.readonlySelectedIndexSignal != nil {
		return c.readonlySelectedIndexSignal.Get()
	}
	if c.selectedIndexSignal != nil {
		return c.selectedIndexSignal.Get()
	}
	return c.selectedIndex
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

// hasFixedHeight reports whether all items have a uniform fixed height.
func (c *config) hasFixedHeight() bool {
	return c.fixedItemHeight > 0
}

// hasItemHeightFn reports whether a per-item height callback is set.
func (c *config) hasItemHeightFn() bool {
	return c.itemHeightFn != nil
}
