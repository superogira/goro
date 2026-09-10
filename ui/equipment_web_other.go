//go:build !js || !wasm

package ui

// equipmentWebEnabled reports whether the page provides the DOM equipment
// window. Native builds always render the canvas window.
func equipmentWebEnabled() bool { return false }

// webSync is a no-op without a page; the canvas window owns the UI.
func (w *EquipmentWindow) webSync(ctx Context) {}

// handleEquipWebAction never runs without a page feeding eq: actions.
func (w *EquipmentWindow) handleEquipWebAction(ctx Context, action string) {}
