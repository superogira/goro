//go:build js && wasm

package ui

import (
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/client"
)

// loginWebEnabled reports whether the page provides the DOM login form.
func loginWebEnabled() bool {
	return js.Global().Get("goroLoginSync").Type() == js.TypeFunction
}

// loginWebSync pushes the seeded credentials to the page form.
func (w *LoginWindow) loginWebSync(ctx client.Context) {
	fn := js.Global().Get("goroLoginSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("user", w.Username)
	obj.Set("pass", w.Password)
	fn.Invoke(obj)
}

// handleLoginWebAction services one drained "login:" action. The DOM form
// reports its field values ahead of the submit so the game's callback
// reads them exactly as the canvas fields would.
func (w *LoginWindow) handleLoginWebAction(ctx client.Context, action string) {
	switch {
	case strings.HasPrefix(action, "login:user:"):
		w.Username = strings.TrimPrefix(action, "login:user:")
	case strings.HasPrefix(action, "login:pass:"):
		w.Password = strings.TrimPrefix(action, "login:pass:")
	case action == "login:submit":
		if w.callbacks.OnSubmit != nil {
			w.callbacks.OnSubmit()
		}
	}
}
