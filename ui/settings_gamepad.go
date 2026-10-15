package ui

import (
	"fmt"
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

// Handheld rows for the settings window: d-pad up/down walks the entries,
// left/right adjusts values, A toggles booleans (or steps a value up). The
// entries reuse the same setters and persistence the mouse widgets use.
// Rows are grouped into tabs (L1/R1 cycles) — the flat list outgrew the
// 480px screen once the v0.20 quality options joined.

type settingsRowKind int

const (
	settingsRowToggle settingsRowKind = iota
	settingsRowChoice
)

type settingsRow struct {
	kind  settingsRowKind
	label string
	value func(ctx client.Context) string
	// adjust applies one step in the direction of delta (-1 left/down,
	// +1 right/up); bool rows ignore the direction and toggle.
	adjust func(ctx client.Context, delta int)
}

func (w *SettingsWindow) displayRows() []settingsRow {
	return []settingsRow{
		{kind: settingsRowToggle, label: "Fullscreen", value: func(ctx client.Context) string {
			return settingsOnOff(settingsRuntimeFullscreen(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetFullscreen(!settingsRuntimeFullscreen(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "VSync (Restart)", value: func(ctx client.Context) string {
			return settingsOnOff(settingsRuntimeVSync(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetVSync(!settingsRuntimeVSync(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "FPS meter", value: func(ctx client.Context) string {
			return settingsOnOff(settingsRuntimeFPS(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetFPS(!settingsRuntimeFPS(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowChoice, label: settingsResolutionLabel(), value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(math.Round(settingsResolutionScale(ctx)*100)))
		}, adjust: func(ctx client.Context, delta int) {
			// The options run 100%→50%; stepping toward 100% is the
			// "increase" direction, so right (delta > 0) lowers the index.
			index := settingsResolutionIndex(ctx) - delta
			if index < 0 {
				index = 0
			}
			if index >= len(resolutionScaleOptions) {
				index = len(resolutionScaleOptions) - 1
			}
			if ctx.Runtime != nil {
				ctx.Runtime.SetResolutionScale(float64(resolutionScaleOptions[index]) / 100)
			}
			w.saveSettings(ctx)
			if !settingsWebEnabled() {
				// Native keeps the startup resolution until relaunch; the
				// pick only lands after a restart.
				render.ShowScreenNotice("Resolution applies after restart")
			}
		}},
		{kind: settingsRowChoice, label: "Anisotropic filtering", value: func(ctx client.Context) string {
			return anisotropyLabel(settingsAnisotropy(ctx))
		}, adjust: func(ctx client.Context, delta int) {
			// 0→2→4→8→16→0; left (delta<0) runs the cycle backwards.
			level := settingsAnisotropy(ctx)
			if delta >= 0 {
				level *= 2
				if level == 0 {
					level = 2
				} else if level > 16 {
					level = 0
				}
			} else {
				level /= 2
				if level < 2 {
					level = 16
				}
			}
			if ctx.Runtime != nil {
				ctx.Runtime.SetAnisotropy(level)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Upscaling (Next map)", value: func(ctx client.Context) string {
			return settingsOnOff(settingsTextureUpscaling(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetTextureUpscaling(!settingsTextureUpscaling(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Smooth world sprites", value: func(ctx client.Context) string {
			return settingsOnOff(settingsSmoothSprites(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetSmoothSprites(!settingsSmoothSprites(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Anti-aliasing 4x (Restart)", value: func(ctx client.Context) string {
			return settingsOnOff(settingsMSAA(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetMSAA(!settingsMSAA(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Bloom", value: func(ctx client.Context) string {
			return settingsOnOff(settingsBloom(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetBloom(!settingsBloom(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Ambient occlusion", value: func(ctx client.Context) string {
			return settingsOnOff(settingsSSAO(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Runtime != nil {
				ctx.Runtime.SetSSAO(!settingsSSAO(ctx))
			}
			w.saveSettings(ctx)
		}},
	}
}

func (w *SettingsWindow) soundRows() []settingsRow {
	return []settingsRow{
		{kind: settingsRowChoice, label: "BGM Volume", value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(math.Round(settingsVolumeBGM(ctx)*100)))
		}, adjust: func(ctx client.Context, delta int) {
			volume := clampVolumeStep(settingsVolumeBGM(ctx) + float64(delta)*0.1)
			if ctx.Audio != nil {
				ctx.Audio.SetBGMVolume(volume)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowChoice, label: "SFX Volume", value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(math.Round(settingsVolumeSFX(ctx)*100)))
		}, adjust: func(ctx client.Context, delta int) {
			volume := clampVolumeStep(settingsVolumeSFX(ctx) + float64(delta)*0.1)
			if ctx.Audio != nil {
				ctx.Audio.SetSFXVolume(volume)
			}
			w.saveSettings(ctx)
		}},
	}
}

func (w *SettingsWindow) gameplayRows() []settingsRow {
	return []settingsRow{
		{kind: settingsRowToggle, label: "No Shift", value: func(ctx client.Context) string {
			return settingsOnOff(settingsNoShift(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Session != nil {
				ctx.Session.NoShift = !settingsNoShift(ctx)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "No Ctrl", value: func(ctx client.Context) string {
			return settingsOnOff(settingsNoCtrl(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Session != nil {
				ctx.Session.NoCtrl = !settingsNoCtrl(ctx)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Less Effects", value: func(ctx client.Context) string {
			return settingsOnOff(settingsLessEffects(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Session != nil {
				ctx.Session.LessEffects = !settingsLessEffects(ctx)
			}
			if ctx.Network != nil {
				_ = ctx.Network.SendLessEffect(settingsLessEffects(ctx))
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Snap to targets", value: func(ctx client.Context) string {
			return settingsOnOff(settingsSnapTargets(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Session != nil {
				ctx.Session.SnapTargets = !settingsSnapTargets(ctx)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowToggle, label: "Snap to items", value: func(ctx client.Context) string {
			return settingsOnOff(settingsSnapItems(ctx))
		}, adjust: func(ctx client.Context, _ int) {
			if ctx.Session != nil {
				ctx.Session.SnapItems = !settingsSnapItems(ctx)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowChoice, label: "Snap Radius", value: func(ctx client.Context) string {
			return fmt.Sprintf("%.2fx", settingsSnapRadius(ctx))
		}, adjust: func(ctx client.Context, delta int) {
			radius := settingsSnapRadius(ctx) + float64(delta)*0.25
			if radius < 0.5 {
				radius = 0.5
			}
			if radius > 3 {
				radius = 3
			}
			if ctx.Session != nil {
				ctx.Session.SnapRadius = radius
			}
			w.saveSettings(ctx)
		}},
	}
}

// settingsRowsForTab returns the active tab's rows.
func (w *SettingsWindow) settingsRowsForTab() []settingsRow {
	switch w.tab {
	case settingsTabSound:
		return w.soundRows()
	case settingsTabGameplay:
		return w.gameplayRows()
	default:
		return w.displayRows()
	}
}

// settingsRows keeps the flat list for callers that span tabs (tests,
// the web mirror).
func (w *SettingsWindow) settingsRows() []settingsRow {
	rows := w.displayRows()
	rows = append(rows, w.soundRows()...)
	rows = append(rows, w.gameplayRows()...)
	return rows
}

// settingsResolutionLabel flags the restart requirement on native builds,
// where the live retarget is web-only.
func settingsResolutionLabel() string {
	if settingsWebEnabled() {
		return "Resolution"
	}
	return "Resolution (Restart)"
}

func settingsOnOff(enabled bool) string {
	if enabled {
		return "On"
	}
	return "Off"
}

func clampVolumeStep(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func settingsResolutionIndex(ctx client.Context) int {
	current := settingsResolutionScale(ctx)
	for i, percent := range resolutionScaleOptions {
		if absFloat(float64(percent)/100-current) < 0.001 {
			return i
		}
	}
	return 0
}

// GamepadNavigate moves the row selection (dy -1 up, +1 down).
func (w *SettingsWindow) GamepadNavigate(ctx client.Context, dy int) {
	if w == nil || !w.IsOpen() || dy == 0 {
		return
	}
	rows := w.settingsRowsForTab()
	next := w.gamepadSelected + dy
	if next < 0 {
		next = 0
	}
	if next >= len(rows) {
		next = len(rows) - 1
	}
	if next == w.gamepadSelected {
		return
	}
	w.gamepadSelected = next
	w.refresh(ctx)
}

// GamepadTab cycles the settings tabs (handheld L1/R1); the row selection
// resets to the top of the newly shown tab.
func (w *SettingsWindow) GamepadTab(ctx client.Context, dir int) {
	if w == nil || !w.IsOpen() || dir == 0 {
		return
	}
	tabs := []settingsTab{settingsTabDisplay, settingsTabSound, settingsTabGameplay}
	idx := 0
	for i, tab := range tabs {
		if tab == w.tab {
			idx = i
			break
		}
	}
	idx = ((idx+dir)%len(tabs) + len(tabs)) % len(tabs)
	if tabs[idx] == w.tab {
		return
	}
	w.tab = tabs[idx]
	w.gamepadSelected = 0
	w.refresh(ctx)
}

// GamepadAdjust applies a left/right step on the selected row.
func (w *SettingsWindow) GamepadAdjust(ctx client.Context, dx int) {
	if w == nil || !w.IsOpen() || dx == 0 {
		return
	}
	rows := w.settingsRowsForTab()
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(rows) {
		return
	}
	rows[w.gamepadSelected].adjust(ctx, dx)
	w.refresh(ctx)
}

// GamepadActivate presses the selected row: toggles booleans, steps values up.
func (w *SettingsWindow) GamepadActivate(ctx client.Context) {
	if w == nil || !w.IsOpen() {
		return
	}
	rows := w.settingsRows()
	if w.gamepadSelected < 0 || w.gamepadSelected >= len(rows) {
		return
	}
	rows[w.gamepadSelected].adjust(ctx, 1)
	w.refresh(ctx)
}
