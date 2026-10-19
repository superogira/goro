package config

import (
	"strings"
	"testing"
)

// TestUserSettingsRoundTrip covers every key the DOM/canvas settings
// writers emit: the stored ini must parse back into the same values, or
// a reload silently reverts the user's picks.
func TestUserSettingsRoundTrip(t *testing.T) {
	want := UserSettings{
		Fullscreen: true, VSync: true, FPS: true,
		ResolutionScale:  0.7,
		Anisotropy:       8,
		TextureUpscaling: true,
		SmoothSprites:    false,
		MSAA:             true,
		Bloom:            true,
		SSAO:             true,
		BGMVolume:        0.33, SFXVolume: 0.44,
		NoShift: true, NoCtrl: true, LessEffects: true,
		SnapTargets: true, SnapItems: true, SnapRadius: 1.5,
	}
	stored := upsertINIValues("", userSettingsValues(want))
	cfg := defaultConfig()
	if err := applyINI(&cfg, strings.NewReader(stored)); err != nil {
		t.Fatalf("parse stored settings: %v", err)
	}
	got := UserSettings{
		Fullscreen: cfg.Window.Fullscreen, VSync: cfg.Render.VSync, FPS: cfg.Render.FPS,
		ResolutionScale:  cfg.Render.ResolutionScale,
		Anisotropy:       cfg.Render.Anisotropy,
		TextureUpscaling: cfg.Render.TextureUpscaling,
		SmoothSprites:    cfg.Render.SmoothSprites,
		MSAA:             cfg.Render.MSAA,
		Bloom:            cfg.Render.Bloom,
		SSAO:             cfg.Render.SSAO,
		BGMVolume:        cfg.Audio.BGMVolume, SFXVolume: cfg.Audio.SFXVolume,
		NoShift: cfg.Gameplay.NoShift, NoCtrl: cfg.Gameplay.NoCtrl, LessEffects: cfg.Gameplay.LessEffects,
		SnapTargets: cfg.Gameplay.SnapTargets, SnapItems: cfg.Gameplay.SnapItems, SnapRadius: cfg.Gameplay.SnapRadius,
	}
	if got != want {
		t.Fatalf("round trip mismatch: stored:\n%s\ngot: %+v want: %+v", stored, got, want)
	}
}
