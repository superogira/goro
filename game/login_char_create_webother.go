//go:build !js || !wasm

package game

import (
	"github.com/kivutar/goro/client"
)

// Native builds always render the canvas character creation.
func charCreateWebEnabled() bool { return false }

func (m *LoginMode) drainCharCreateWebActions(ctx client.Context) {}

func (m *LoginMode) syncCharCreateWeb(ctx client.Context) {}
