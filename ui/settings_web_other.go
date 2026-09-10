//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/client"
)

// settingsWebEnabled reports whether the page provides the DOM settings
// window. Native builds always render the canvas window.
func settingsWebEnabled() bool { return false }

// webSync is a no-op without a page; the canvas window owns the UI.
func (w *SettingsWindow) webSync(ctx client.Context) {}

// handleSettingsWebAction never runs without a page feeding set: actions.
func (w *SettingsWindow) handleSettingsWebAction(ctx client.Context, action string) {}
