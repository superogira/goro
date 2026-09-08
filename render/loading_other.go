//go:build !js || !wasm

package render

// SetWebLoading is the native no-op twin of the DOM loading hook.
func SetWebLoading(visible bool) {}
