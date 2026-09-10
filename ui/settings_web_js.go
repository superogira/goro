//go:build js && wasm

package ui

import (
	"strconv"
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/client"
)

// settingsWebEnabled reports whether the page provides the DOM settings
// window.
func settingsWebEnabled() bool {
	return js.Global().Get("goroSettingsSync").Type() == js.TypeFunction
}

// webSync pushes the settings snapshot to the page.
func (w *SettingsWindow) webSync(ctx client.Context) {
	fn := js.Global().Get("goroSettingsSync")
	if fn.Type() != js.TypeFunction {
		return
	}
	obj := js.Global().Get("Object").New()
	obj.Set("open", w.webOpen)
	obj.Set("fullscreen", settingsRuntimeFullscreen(ctx))
	obj.Set("vsync", settingsRuntimeVSync(ctx))
	obj.Set("fps", settingsRuntimeFPS(ctx))
	obj.Set("bgm", settingsVolumeBGM(ctx))
	obj.Set("sfx", settingsVolumeSFX(ctx))
	obj.Set("noShift", settingsNoShift(ctx))
	obj.Set("noCtrl", settingsNoCtrl(ctx))
	obj.Set("lessEffects", settingsLessEffects(ctx))
	obj.Set("snapTargets", settingsSnapTargets(ctx))
	obj.Set("snapItems", settingsSnapItems(ctx))
	fn.Invoke(obj)
}

// handleSettingsWebAction services one drained "set:" action. Each change
// applies immediately (the same runtime calls the canvas handlers make) and
// persists through SaveUserSettings — on web that lands in localStorage.
func (w *SettingsWindow) handleSettingsWebAction(ctx client.Context, action string) {
	resync := true
	switch {
	case action == "set:close":
		w.webOpen = false
	case strings.HasPrefix(action, "set:fullscreen:"):
		if ctx.Runtime != nil {
			ctx.Runtime.SetFullscreen(action[len("set:fullscreen:"):] == "1")
		}
	case strings.HasPrefix(action, "set:vsync:"):
		if ctx.Runtime != nil {
			// Live on web: retargets the frame pacing immediately.
			ctx.Runtime.SetVSync(action[len("set:vsync:"):] == "1")
		}
	case strings.HasPrefix(action, "set:fps:"):
		if ctx.Runtime != nil {
			ctx.Runtime.SetFPS(action[len("set:fps:"):] == "1")
		}
	case strings.HasPrefix(action, "set:bgm:"):
		if v, err := strconv.ParseFloat(strings.TrimPrefix(action, "set:bgm:"), 64); err == nil && ctx.Audio != nil {
			ctx.Audio.SetBGMVolume(v)
		}
	case strings.HasPrefix(action, "set:sfx:"):
		if v, err := strconv.ParseFloat(strings.TrimPrefix(action, "set:sfx:"), 64); err == nil && ctx.Audio != nil {
			ctx.Audio.SetSFXVolume(v)
		}
	case strings.HasPrefix(action, "set:noshift:"):
		if ctx.Session != nil {
			ctx.Session.NoShift = action[len("set:noshift:"):] == "1"
		}
	case strings.HasPrefix(action, "set:noctrl:"):
		if ctx.Session != nil {
			ctx.Session.NoCtrl = action[len("set:noctrl:"):] == "1"
		}
	case strings.HasPrefix(action, "set:lesseffects:"):
		enabled := action[len("set:lesseffects:"):] == "1"
		if ctx.Session != nil {
			ctx.Session.LessEffects = enabled
		}
		if ctx.Network != nil {
			_ = ctx.Network.SendLessEffect(enabled)
		}
	case strings.HasPrefix(action, "set:snap:"):
		if ctx.Session != nil {
			ctx.Session.SnapTargets = action[len("set:snap:"):] == "1"
		}
	case strings.HasPrefix(action, "set:itemsnap:"):
		if ctx.Session != nil {
			ctx.Session.SnapItems = action[len("set:itemsnap:"):] == "1"
		}
	default:
		resync = false
	}
	if resync {
		w.saveSettings(ctx)
		w.webSyncKey = ""
	}
}
