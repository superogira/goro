//go:build !js || !wasm

package gogpu

// SetBrowserVSync only affects the browser frame pacing; native builds
// configure the present mode through the surface instead.
func SetBrowserVSync(enabled bool) {}

// SetBrowserResolutionScale only affects the browser canvas; native
// surfaces follow the OS window size.
func SetBrowserResolutionScale(scale float64) {}
