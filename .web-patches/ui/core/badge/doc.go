// Package badge provides a small notification badge widget that displays a
// count or a status dot.
//
// A badge is typically overlaid on another widget (an icon, an avatar, a tab)
// to draw attention to unread items, pending notifications, or status. Because
// gogpu/ui favors composition, the badge itself is a self-contained leaf
// widget: position it over another widget using a stacking container rather
// than attaching it directly.
//
// Construction uses functional options for immutable configuration, while
// fluent methods handle mutable styling:
//
//	b := badge.New(
//	    badge.Count(5),
//	    badge.Max(99),
//	).Padding(2)
//
// # Modes
//
// A badge renders in one of two modes:
//
//   - Count mode (default): displays a number inside a pill shape. A single
//     digit renders as a circle; multiple digits widen into a pill. Counts
//     greater than [Max] are shown as "N+" (for example "99+").
//   - Dot mode (enabled via [Dot]): displays a small filled circle with no
//     text, used as a lightweight "has updates" indicator.
//
// # Visibility
//
// In count mode, a badge with a non-positive count is hidden (it reports a
// zero size and draws nothing) unless [ShowZero] is set. Dot badges are always
// visible.
//
// # Visual Style
//
// The visual rendering is provided by a [Painter] implementation. Each design
// system (Material 3, Fluent, Cupertino) may supply its own painter to render
// badges in the appropriate visual style.
//
// If no painter is set, [DefaultPainter] is used, which draws a minimal badge
// suitable for testing and prototyping.
//
// # Signal Binding
//
// Badge properties can be bound to reactive signals from the [state] package.
// When a signal value changes, the badge automatically reflects the new state.
//
//   - [CountSignal] / [CountReadonlySignal] -- one-way read binding for the count
//   - [CountFn] -- dynamic function evaluated on each draw
//   - [DisabledSignal] / [DisabledReadonlySignal] -- one-way read binding for disabled state
//
// Example:
//
//	unread := state.NewSignal[int](0)
//	b := badge.New(badge.CountSignal(unread))
//	unread.Set(3) // badge updates to show "3"
//
// # Accessibility
//
// Badges are display-only widgets. They do not accept focus or handle input
// events. Screen readers should announce the badge's meaning via the
// accessibility tree of the widget the badge annotates.
package badge
