//go:build js && wasm

package ui

import (
	"strconv"
	"strings"
	"sync"
	"syscall/js"
)

// The HUD and basic menu render as page DOM on the web build — the same
// reason the dormant chat log does: browser text is crisp at every device
// scale and zoom level, while canvas-rastered text blurs on Windows
// display scaling. DOM composites on the browser's own threads, so the
// wasm canvas pays nothing for their updates.

// hudWebActionQueue marshals DOM interactions (button taps, close, tab)
// onto the game goroutine.
var hudWebActionQueue = struct {
	sync.Mutex
	actions []string
}{}

var hudWebHooksInstalled bool

// hudWebInstallHooks exposes the page hooks: goroHudSync pushes state
// (called from the game goroutine) and goroMenuAction/goroHudAction feed
// taps back (called from DOM events).
func hudWebInstallHooks() {
	if hudWebHooksInstalled {
		return
	}
	hudWebHooksInstalled = true
	js.Global().Set("goroHudAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "hud:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
	js.Global().Set("goroMenuAction", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 1 && args[0].Type() == js.TypeString {
			hudWebActionQueue.Lock()
			hudWebActionQueue.actions = append(hudWebActionQueue.actions, "menu:"+args[0].String())
			hudWebActionQueue.Unlock()
		}
		return nil
	}))
}

// hudWebDrainActions takes queued DOM interactions whose action starts
// with the given prefix ("hud:" or "menu:") and leaves the rest queued.
// Consumers run at different points of the frame loop, so a consumer must
// never take — let alone discard — another consumer's actions.
func hudWebDrainActions(prefix string) []string {
	hudWebActionQueue.Lock()
	defer hudWebActionQueue.Unlock()
	if len(hudWebActionQueue.actions) == 0 {
		return nil
	}
	var out []string
	kept := hudWebActionQueue.actions[:0]
	for _, action := range hudWebActionQueue.actions {
		if strings.HasPrefix(action, prefix) {
			out = append(out, action)
		} else {
			kept = append(kept, action)
		}
	}
	hudWebActionQueue.actions = kept
	return out
}

// hudWebSync pushes the full HUD state and both windows' visibility to the
// page. Values arrive pre-formatted so the page only ever interpolates.
func hudWebSync(fields [15]string) {
	sync := js.Global().Get("goroHudSync")
	if sync.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	names := []string{
		"open", "name", "job",
		"hp", "maxHp", "sp", "maxSp",
		"baseLv", "baseExp", "nextBaseExp",
		"jobLv", "jobExp", "nextJobExp",
		"weight", "zeny",
	}
	for i, name := range names {
		if i == 0 {
			obj.Set(name, fields[0] == "1")
			continue
		}
		obj.Set(name, fields[i])
	}
	sync.Invoke(obj)
}

// menuWebSync reports the menu window's visibility.
func menuWebSync(open bool) {
	sync := js.Global().Get("goroHudSync")
	if sync.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("menuOpen", open)
	sync.Invoke(obj)
}

// hudWebEnabled reports whether the page provides the DOM HUD.
func hudWebEnabled() bool {
	return js.Global().Get("goroHudSync").Type() == js.TypeFunction
}

// hudFormatNumber groups thousands with commas, matching the canvas HUD.
func hudFormatNumber(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := strconv.FormatInt(value, 10)
	if len(text) <= 3 {
		return sign + text
	}
	var b strings.Builder
	prefix := len(text) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(text[:prefix])
	for i := prefix; i < len(text); i += 3 {
		b.WriteByte(',')
		b.WriteString(text[i : i+3])
	}
	return sign + b.String()
}
