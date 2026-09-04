//go:build js && wasm

package render

import "syscall/js"

// webFullscreenButton enables the on-screen fullscreen toggle on the web
// build only. The button lives outside the widget UI layer (drawn straight
// to the frame like the FPS meter) so it stays visible with ?noui=1 and
// never participates in dirty-region repaints.
const webFullscreenButton = true

var lastFullscreenButtonRect [4]int

// publishFullscreenButtonRect exposes the button's canvas-space rect as
// window.__goroFSBtn. The browser platform layer consumes taps inside that
// rect before the game's pointer pipeline sees them and flips the fullscreen
// state synchronously in the gesture task — requestFullscreen is rejected
// outside a user activation window.
func publishFullscreenButtonRect(x, y, size int) {
	if lastFullscreenButtonRect == [4]int{x, y, size, size} {
		return
	}
	lastFullscreenButtonRect = [4]int{x, y, size, size}
	rect := js.Global().Get("Object").New()
	rect.Set("x", x)
	rect.Set("y", y)
	rect.Set("w", size)
	rect.Set("h", size)
	js.Global().Set("__goroFSBtn", rect)
}
