//go:build js && wasm

package ui

import (
	"fmt"
	"sync/atomic"
	"syscall/js"
)

// The in-canvas console rasterizes the whole message list on every append
// (wrap, layout, glyph rendering, texture upload). That is a 15-80ms CPU
// spike per received chat line — invisible on desktop, a visible hitch on
// tablets. On the web build the dormant chat log therefore lives in the
// page as plain DOM text: the browser composites it on its own threads and
// the game canvas pays nothing. The canvas console still renders while the
// player is actively typing (input focus, history, IME), where the raster
// cost is a one-off on open rather than a per-message stutter.

// consoleWebLogEnabled reports whether the dormant chat log renders as page
// DOM instead of canvas widgets.
func consoleWebLogEnabled() bool {
	return js.Global().Get("goroConsoleSync").Type() == js.TypeFunction
}

// consoleWebSync pushes the full visible message list (newest last) and the
// console's active flag to the page hook window.goroConsoleSync. The page
// shows the log only while the console is dormant; the full-list resync
// keeps DOM and canvas interchangeable on focus changes.
func consoleWebSync(active bool, lines []ConsoleMessage) {
	fn := js.Global().Get("goroConsoleSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	arr := js.Global().Get("Array").New(len(lines))
	for i, line := range lines {
		arr.SetIndex(i, js.Global().Get("Array").New(
			line.Text,
			fmt.Sprintf("rgba(%d,%d,%d,%d)", line.Color.R, line.Color.G, line.Color.B, line.Color.A),
		))
	}
	fn.Invoke(active, arr)
}

// consoleTapPending marshals a DOM tap onto the game goroutine: the JS
// callback cannot touch game state directly without racing the frame loop,
// so it just raises a flag that UpdatePresentation consumes.
var consoleTapPending int32

var consoleTapHookInstalled bool

// consoleWebInstallTapHook exposes window.goroConsoleTap for the page: the
// dormant DOM log calls it when tapped (tablets have no Enter key).
func consoleWebInstallTapHook() {
	if consoleTapHookInstalled {
		return
	}
	consoleTapHookInstalled = true
	js.Global().Set("goroConsoleTap", js.FuncOf(func(this js.Value, args []js.Value) any {
		atomic.AddInt32(&consoleTapPending, 1)
		return nil
	}))
}

// consoleWebConsumeTap reports and clears a pending DOM tap.
func consoleWebConsumeTap() bool {
	return atomic.SwapInt32(&consoleTapPending, 0) > 0
}
