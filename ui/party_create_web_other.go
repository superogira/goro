//go:build !js || !wasm

package ui

// Native stubs: no DOM panel exists, so the caller falls back to the
// canvas window.

func partyCreateWebSync(open bool) bool { return false }

func drainPartyCreateWebActions() (action PartyCreateWindowAction, ok, cancelled bool) {
	return PartyCreateWindowAction{}, false, false
}
