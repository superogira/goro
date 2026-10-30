package app

import (
	"testing"

	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
)

func TestNewRestoresFogAndAuraSettings(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		g, err := New(config.Config{
			Headless: true,
			DataDir:  t.TempDir(),
			Fog:      config.FogConfig{Enabled: enabled},
			Gameplay: config.GameplayConfig{SimpleAura: enabled},
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(g.Close)
		if g.session.FogEnabled != enabled || g.session.SimpleAura != enabled {
			t.Fatalf("fog=%t simple aura=%t, want %t", g.session.FogEnabled, g.session.SimpleAura, enabled)
		}
	}
}

func TestNewForceUserAIEnablesCompanionCustomAI(t *testing.T) {
	g, err := New(config.Config{
		DataDir: t.TempDir(),
		Window: config.WindowConfig{
			Width:  1280,
			Height: 720,
		},
		Packet: config.PacketConfig{
			ClientDate: 20080910,
		},
		Audio: config.AudioConfig{
			BGMVolume: 0.55,
			SFXVolume: 0.55,
		},
		Render: config.RenderConfig{
			GraphicsAPI: "vulkan",
			VSync:       true,
		},
		Gameplay: config.GameplayConfig{
			ForceUserAI: true,
		},
		Log: glog.LogConfig{
			Level: "info",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !g.session.HomunculusCustomAI || !g.session.MercenaryCustomAI {
		t.Fatalf("custom AI flags = homunculus:%v mercenary:%v, want both true", g.session.HomunculusCustomAI, g.session.MercenaryCustomAI)
	}
}
