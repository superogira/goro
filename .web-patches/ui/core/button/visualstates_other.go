//go:build !js || !wasm

package button

// visualStateTransitionsEnabled: native builds keep hover/press repaints —
// the raster is microseconds from local assets and the feedback matters.
const visualStateTransitionsEnabled = true
