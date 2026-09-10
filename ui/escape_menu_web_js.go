//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/kivutar/goro/client"
)

// escapeMenuWebEnabled reports whether the page provides the DOM option
// menu.
func escapeMenuWebEnabled() bool {
	return js.Global().Get("goroOptionSync").Type() == js.TypeFunction
}

// webSync pushes the option menu state to the page.
func (m *EscapeMenu) webSync(ctx client.Context) {
	fn := js.Global().Get("goroOptionSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", m.webOpen)
	obj.Set("death", m.deathMode)
	obj.Set("revive", client.AutoReviveAvailable(ctx))
	obj.Set("pending", m.pending)
	fn.Invoke(obj)
}

// handleOptionWebAction services one drained "esc:" action. Button presses
// mirror the canvas handlers: navigation actions set m.action for the world
// mode to consume; quit sends straight away.
func (m *EscapeMenu) handleOptionWebAction(ctx client.Context, action string) {
	resync := true
	switch action {
	case "esc:close", "esc:cancel":
		m.webOpen = false
	case "esc:settings":
		m.action = EscapeMenuActionSettings
		m.webOpen = false
	case "esc:char":
		m.action = EscapeMenuActionCharacterSelect
	case "esc:savepoint":
		m.action = EscapeMenuActionSavePoint
	case "esc:revive":
		m.action = EscapeMenuActionAutoRevive
	case "esc:exit":
		m.RequestQuitGame(ctx)
	default:
		resync = false
	}
	if resync {
		m.webSyncKey = ""
	}
}
