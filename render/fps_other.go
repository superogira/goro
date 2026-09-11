//go:build !js || !wasm

package render

// webFPSReady is false on native builds: the FPS meter draws on canvas.
func webFPSReady() bool { return false }

// SetWebFPS is a no-op on native builds.
func SetWebFPS(text string) {}
