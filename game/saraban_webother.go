//go:build !js || !wasm

package game

import (
	"github.com/kivutar/goro/world"
)

// Native builds have no DOM registry window: clicks on the portal NPC run
// the (empty) server script like any other NPC.
func sarabanPortalActor(actor world.Actor) bool { return false }

func openSarabanWeb() bool { return false }
