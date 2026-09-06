//go:build js && wasm

package textfield

import (
	"syscall/js"

	"github.com/gogpu/ui/geometry"
)

// applyTextInputKeyboard raises or lowers the OS virtual keyboard for the
// game's text fields on the web build. The browser platform exposes
// window.goroShowKeyboard()/goroHideKeyboard(), backed by a hidden <input>
// whose focused state makes tablets show their on-screen keyboard. The
// field's current text is seeded into the input so the input-event diff
// (append/delete) stays in sync with the field, and hint selects the
// keyboard's action-key label (next/send).
//
// bounds is the focused field's window-space rect (empty when the widget
// has not been stamped yet). The page uses it to pan the game canvas while
// the keyboard overlays the viewport, keeping the field visible above it.
func applyTextInputKeyboard(focused bool, text, hint string, bounds geometry.Rect) {
	global := js.Global()
	if focused {
		if fn := global.Get("goroShowKeyboard"); fn.Type() == js.TypeFunction {
			fn.Invoke(text, hint, bounds.Min.X, bounds.Min.Y, bounds.Width(), bounds.Height())
		}
		return
	}
	if fn := global.Get("goroHideKeyboard"); fn.Type() == js.TypeFunction {
		fn.Invoke()
	}
}
