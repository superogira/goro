//go:build !js || !wasm

package gogpu

// compatMouseFromTouch is false on native platforms: the OS still delivers
// real mouse events for touch taps, so mapping pointer events too would
// double-fire every press.
const compatMouseFromTouch = false
