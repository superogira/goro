//go:build !js || !wasm

package login

import (
	"github.com/kivutar/goro/client"
)

// Native builds always render the canvas character creation.
func charCreateWebEnabled() bool { return false }

func (m *Screen) drainCharCreateWebActions(ctx client.Context) {}

func (m *Screen) syncCharCreateWeb(ctx client.Context) {}
