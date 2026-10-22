package db

import (
	"image/color"
	"time"
)

func shieldChainEffectSpec() EffectSpec {
	spec := shieldBoomerangEffectSpec()
	spec.Duration = 55 * EffectFrameDuration
	spec.Components[0].FuncName = "ShieldChain"
	return spec
}

func pressureEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 76 * EffectFrameDuration,
		SFX:      []string{"effect\\프레셔.wav"}, SFXDelays: []time.Duration{3 * EffectFrameDuration},
		CameraShake: 200 * time.Millisecond, CameraShakeDelay: 32 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Pressure", TextureFile: "effect/cross_old.bmp",
			TextureFiles: []string{"effect/explosive_1_128.bmp"},
		}},
	}
}

func sacrificeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 51 * EffectFrameDuration,
		SFX:      []string{"effect\\세크리파이스.wav"}, SFXDelays: []time.Duration{22 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga",
		}},
	}
}

func gospelGroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "FlatColorTile",
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 13},
				SizeStart:      1,
				RenderBefore:   true,
				AttachedEntity: false,
			},
			{
				Kind: EffectComponentFUNC, FuncName: "GospelCross", TextureFile: "effect/cross_old.bmp", Repeat: true,
			},
		},
	}
}

func bottomGospelEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\가스펠.wav")
}
