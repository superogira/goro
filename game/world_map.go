package game

import (
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
)

// Use the physical key left of 1, including on AZERTY. Do not steal AltGr.
func worldMapShortcutDown(in *input.State) bool {
	return in != nil && in.Pressed(input.KeyCtrl) && !in.Pressed(input.KeyAlt) &&
		!in.Pressed(input.KeyShift) && !in.KeyCodeDown(gpucontext.KeyRightAlt) &&
		!in.KeyCodeDown(gpucontext.KeyLeftSuper) && !in.KeyCodeDown(gpucontext.KeyRightSuper)
}

func (m *WorldMode) toggleWorldMapFromInput(ctx client.Context) bool {
	if !worldMapShortcutDown(ctx.Input) || !ctx.Input.KeyCodeJustPressed(gpucontext.KeyGrave) {
		return false
	}
	if !m.ui.worldMap.IsOpen() && m.ui.nonConsoleKeyboardInputBlocked(ctx) {
		return false
	}
	ctx.Input.ConsumeKeyCodePress(gpucontext.KeyGrave)
	if err := m.ui.worldMap.Toggle(ctx); err != nil {
		m.ui.console.AddErrorMessage("World map unavailable: %v", err)
		glog.Warnf("world map: %v", err)
	}
	return true
}
