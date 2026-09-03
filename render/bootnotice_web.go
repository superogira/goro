//go:build js && wasm

package render

import "syscall/js"

// notifyBootReady marks the boot as finished in the page: index.html keeps a
// loading overlay visible until the game presents its first real frame, so
// the first page load is not a silent black screen.
func notifyBootReady() {
	js.Global().Set("goroFirstFrame", true)
}
