//go:build !js || !wasm

package ui

// Native builds keep the HUD and basic menu on the game canvas, where the
// raster is dpi-matched by the window system and text renders crisply.

func hudWebInstallHooks() {}

func hudWebDrainActions(prefix string) []string { return nil }

func hudWebSync(fields [15]string) {}

func menuWebSync(open bool) {}

func hudWebEnabled() bool { return false }

func statsWebSync(open bool, rows [6]statRow, derived [][2]string, points int, canIncrease [6]bool, guild string) {}

func statsWebEnabled() bool { return false }

func hotbarWebEnabled() bool { return false }

func hotbarWebEnabledStub() {}

func DrainCameraActions() []string { return nil }

func pickupWebEnabled() bool { return false }

func pickupWebShow(text string) {}

func (b *ShortcutBar) hotbarWebSync(ctx Context) {}

func (b *ShortcutBar) hotbarWebKey(ctx Context) string { return "" }

func hudFormatNumber(value int64) string { return formatHUDNumber(value) }
