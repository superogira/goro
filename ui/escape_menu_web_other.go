//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/client"
)

// escapeMenuWebEnabled reports whether the page provides the DOM option
// menu. Native builds always render the canvas window.
func escapeMenuWebEnabled() bool { return false }

// webSync is a no-op without a page; the canvas window owns the UI.
func (m *EscapeMenu) webSync(ctx client.Context) {}

// handleOptionWebAction never runs without a page feeding esc: actions.
func (m *EscapeMenu) handleOptionWebAction(ctx client.Context, action string) {}
