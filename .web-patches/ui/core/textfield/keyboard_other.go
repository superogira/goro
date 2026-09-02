//go:build !js || !wasm

package textfield

// applyTextInputKeyboard is a no-op off the browser: native platforms get
// text through their real window system.
func applyTextInputKeyboard(bool, string, string) {}
