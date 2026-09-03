//go:build js && wasm

package gogpu

// compatMouseFromTouch reports whether touch/pen pointers should also feed
// the legacy mouse handlers. On the browser the platform preventDefaults
// pointerdown, which suppresses the compatibility mouse events entirely —
// without this mapping, taps would never reach mouse-driven UI.
const compatMouseFromTouch = true
