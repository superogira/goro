// Package chip provides a compact chip widget for tags, filters, and inline
// actions, following the Material 3 chip guidelines.
//
// A chip is a small, interactive, pill-shaped element. It can act as a simple
// action (assist/suggestion chip) or as a toggleable filter (filter chip).
//
// Construction uses functional options for immutable configuration, while
// fluent methods handle mutable styling:
//
//	c := chip.New(
//	    chip.Label("In stock"),
//	    chip.Selectable(true),
//	    chip.OnSelectedChanged(func(sel bool) { applyFilter(sel) }),
//	).Padding(2)
//
// # Modes
//
//   - Action chip (default): clicking invokes the [OnClick] handler. Use for
//     assist or suggestion chips.
//   - Filter chip ([Selectable] is true): clicking toggles the selected state
//     and invokes [OnSelectedChanged]. The chip renders a filled background
//     when selected and an outlined background when not.
//
// # Selection write-back
//
// On activation a selectable chip toggles its selected state and writes the
// new value back to the appropriate source, matching the checkbox widget:
//
//   - With a writable [SelectedSignal], the new value is written to the signal
//     (two-way binding).
//   - With only a static value, the chip updates its own internal state.
//   - With a read-only source ([SelectedFn] or [SelectedReadonlySignal]), the
//     chip leaves the source untouched; the owner is responsible for updating
//     it in response to [OnSelectedChanged].
//
// # Visual Style
//
// The visual rendering is provided by a [Painter] implementation. Each design
// system (Material 3, Fluent, Cupertino) may supply its own painter. If no
// painter is set, [DefaultPainter] is used.
//
// # Signal Binding
//
// Chip properties can be bound to reactive signals from the [state] package:
//
//   - [LabelSignal] / [LabelReadonlySignal] / [LabelFn] -- the label text
//   - [SelectedSignal] / [SelectedReadonlySignal] / [SelectedFn] -- selected state
//   - [DisabledSignal] / [DisabledReadonlySignal] / [DisabledFn] -- disabled state
//
// # Accessibility
//
// A chip is focusable when visible, enabled, and not disabled. It activates on
// the Enter and Space keys, matching the button widget.
package chip
