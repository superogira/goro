// Package listview provides a virtualized list widget that renders only
// visible items, enabling efficient display of large datasets.
//
// Construction uses functional options for immutable configuration:
//
//	lv := listview.New(
//	    listview.ItemCount(len(data)),
//	    listview.FixedItemHeight(48),
//	    listview.BuildItem(func(ctx listview.ItemContext) widget.Widget {
//	        return primitives.NewText(data[ctx.Index].Name)
//	    }),
//	)
//
// # Height Modes
//
// Three height modes are available, each with different performance
// characteristics:
//
//   - [FixedItemHeight] -- O(1) fast path, all items share the same height
//   - [ItemHeightFn] -- known heights via callback, O(log n) lookup
//   - Default (lazy measurement) -- items measured on scroll, uses estimated average
//
// # ScrollView Composition
//
// ListView composes an internal [scrollview.Widget] for scroll handling.
// All scroll features (wheel, keyboard PageUp/Down/Home/End, scrollbar
// thumb drag, momentum, signals) are provided automatically.
//
// # Selection
//
// Optional item selection is supported via [SelectedIndex] or
// [SelectedIndexSignal]. Selection state is passed to the builder
// callback through [ItemContext.Selected].
//
// # Signal Binding
//
// List properties can be bound to reactive signals from the [state] package.
// When a signal value changes, the list automatically reflects the new state.
//
//   - [ItemCountSignal] -- one-way binding for item count
//   - [SelectedIndexSignal] -- TWO-WAY binding for selected item
//   - [ScrollYSignal] -- TWO-WAY binding for scroll offset (via internal ScrollView)
//   - [DisabledSignal] -- one-way binding for disabled state
//
// Example with signals:
//
//	count := state.NewSignal(100)
//	selected := state.NewSignal(-1)
//
//	lv := listview.New(
//	    listview.ItemCountSignal(count),
//	    listview.FixedItemHeight(56),
//	    listview.SelectedIndexSignal(selected),
//	    listview.BuildItem(func(ctx listview.ItemContext) widget.Widget {
//	        return buildContactRow(contacts[ctx.Index], ctx.Selected)
//	    }),
//	    listview.OnEndReached(func() { loadMoreContacts() }),
//	    listview.Divider(true),
//	)
//
// # Visual Style
//
// The visual rendering of list-specific elements (dividers, empty state,
// item backgrounds, selection highlights) is provided by a [Painter]
// implementation. Each design system (Material 3, Fluent, Cupertino)
// supplies its own painter.
//
// If no painter is set, [DefaultPainter] is used, which draws minimal
// visuals suitable for testing and prototyping.
//
// # Accessibility
//
// ListView implements [a11y.Accessible] with [a11y.RoleList]. Visible
// items are exposed as [a11y.RoleListItem] children. Keyboard navigation
// with arrow keys moves selection between items.
//
// # Focus
//
// ListView implements [widget.Focusable] and participates in tab navigation.
// When focused, arrow keys move selection and Home/End jump to first/last item.
package listview
