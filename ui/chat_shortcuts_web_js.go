//go:build js && wasm

package ui

import (
	"fmt"
	"strconv"
	"strings"
	"syscall/js"
)

// DOM twin of the chat shortcuts window (Alt+M): the page renders the ten
// command rows and the View/Close footer; the game keeps ownership of the
// command table, sanitization and persistence (goro.ini on native,
// localStorage on web via saveUserConfigValues).

// chatShortcutsWebEnabled reports whether the page provides the DOM chat
// shortcuts panel.
func chatShortcutsWebEnabled() bool {
	return js.Global().Get("goroChatShortcutsSync").Type() == js.TypeFunction
}

// drainChatShortcutsWebActions services the panel's "cshort:" actions:
// per-slot command drafts, the emote View button and close.
func (w *ChatShortcutsWindow) drainChatShortcutsWebActions(ctx Context) {
	for _, action := range hudWebDrainActions("cshort:") {
		switch {
		case action == "cshort:close":
			w.webOpen = false
		case action == "cshort:view":
			if w.onView != nil {
				w.onView()
			}
		case strings.HasPrefix(action, "cshort:set:"):
			parts := strings.SplitN(strings.TrimPrefix(action, "cshort:set:"), ":", 2)
			if len(parts) != 2 {
				continue
			}
			slot, err := strconv.Atoi(parts[0])
			if err != nil || slot < 0 || slot >= len(w.commands) {
				continue
			}
			w.selected = slot
			w.setCommand(slot, parts[1])
		}
	}
}

// syncChatShortcutsWeb pushes the panel state when the open flag or the
// command table changed.
func (w *ChatShortcutsWindow) syncChatShortcutsWeb(ctx Context) {
	if !chatShortcutsWebEnabled() {
		return
	}
	hudWebInstallHooks()
	key := fmt.Sprintf("%t|%v", w.webOpen, w.commands)
	if key == w.webSyncKey {
		return
	}
	w.webSyncKey = key
	obj := js.Global().Get("Object").New()
	obj.Set("open", w.webOpen)
	obj.Set("selected", w.selected)
	commands := js.Global().Get("Array").New(len(w.commands))
	for slot, command := range w.commands {
		commands.SetIndex(slot, command)
	}
	obj.Set("commands", commands)
	js.Global().Get("goroChatShortcutsSync").Invoke(obj)
}
