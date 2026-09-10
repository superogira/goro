//go:build !js || !wasm

package ui

import (
	"github.com/kivutar/goro/client"
)

// serviceWebEnabled reports whether the page provides the DOM service
// window. Native builds always render the canvas window.
func serviceWebEnabled() bool { return false }

// serviceWebSync is a no-op without a page.
func (w *ServiceWindow) serviceWebSync(ctx client.Context) {}

// handleServiceWebAction never runs without a page feeding svc: actions.
func (w *ServiceWindow) handleServiceWebAction(ctx client.Context, action string) {}
