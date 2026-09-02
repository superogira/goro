//go:build !js || !wasm

package textfield

// notifyTextInputFocused is a no-op off the browser: native platforms get
// text through their real window system.
func notifyTextInputFocused(bool, string) {}
