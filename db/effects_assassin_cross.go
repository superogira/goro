package db

import "time"

func soulBreakerEffectSpec(meteor bool) EffectSpec {
	spec := funcEffectSpec("SoulBreaker", 300*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/purpleslash.tga"
	spec.SFX = []string{"effect\\기공포.wav"}
	spec.SFXDelays = []time.Duration{20 * EffectFrameDuration}
	if meteor {
		spec.SFX = []string{"effect\\메테오 어썰트.wav"}
		spec.SFXDelays = nil
	}
	return spec
}

func enchantDeadlyPoisonEffectSpec() EffectSpec {
	spec := funcEffectSpec("RisingParticles", 120*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "particle3"
	for _, tick := range []int{0, 5, 11, 18, 26, 40} {
		spec.SFX = append(spec.SFX, "effect\\assasin_enchantpoison.wav")
		spec.SFXDelays = append(spec.SFXDelays, time.Duration(tick)*EffectFrameDuration)
	}
	return spec
}

func soulBreakerBaseEffectSpec() EffectSpec {
	return soulBreakerEffectSpec(false)
}

func soulBreaker2EffectSpec() EffectSpec {
	return soulBreakerEffectSpec(true)
}
