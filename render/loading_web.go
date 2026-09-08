//go:build js && wasm

package render

import "syscall/js"

// The Now Loading cover's label renders as page DOM: browser text is crisp
// at every scale and the overlay composites without touching the game
// canvas. The background image and dim stay on the canvas.

var webLoadingShown bool

// SetWebLoading shows or hides the DOM loading label; memoized so the JS
// bridge only fires on transitions.
func SetWebLoading(visible bool) {
	if webLoadingShown == visible {
		return
	}
	webLoadingShown = visible
	fn := js.Global().Get("goroLoadingSet")
	if fn.Type() != js.TypeFunction {
		return
	}
	fn.Invoke(visible)
}
