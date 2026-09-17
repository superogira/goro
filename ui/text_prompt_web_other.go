//go:build !js || !wasm

package ui

// textPromptWebSync is the native stub: no DOM panel exists, so callers
// fall back to the canvas window.
func textPromptWebSync(title, label, placeholder string, maxLength int, open bool) bool {
	return false
}

// drainTextPromptWebActions is the native stub for the DOM prompt queue.
func drainTextPromptWebActions() (submit string, cancelled bool) {
	return "", false
}
