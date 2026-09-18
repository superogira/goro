package ui

import (
	"fmt"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	settingsWindowW = 300
	settingsWindowH = 430
)

type SettingsWindow struct {
	Window
	webOpen    bool
	webSyncKey string
	// gamepadSelected is the handheld row selection over the settings rows.
	gamepadSelected int
}

// Toggle opens or closes the settings window (the handheld menu entry and
// the B close stack use it).
func (w *SettingsWindow) Toggle(ctx client.Context) {
	if settingsWebEnabled() {
		w.webOpen = !w.webOpen
		w.webSyncKey = ""
		w.webSync(ctx)
		return
	}
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	if w.IsOpen() {
		w.Close()
		w.Publish(ctx)
		return
	}
	w.OpenWindow(ctx)
}

// IsOpen reports the settings window's open state. On web the DOM panel
// owns visibility; the input blocker chain consults this answer.
func (w *SettingsWindow) IsOpen() bool {
	if settingsWebEnabled() {
		return w.webOpen
	}
	return w.Window.IsOpen()
}

func (w *SettingsWindow) OpenWindow(ctx client.Context) {
	if settingsWebEnabled() {
		w.webOpen = true
		w.webSyncKey = ""
		w.webSync(ctx)
		return
	}
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	w.gamepadSelected = 0
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) Update(ctx client.Context) bool {
	if settingsWebEnabled() {
		w.UpdateWeb(ctx)
		return false
	}
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

// UpdateWeb services DOM settings actions and state sync. It must run from
// the head of the world update, before map-fade and disconnect-dialog
// early-returns: while those states persist, the window chain never reaches
// Update, a queued "set:scale:" from the DOM sits unapplied, and the native
// select keeps the picked value — the canvas then renders at the old scale
// while the settings claim the new one.
func (w *SettingsWindow) UpdateWeb(ctx client.Context) {
	if !settingsWebEnabled() {
		return
	}
	w.ctx = ctx
	for _, action := range hudWebDrainActions("set:") {
		w.handleSettingsWebAction(ctx, action)
	}
	if key := w.settingsWebKey(ctx); key != w.webSyncKey {
		w.webSyncKey = key
		w.webSync(ctx)
	}
}

func (w *SettingsWindow) Rebind(ctx client.Context) {
	if settingsWebEnabled() {
		w.ctx = ctx
		w.webSyncKey = ""
		return
	}
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx)
}

func (w *SettingsWindow) widgetTree(ctx client.Context) widget.Widget {
	return Win(
		Title("Settings"),
		CloseButton(true),
		OnClose(w.Close),
		Size(settingsWindowW, settingsWindowH),
		Content(w.contentTree(ctx)),
	)
}

// contentTree renders the handheld settings rows: d-pad up/down moves the
// selection, left/right adjusts the value, A toggles booleans. The same
// setters and persistence the upstream mouse widgets used back every row.
func (w *SettingsWindow) contentTree(ctx client.Context) widget.Widget {
	rows := w.settingsRows()
	children := make([]widget.Widget, 0, len(rows)+3)
	section := func(label string) {
		children = append(children, rotheme.Label(label))
	}

	rowWidget := func(index int, row settingsRow) widget.Widget {
		selected := index == w.gamepadSelected
		background := rotheme.Default.Colors.PanelBody
		if selected {
			background = rotheme.Default.Colors.ButtonHover
		}
		return primitives.HBox(
			primitives.Box(
				rotheme.Text(row.label).
					Color(itemInfoWidgetColor(inventoryTextColor)),
			).Width(130),
			primitives.Expanded(
				rotheme.Text(row.value(ctx)).
					Color(itemInfoWidgetColor(inventoryTextColor)),
			),
		).
			Height(22).
			Background(background)
	}

	section("Display")
	for i := 0; i < 4; i++ {
		children = append(children, rowWidget(i, rows[i]))
	}
	section("Sound")
	for i := 4; i < 6; i++ {
		children = append(children, rowWidget(i, rows[i]))
	}
	section("Gameplay")
	for i := 6; i < len(rows); i++ {
		children = append(children, rowWidget(i, rows[i]))
	}
	return primitives.Box(children...).
		Padding(14).
		Gap(4)
}

func (w *SettingsWindow) refresh(ctx client.Context) {
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) saveSettings(ctx client.Context) {
	settings := config.UserSettings{
		Fullscreen:      settingsRuntimeFullscreen(ctx),
		VSync:           settingsRuntimeVSync(ctx),
		FPS:             settingsRuntimeFPS(ctx),
		ResolutionScale: settingsResolutionScale(ctx),
		BGMVolume:       settingsVolumeBGM(ctx),
		SFXVolume:       settingsVolumeSFX(ctx),
		NoShift:         settingsNoShift(ctx),
		NoCtrl:          settingsNoCtrl(ctx),
		LessEffects:     settingsLessEffects(ctx),
		SnapTargets:     settingsSnapTargets(ctx),
		SnapItems:       settingsSnapItems(ctx),
		SnapRadius:      settingsSnapRadius(ctx),
	}
	path, err := ctx.Config.SaveUserSettings(settings)
	if err != nil {
		glog.Warnf("settings save failed: %v", err)
		return
	}
	glog.Debugf("settings saved path=%s", path)
}

func settingsVolumeBGM(ctx client.Context) float64 {
	if ctx.Audio != nil {
		return ctx.Audio.BGMVolume()
	}
	return ctx.Config.Audio.BGMVolume
}

func settingsVolumeSFX(ctx client.Context) float64 {
	if ctx.Audio != nil {
		return ctx.Audio.SFXVolume()
	}
	return ctx.Config.Audio.SFXVolume
}

func settingsRuntimeFullscreen(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.Fullscreen()
	}
	return ctx.Config.Window.Fullscreen
}

func settingsRuntimeVSync(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.VSync()
	}
	return ctx.Config.Render.VSync
}

func settingsRuntimeFPS(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.FPS()
	}
	return ctx.Config.Render.FPS
}

// settingsResolutionScale returns the live canvas backing-store scale,
// falling back to the config value before the runtime exists.
func settingsResolutionScale(ctx client.Context) float64 {
	if ctx.Runtime != nil {
		return ctx.Runtime.ResolutionScale()
	}
	if s := ctx.Config.Render.ResolutionScale; s > 0 && s <= 1 {
		return s
	}
	return 1
}

// resolutionScaleOptions are the selectable render scales, in percent.
var resolutionScaleOptions = []int{100, 90, 80, 70, 60, 50}

// resolutionScaleButtons builds the scale picker: the active percent uses
// the highlighted button style so the current level reads at a glance.
func resolutionScaleButtons(ctx client.Context, onPick func(float64)) widget.Widget {
	current := settingsResolutionScale(ctx)
	buttons := make([]widget.Widget, 0, len(resolutionScaleOptions))
	for _, percent := range resolutionScaleOptions {
		scale := float64(percent) / 100
		label := fmt.Sprintf("%d%%", percent)
		if absFloat(scale-current) < 0.001 {
			label = "[" + label + "]"
		}
		buttons = append(buttons, rotheme.Button(label, func() {
			onPick(scale)
		}))
	}
	return primitives.HBox(buttons...).Gap(4)
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func settingsNoShift(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.NoShift
	}
	return ctx.Config.Gameplay.NoShift
}

func settingsNoCtrl(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.NoCtrl
	}
	return ctx.Config.Gameplay.NoCtrl
}

func settingsLessEffects(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.LessEffects
	}
	return ctx.Config.Gameplay.LessEffects
}

func settingsSnapTargets(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapTargets
	}
	return ctx.Config.Gameplay.SnapTargets
}

func settingsSnapItems(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapItems
	}
	return ctx.Config.Gameplay.SnapItems
}

// settingsSnapRadius returns the live pick/snap area multiplier, defaulting
// to 1 before the session exists.
func settingsSnapRadius(ctx client.Context) float64 {
	if ctx.Session != nil && ctx.Session.SnapRadius > 0 {
		return ctx.Session.SnapRadius
	}
	if v := ctx.Config.Gameplay.SnapRadius; v > 0 {
		return v
	}
	return 1
}
