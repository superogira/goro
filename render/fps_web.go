//go:build js && wasm

package render

import "syscall/js"

// webFPSReady reports whether the page provides the DOM FPS hook.
func webFPSReady() bool {
	return js.Global().Get("goroFpsSync").Type() == js.TypeFunction
}

// SetWebFPS pushes the FPS meter text to the page. The DOM meter stays
// crisp at any resolution scale, unlike the canvas text overlay.
func SetWebFPS(text string) {
	if fn := js.Global().Get("goroFpsSync"); fn.Type() == js.TypeFunction {
		fn.Invoke(text)
	}
}
