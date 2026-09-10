//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/client"
)

// loginWebEnabled reports whether the page provides the DOM login form.
// Native builds always render the canvas window.
func loginWebEnabled() bool { return false }

// loginWebSync is a no-op without a page.
func (w *LoginWindow) loginWebSync(ctx client.Context) {}

// handleLoginWebAction never runs without a page feeding login: actions.
func (w *LoginWindow) handleLoginWebAction(ctx client.Context, action string) {}
