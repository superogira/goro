//go:build !js || !wasm

package game

import (
	"github.com/kivutar/goro/client"
)

// Native builds always render the canvas character select.
func charSelectWebEnabled() bool { return false }

func (m *LoginMode) skipCanvasCharSelectBackground() bool { return false }

func (m *LoginMode) drainCharSelectWebActions(ctx client.Context) {}

func (m *LoginMode) syncCharSelectWeb(ctx client.Context) {}
