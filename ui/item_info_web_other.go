//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/client"
)

// Native build: item descriptions are canvas ItemWindows; the DOM manager
// is a no-op so the world update chain stays platform-independent.
func UpdateItemInfoWebWindows(ctx client.Context) bool { return false }
