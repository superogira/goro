package db

import "time"

func firstAidEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: time.Second,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{{
			Kind:          EffectComponent2D,
			TextureFile:   "effect/pikapika2.bmp",
			Duration:      time.Second,
			AlphaMax:      0.2,
			BlendAdditive: true,
			FadeOut:       true,
			PosZ:          2,
			SizeStart:     100 * EffectPixelRatio,
			SizeEnd:       100 * EffectPixelRatio,
		}},
	}
}
