// Package scrollview provides a scrollable container widget for clipping
// and navigating content that exceeds the viewport size.
//
// Construction uses functional options for immutable configuration:
//
//	sv := scrollview.New(content,
//	    scrollview.DirectionOpt(scrollview.Vertical),
//	    scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
//	    scrollview.OnScroll(handleScroll),
//	)
//
// # Scroll Direction
//
// Three directions are available:
//   - [Vertical] (default) -- content scrolls up/down
//   - [Horizontal] -- content scrolls left/right
//   - [Both] -- content scrolls in both directions
//
// # Scrollbar Visibility
//
// Scrollbar display is controlled by [ScrollbarVisibility]:
//   - [ScrollbarAuto] (default) -- show scrollbar when content overflows
//   - [ScrollbarAlways] -- always show scrollbar
//   - [ScrollbarNever] -- never show scrollbar
//
// # Visual Style
//
// The scrollbar rendering is provided by a [Painter] implementation.
// Each design system (Material 3, Fluent, Cupertino) supplies its own
// painter to render scrollbars in the appropriate visual style.
//
// If no painter is set, [DefaultPainter] is used, which draws minimal
// scrollbars suitable for testing and prototyping.
//
// # Interaction
//
// ScrollView responds to:
//   - Mouse wheel for vertical/horizontal scrolling
//   - Mouse drag on scrollbar thumb to scroll
//   - Keyboard: Up/Down arrows for small steps, Page Up/Page Down for
//     viewport-sized steps, Home/End to scroll to top/bottom
//
// # Signal Binding
//
// Scroll positions can be bound to reactive signals from the [state] package.
// When a signal value changes, the scroll position automatically reflects
// the new state:
//
//   - [ScrollXSignal] -- TWO-WAY binding for horizontal scroll offset
//   - [ScrollYSignal] -- TWO-WAY binding for vertical scroll offset
//
// Example:
//
//	scrollY := state.NewSignal[float32](0)
//	sv := scrollview.New(content,
//	    scrollview.ScrollYSignal(scrollY),
//	    scrollview.OnScroll(func(x, y float32) {
//	        fmt.Printf("scroll: %.0f, %.0f\n", x, y)
//	    }),
//	)
//
// # Focus
//
// ScrollView implements [widget.Focusable] and participates in tab
// navigation. When focused, keyboard navigation is available.
package scrollview
