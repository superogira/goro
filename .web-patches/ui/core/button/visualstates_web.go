//go:build js && wasm

package button

// visualStateTransitionsEnabled controls whether hover/press state changes
// repaint the button. Disabled on the web build: every tap runs the button
// state machine through hover→press→release, and each transition re-rasters
// the button on the CPU — a visible hitch on tablets. The button still
// tracks state and fires clicks; it just always paints its normal look.
// Paint-state gating in ButtonPainter is unnecessary: with redraws skipped,
// the painter never re-runs with a different state.
const visualStateTransitionsEnabled = false
