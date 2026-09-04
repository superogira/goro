//go:build !js || !wasm

package render

// Native builds keep the OS window chrome (and its own fullscreen path), so
// the on-screen toggle is web-only.
const webFullscreenButton = false

func publishFullscreenButtonRect(_, _, _ int) {}
