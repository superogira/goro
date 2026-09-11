//go:build js && wasm

package render

import "syscall/js"

// webVersionBadgeReady reports whether the page provides the DOM version
// badge hook.
func webVersionBadgeReady() bool {
	return js.Global().Get("goroVersionSync").Type() == js.TypeFunction
}

// SetWebVersionBadge pushes the build label to the page's DOM badge. The
// label never changes within a session, so the first push sticks.
func SetWebVersionBadge(label string) {
	if fn := js.Global().Get("goroVersionSync"); fn.Type() == js.TypeFunction {
		fn.Invoke(label)
	}
}
