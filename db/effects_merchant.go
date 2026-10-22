package db

import "time"

func mammoniteEffectSpec() EffectSpec {
	spec := strEffectSpecAttachedMin("maemor", "memor_min", "", false)
	spec.Duration = 100 * EffectFrameDuration
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "MammoniteCoins",
		SpriteFile: "동전", Duration: spec.Duration,
	})
	return spec
}

func cartRevolutionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("cartrevolution", "", false)
	spec.Duration = 300 * EffectFrameDuration
	spec.SFX = []string{"effect\\ef_magnumbreak.wav", "effect\\ef_magnumbreak.wav"}
	spec.SFXDelays = []time.Duration{7 * EffectFrameDuration, 20 * EffectFrameDuration}
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "CartRevolution", Duration: 40 * EffectFrameDuration,
		TextureFile: "effect/ring_yellow.tga", TextureFiles: []string{"effect/대폭발.tga"},
	})
	return spec
}

func loudEffectSpec() EffectSpec {
	return strEffectSpecAttached("loud", "effect\\고성방가.wav", false)
}
