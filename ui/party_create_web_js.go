//go:build js && wasm

package ui

import (
	"strings"
	"syscall/js"
)

// DOM twin of the party creation window (#goro-partycreate): the page
// renders the name field plus the pickup/sharing radios in the family
// styling; actions flow back through the "pcr:" queue. Field values
// arrive joined with \x1f so a party name may contain colons. Native
// builds keep the canvas window.

func partyCreateWebSync(open bool) bool {
	sync := js.Global().Get("goroPartyCreateSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	sync.Invoke(obj)
	return true
}

// drainPartyCreateWebActions services the form's "pcr:" actions.
func drainPartyCreateWebActions() (action PartyCreateWindowAction, ok, cancelled bool) {
	for _, a := range hudWebDrainActions("pcr:") {
		switch {
		case a == "pcr:cancel":
			cancelled = true
		case strings.HasPrefix(a, "pcr:ok"):
			fields := strings.Split(strings.TrimPrefix(a, "pcr:ok"), "\x1f")
			if len(fields) != 4 {
				continue
			}
			action = PartyCreateWindowAction{
				Name:         strings.TrimSpace(fields[1]),
				ItemPickup:   parsePartySettingUint8(fields[2]),
				ItemDivision: parsePartySettingUint8(fields[3]),
			}
			ok = true
		}
	}
	return action, ok, cancelled
}
