//go:build !js || !wasm

package ui

// Native builds keep the HUD and basic menu on the game canvas, where the
// raster is dpi-matched by the window system and text renders crisply.

func hudWebInstallHooks() {}

func hudWebDrainActions(prefix string) []string { return nil }

func hudWebSync(fields [15]string) {}

func menuWebSync(open bool) {}

func hudWebEnabled() bool { return false }

func hudFormatNumber(value int64) string { return formatHUDNumber(value) }
