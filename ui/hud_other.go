//go:build !js || !wasm

package ui

import (
	"image"

	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

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

func inventoryWebEnabled() bool { return false }

func (w *InventoryBagWindow) itemInfoWebSync(ctx Context, item session.InventoryItem) {}

func inventoryWebSync(open bool, tab int, items []session.InventoryItem, icons []string, names []string, weights []int) {}

func hotbarWebIcon(key string, img image.Image) string { return "" }

func DrainCameraActions() []string { return nil }

func pickupWebEnabled() bool { return false }

func pickupWebShow(text string, icon string) {}

func pickupWebIcon(manager *res.Manager, item session.InventoryItem) string { return "" }

func (b *ShortcutBar) hotbarWebSync(ctx Context) {}

func (b *ShortcutBar) hotbarWebKey(ctx Context) string { return "" }

func hudFormatNumber(value int64) string { return formatHUDNumber(value) }
