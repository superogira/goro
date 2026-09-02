package listview

import (
	"github.com/gogpu/ui/cdk"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

// Option configures a list view during construction.
type Option func(*config)

// --- Item Count ---

// ItemCount sets the total number of items in the list.
func ItemCount(n int) Option {
	return func(c *config) {
		c.itemCount = n
	}
}

// ItemCountFn sets a dynamic function that returns the item count.
// When set, this takes precedence over [ItemCount] but not over [ItemCountSignal].
func ItemCountFn(fn func() int) Option {
	return func(c *config) {
		c.itemCountFn = fn
	}
}

// ItemCountSignal binds the item count to a reactive signal.
// When set, the signal takes precedence over [ItemCountFn] and [ItemCount]
// but not over [ItemCountReadonlySignal].
func ItemCountSignal(sig state.Signal[int]) Option {
	return func(c *config) {
		c.itemCountSignal = sig
	}
}

// ItemCountReadonlySignal binds the item count to a read-only signal.
// When set, this takes highest precedence over all other item count sources.
func ItemCountReadonlySignal(sig state.ReadonlySignal[int]) Option {
	return func(c *config) {
		c.readonlyItemCountSignal = sig
	}
}

// --- Item Builder ---

// BuildItem sets the callback that creates a widget for each visible item.
// The callback receives an [ItemContext] with the item's index and state.
// This is the primary convenience API for providing list content.
//
// Internally, the function is wrapped into a [cdk.FuncContent] so that the
// list view uniformly operates on [cdk.Content] for item rendering.
func BuildItem(fn func(ctx ItemContext) widget.Widget) Option {
	return func(c *config) {
		c.itemContent = cdk.FuncContent[ItemContext]{Fn: fn}
	}
}

// ItemContent sets the [cdk.Content] used to render each visible item.
// This is the enterprise API for providing list content — use it when you
// need reusable content implementations or progressive complexity
// (string → function → widget).
//
// For most use cases, [BuildItem] is simpler and sufficient.
func ItemContent(content cdk.Content[ItemContext]) Option {
	return func(c *config) {
		c.itemContent = content
	}
}

// --- Height Modes ---

// FixedItemHeight sets a uniform height for all items.
// This enables the O(1) fast path with no per-item measurement.
// Mutually exclusive with [ItemHeightFn].
func FixedItemHeight(h float32) Option {
	return func(c *config) {
		c.fixedItemHeight = h
	}
}

// ItemHeightFn sets a callback that returns the height for the item at
// the given index. Heights are cached after first call, enabling O(log n)
// lookup via binary search on cumulative offsets.
// Mutually exclusive with [FixedItemHeight].
func ItemHeightFn(fn func(index int) float32) Option {
	return func(c *config) {
		c.itemHeightFn = fn
	}
}

// EstimatedItemHeight sets the estimated height for items that have not
// been measured yet. Only used when neither [FixedItemHeight] nor
// [ItemHeightFn] is set. Default: 48 pixels.
func EstimatedItemHeight(h float32) Option {
	return func(c *config) {
		c.estimatedItemHeight = h
	}
}

// --- Selection ---

// SelectionModeOpt sets the selection mode for the list.
// Default is [SelectionNone].
func SelectionModeOpt(mode SelectionMode) Option {
	return func(c *config) {
		c.selectionMode = mode
	}
}

// SelectedIndex sets the initially selected item index.
// Use -1 for no selection.
func SelectedIndex(index int) Option {
	return func(c *config) {
		c.selectedIndex = index
	}
}

// SelectedIndexSignal binds the selected index to a reactive signal.
// This is a TWO-WAY binding: the widget reads from the signal, and
// when the user selects an item, the new index is written back.
func SelectedIndexSignal(sig state.Signal[int]) Option {
	return func(c *config) {
		c.selectedIndexSignal = sig
	}
}

// SelectedIndexReadonlySignal binds the selected index to a read-only signal.
// When set, this takes highest precedence over all other selection sources.
func SelectedIndexReadonlySignal(sig state.ReadonlySignal[int]) Option {
	return func(c *config) {
		c.readonlySelectedIndexSignal = sig
	}
}

// OnItemClick sets the callback invoked when an item is clicked.
func OnItemClick(fn func(index int)) Option {
	return func(c *config) {
		c.onItemClick = fn
	}
}

// OnSelectionChange sets the callback invoked when the selected item changes.
func OnSelectionChange(fn func(index int)) Option {
	return func(c *config) {
		c.onSelectionChange = fn
	}
}

// --- Disabled State ---

// Disabled sets the list view's disabled state.
func Disabled(d bool) Option {
	return func(c *config) {
		c.disabled = d
	}
}

// DisabledFn sets a dynamic function for the disabled state.
// When set, this takes precedence over [Disabled] but not over [DisabledSignal].
func DisabledFn(fn func() bool) Option {
	return func(c *config) {
		c.disabledFn = fn
	}
}

// DisabledSignal binds the disabled state to a reactive signal.
// When set, the signal takes precedence over [DisabledFn] and [Disabled]
// but not over [DisabledReadonlySignal].
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

// --- Scroll ---

// ScrollYSignal binds the vertical scroll offset to a reactive signal.
// This is a TWO-WAY binding passed through to the internal ScrollView.
func ScrollYSignal(sig state.Signal[float32]) Option {
	return func(c *config) {
		c.scrollYSignal = sig
	}
}

// OnScroll sets the callback invoked when the scroll position changes.
func OnScroll(fn func(offset float32)) Option {
	return func(c *config) {
		c.onScroll = fn
	}
}

// --- Infinite Scroll ---

// OnEndReached sets the callback invoked when the user scrolls near the
// end of the list. Useful for infinite scroll / pagination.
// The threshold is configurable via [EndReachedThreshold].
func OnEndReached(fn func()) Option {
	return func(c *config) {
		c.onEndReached = fn
	}
}

// EndReachedThreshold sets how many items from the end triggers [OnEndReached].
// Default: 5 items.
func EndReachedThreshold(n int) Option {
	return func(c *config) {
		c.endReachedThreshold = n
	}
}

// --- Visual Options ---

// Divider enables drawing a divider between list items.
// The divider is drawn by the [Painter].
func Divider(enabled bool) Option {
	return func(c *config) {
		c.divider = enabled
	}
}

// Overscan sets the number of extra items to render above and below the
// visible viewport. Higher values reduce blank flashes during fast scroll
// at the cost of rendering more items. Default: 3 items.
func Overscan(n int) Option {
	return func(c *config) {
		c.overscan = n
	}
}

// PainterOpt sets the painter used to render list-specific visuals.
// Each design system provides its own painter. If not set,
// [DefaultPainter] is used.
func PainterOpt(p Painter) Option {
	return func(c *config) {
		c.painter = p
	}
}

// --- Accessibility ---

// A11yLabel sets the accessibility label for the list.
func A11yLabel(label string) Option {
	return func(c *config) {
		c.a11yLabel = label
	}
}
