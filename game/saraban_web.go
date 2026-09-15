//go:build js && wasm

package game

import (
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/world"
)

// The AEOPD document registry NPC (npc/other/aeopd.txt) is a hidden NPC in
// front of the Prontera Library bookshelf. Clicking it does not open a
// server dialog: the web client opens the DOM registry window (login +
// document list) instead, served by the webbridge /aeopd endpoints.

// sarabanNPCName is the script name of the portal NPC; the part after the
// '#' is a disambiguation tag, so both spellings match.
const sarabanNPCName = "AEOPD"

// sarabanPortalActor reports whether the clicked NPC is the registry
// portal.
func sarabanPortalActor(actor world.Actor) bool {
	base := actor.Name
	if i := strings.IndexByte(base, '#'); i >= 0 {
		base = base[:i]
	}
	return strings.TrimSpace(base) == sarabanNPCName
}

// openSarabanWeb tells the page to show the registry window. It reports
// whether the page provides the hook (native stubs return false and the
// click falls through to a normal NPC contact).
func openSarabanWeb() bool {
	fn := js.Global().Get("goroSarabanOpen")
	if fn.Type() != js.TypeFunction {
		return false
	}
	fn.Invoke()
	return true
}
