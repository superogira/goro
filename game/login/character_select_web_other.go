//go:build !js || !wasm

package login

import (
	"github.com/kivutar/goro/client"
)

// Native builds always render the canvas character select.
func charSelectWebEnabled() bool { return false }

func (m *Screen) skipCanvasCharSelectBackground() bool { return false }

func (m *Screen) drainCharSelectWebActions(ctx client.Context) {}

func (m *Screen) syncCharSelectWeb(ctx client.Context) {}
