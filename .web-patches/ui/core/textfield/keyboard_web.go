//go:build js && wasm

package textfield

import "syscall/js"

// applyTextInputKeyboard raises or lowers the OS virtual keyboard for the
// game's text fields on the web build. The browser platform exposes
// window.goroShowKeyboard()/goroHideKeyboard(), backed by a hidden <input>
// whose focused state makes tablets show their on-screen keyboard. The
// field's current text is seeded into the input so the input-event diff
// (append/delete) stays in sync with the field, and hint selects the
// keyboard's action-key label (next/send).
func applyTextInputKeyboard(focused bool, text, hint string) {
	global := js.Global()
	if focused {
		if fn := global.Get("goroShowKeyboard"); fn.Type() == js.TypeFunction {
			fn.Invoke(text, hint)
		}
		return
	}
	if fn := global.Get("goroHideKeyboard"); fn.Type() == js.TypeFunction {
		fn.Invoke()
	}
}
