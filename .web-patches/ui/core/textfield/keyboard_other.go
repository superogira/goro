//go:build !js || !wasm

package textfield

import "github.com/gogpu/ui/geometry"

// applyTextInputKeyboard is a no-op off the browser: native platforms get
// text through their real window system.
func applyTextInputKeyboard(bool, string, string, geometry.Rect) {}
