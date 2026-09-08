//go:build !js || !wasm

package ui

import "time"

// Native builds keep the canvas minimap window.

func minimapWebEnabled() bool { return false }

func (m *Minimap) minimapWebMapData(size int) string { return "" }

func (m *Minimap) minimapWebSync(ctx Context, now time.Time) {}

func (m *Minimap) minimapWebSyncClosed() {}
