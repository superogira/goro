package ui

import (
	"fmt"

	"github.com/kivutar/goro/client"
)

// Handheld rows for the settings window: d-pad up/down walks the entries,
// left/right adjusts values, A toggles booleans (or steps a value up). The
// entries reuse the same setters and persistence the mouse widgets use.

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

func (w *SettingsWindow) settingsRows() []settingsRow {
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
		{kind: settingsRowChoice, label: "Resolution", value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(settingsResolutionScale(ctx)*100))
		}, adjust: func(ctx client.Context, delta int) {
			index := settingsResolutionIndex(ctx)
			index += delta
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
		}},
		{kind: settingsRowChoice, label: "BGM Volume", value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(settingsVolumeBGM(ctx)*100))
		}, adjust: func(ctx client.Context, delta int) {
			volume := clampVolumeStep(settingsVolumeBGM(ctx) + float64(delta)*0.1)
			if ctx.Audio != nil {
				ctx.Audio.SetBGMVolume(volume)
			}
			w.saveSettings(ctx)
		}},
		{kind: settingsRowChoice, label: "SFX Volume", value: func(ctx client.Context) string {
			return fmt.Sprintf("%d%%", int(settingsVolumeSFX(ctx)*100))
		}, adjust: func(ctx client.Context, delta int) {
			volume := clampVolumeStep(settingsVolumeSFX(ctx) + float64(delta)*0.1)
			if ctx.Audio != nil {
				ctx.Audio.SetSFXVolume(volume)
			}
			w.saveSettings(ctx)
		}},
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
	rows := w.settingsRows()
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

// GamepadAdjust applies a left/right step on the selected row.
func (w *SettingsWindow) GamepadAdjust(ctx client.Context, dx int) {
	if w == nil || !w.IsOpen() || dx == 0 {
		return
	}
	rows := w.settingsRows()
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
