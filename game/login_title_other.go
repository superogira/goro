//go:build !js || !wasm

package game

import (
	"github.com/kivutar/goro/client"
)

// titleWebEnabled reports whether the page provides the DOM title layer.
// Native builds always render the canvas title.
func titleWebEnabled() bool { return false }

// skipCanvasTitleBackground is always false on native builds.
func (m *LoginMode) skipCanvasTitleBackground() bool { return false }

// syncTitleWeb is a no-op without a page.
func (m *LoginMode) syncTitleWeb(ctx client.Context, alpha float64) {}
