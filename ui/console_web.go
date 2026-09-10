//go:build js && wasm

package ui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
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

// consoleWebSync pushes the visible message list (newest last), the
// console's active flag, and the current draft text to the page hook
// window.goroConsoleSync. The DOM panel owns both the log and the input
// field on web; the draft keeps the DOM input and the game's history
// navigation in step.
func consoleWebSync(active bool, draft string, lines []ConsoleMessage) {
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
	fn.Invoke(active, draft, arr)
}

var consoleActionHookInstalled bool

// consoleWebInstallActionHook exposes window.goroConsoleAction: the page's
// DOM input field reports typing, submissions, cancel, and history walks
// through the shared action queue.
func consoleWebInstallActionHook() {
	if consoleActionHookInstalled {
		return
	}
	consoleActionHookInstalled = true
	js.Global().Set("goroConsoleAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "console:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
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

// consoleMessageQueue holds messages injected from the page until the game
// goroutine drains them.
var consoleMessageQueue = struct {
	sync.Mutex
	messages []ConsoleMessage
}{}

var consoleMessageHookInstalled bool

// consoleWebInstallMessageHook exposes window.goroConsoleMessage(text, cssColor)
// so automated checks (and page scripting) can inject incoming chat lines
// without a second client.
func consoleWebInstallMessageHook() {
	if consoleMessageHookInstalled {
		return
	}
	consoleMessageHookInstalled = true
	js.Global().Set("goroConsoleMessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 1 || args[0].Type() != js.TypeString || args[0].String() == "" {
			return nil
		}
		color := color.RGBA{R: 252, G: 221, B: 128, A: 255}
		if len(args) >= 2 && args[1].Type() == js.TypeString {
			if parsed, ok := parseCSSColor(args[1].String()); ok {
				color = parsed
			}
		}
		consoleMessageQueue.Lock()
		consoleMessageQueue.messages = append(consoleMessageQueue.messages, ConsoleMessage{Text: args[0].String(), Color: color})
		consoleMessageQueue.Unlock()
		return nil
	}))
}

// consoleWebDrainMessages hands queued page messages to the game goroutine.
func consoleWebDrainMessages() []ConsoleMessage {
	consoleMessageQueue.Lock()
	defer consoleMessageQueue.Unlock()
	if len(consoleMessageQueue.messages) == 0 {
		return nil
	}
	out := consoleMessageQueue.messages
	consoleMessageQueue.messages = nil
	return out
}

// parseCSSColor understands the #rgb/#rrggbb/rgba(r,g,b,a) forms this
// bridge emits.
func parseCSSColor(s string) (color.RGBA, bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		hex := strings.TrimPrefix(s, "#")
		switch len(hex) {
		case 3:
			var r, g, b uint8
			if _, err := fmt.Sscanf(hex, "%1x%1x%1x", &r, &g, &b); err != nil {
				return color.RGBA{}, false
			}
			return color.RGBA{R: r * 17, G: g * 17, B: b * 17, A: 255}, true
		case 6:
			var r, g, b uint8
			if _, err := fmt.Sscanf(hex, "%2x%2x%2x", &r, &g, &b); err != nil {
				return color.RGBA{}, false
			}
			return color.RGBA{R: r, G: g, B: b, A: 255}, true
		}
		return color.RGBA{}, false
	}
	if strings.HasPrefix(s, "rgba(") && strings.HasSuffix(s, ")") {
		var r, g, b, a int
		if _, err := fmt.Sscanf(strings.TrimSuffix(strings.TrimPrefix(s, "rgba("), ")"), "%d,%d,%d,%d", &r, &g, &b, &a); err != nil {
			return color.RGBA{}, false
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}, true
	}
	return color.RGBA{}, false
}
