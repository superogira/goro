//go:build js && wasm

package ui

import "syscall/js"

// DOM twin of the in-world chat room board tooltip: hovering a board whose
// title was trimmed by the pill shows the full label near the cursor.
// Display-only — the tooltip never takes pointer events, so no action hook
// is needed.

// BoardTipWebAvailable reports whether the page provides the tooltip.
func BoardTipWebAvailable() bool {
	return js.Global().Get("goroBoardTipSync").Type() == js.TypeFunction
}

// BoardTipWebSync shows (text != "") or hides the tooltip. x/y are the
// cursor's screen coordinates; the page anchors the bubble near them.
func BoardTipWebSync(text string, x, y int) {
	sync := js.Global().Get("goroBoardTipSync")
	if sync.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("text", text)
	obj.Set("x", x)
	obj.Set("y", y)
	obj.Set("show", text != "")
	sync.Invoke(obj)
}
