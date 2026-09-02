// Package slider provides a draggable slider widget for selecting a value
// from a continuous or discrete range.
//
// Construction uses functional options for immutable configuration,
// while fluent methods handle mutable styling:
//
//	s := slider.New(
//	    slider.Min(0),
//	    slider.Max(100),
//	    slider.Value(50),
//	    slider.OnChange(handleChange),
//	).Padding(8)
//
// # Visual Style
//
// The visual rendering is provided by a [Painter] implementation.
// Each design system (Material 3, Fluent, Cupertino) supplies its own
// painter to render sliders in the appropriate visual style.
//
// If no painter is set, [DefaultPainter] is used, which draws a minimal
// slider suitable for testing and prototyping.
//
// # Orientation
//
// Two orientations are available:
//   - [Horizontal] (default) -- thumb moves left-to-right
//   - [Vertical] -- thumb moves bottom-to-top
//
// # Value Range
//
// The slider operates within [Min, Max] bounds. If Step is set to a value
// greater than zero, the value snaps to the nearest step. When Step is zero,
// the slider is continuous.
//
// # Marks
//
// Optional [Mark] values can annotate specific positions on the track
// (e.g. tick marks or labeled positions). Marks are purely visual and
// do not constrain the slider value.
//
// # Interaction
//
// Sliders respond to:
//   - Mouse click on the track to jump to a value
//   - Mouse drag on the thumb to continuously adjust the value
//   - Keyboard: Left/Down to decrease, Right/Up to increase,
//     Home for minimum, End for maximum, Page Up/Down for larger steps
//
// Disabled sliders ignore all interaction and are drawn with a dimmed
// appearance.
//
// # Signal Binding
//
// Slider properties can be bound to reactive signals from the [state] package.
// When a signal value changes, the slider automatically reflects the new state.
//
//   - [ValueSignal] -- TWO-WAY binding: reads value from signal AND writes back on change
//   - [DisabledSignal] -- one-way binding for the disabled state
//
// Example:
//
//	volume := state.NewSignal[float32](50)
//	s := slider.New(
//	    slider.Min(0),
//	    slider.Max(100),
//	    slider.ValueSignal(volume),
//	    slider.OnChange(func(v float32) {
//	        fmt.Printf("volume: %.0f\n", v)
//	    }),
//	)
//
// # Focus
//
// Sliders implement [widget.Focusable] and participate in tab navigation.
// A focus ring is drawn around the thumb when the slider has keyboard focus.
package slider
