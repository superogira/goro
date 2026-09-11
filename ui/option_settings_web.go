package ui

import (
	"fmt"

	"github.com/kivutar/goro/client"
)

// optionWebKey summarizes the DOM option menu's contents.
func (m *EscapeMenu) optionWebKey(ctx client.Context) string {
	return fmt.Sprintf("%t|%t|%t|%t", m.webOpen, m.deathMode, client.AutoReviveAvailable(ctx), m.pending)
}

// settingsWebKey summarizes the DOM settings window's contents. The
// resolution scale must be part of the key: without it a runtime scale
// change that skips the action handler (boot override, future callers)
// never re-syncs the picker, leaving it showing a stale percent.
func (w *SettingsWindow) settingsWebKey(ctx client.Context) string {
	return fmt.Sprintf("%t|%t|%t|%t|%.2f|%.2f|%.2f|%t|%t|%t|%t|%t",
		w.webOpen,
		settingsRuntimeFullscreen(ctx), settingsRuntimeVSync(ctx), settingsRuntimeFPS(ctx),
		settingsResolutionScale(ctx),
		settingsVolumeBGM(ctx), settingsVolumeSFX(ctx),
		settingsNoShift(ctx), settingsNoCtrl(ctx), settingsLessEffects(ctx),
		settingsSnapTargets(ctx), settingsSnapItems(ctx))
}
