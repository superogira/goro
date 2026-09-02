//go:build js && wasm

package textfield

import "syscall/js"

// notifyTextInputFocused raises the OS virtual keyboard when a text field
// gains focus on the web build. The browser platform exposes
// window.goroShowKeyboard()/goroHideKeyboard(), backed by a hidden <input>
// whose focused state makes tablets show their on-screen keyboard.
func notifyTextInputFocused(focused bool) {
	global := js.Global()
	if focused {
		if fn := global.Get("goroShowKeyboard"); fn.Type() == js.TypeFunction {
			fn.Invoke()
		}
		return
	}
	if fn := global.Get("goroHideKeyboard"); fn.Type() == js.TypeFunction {
		fn.Invoke()
	}
}
