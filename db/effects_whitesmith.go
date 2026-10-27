package db

import "time"

func cartTerminationEffectSpec() EffectSpec {
	spec := funcEffectSpec("CartTermination", 200*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/whitelight.tga"
	spec.SFX = []string{"effect\\wizard_fire_pillar_b.wav"}
	spec.SFXDelays = []time.Duration{30 * EffectFrameDuration}
	return spec
}

func meltdownEffectSpec() EffectSpec {
	return strEffectSpecAttached("melt", "effect\\black_overthrust.wav", false)
}

func cartBoostEffectSpec() EffectSpec {
	return strEffectSpecAttached("cart", "effect\\ef_incagility.wav", false)
}
