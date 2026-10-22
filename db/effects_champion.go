package db

import "time"

func palmStrikeEffectSpec() EffectSpec {
	spec := hitLineEffectSpec()
	spec.Duration = 200 * EffectFrameDuration
	return spec
}

func tigerFistEffectSpec() EffectSpec {
	spec := sacrificeEffectSpec()
	spec.Duration = 200 * EffectFrameDuration
	spec.SFX = []string{"effect\\mon_맹룡과강.wav"}
	spec.SFXDelays = []time.Duration{5 * EffectFrameDuration}
	return spec
}

func chainCrushEffectSpec() EffectSpec {
	spec := funcEffectSpec("ChainCrush", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/line3y.tga"
	spec.SFX = []string{"effect\\연주붕격.wav"}
	spec.SFXDelays = []time.Duration{44 * EffectFrameDuration}
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 44 * EffectFrameDuration
	return spec
}

func portal5EffectSpec() EffectSpec {
	spec := windWalkEffectSpec()
	spec.SFX = nil
	return spec
}

func beginAsura11EffectSpec() EffectSpec {
	return beginAsuraEffectSpec(true)
}
