package db

import (
	"image/color"
	"time"
)

func stealEffectSpec() EffectSpec {
	spec := funcEffectSpec("Steal", 50*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = "particle7"
	spec.SFX = []string{"effect\\ef_steal.wav"}
	return spec
}

func detoxicationEffectSpec() EffectSpec {
	spec := funcEffectSpec("RisingParticles", 100*EffectFrameDuration, false)
	spec.Components[0].SpriteFile = "particle2"
	spec.SFX = []string{"effect\\ef_detoxication.wav"}
	return spec
}

func sprinkleSandEffectSpec() EffectSpec {
	spec := funcEffectSpec("SprinkleSand", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	return spec
}

func throwItem3EffectSpec() EffectSpec {
	return throwItemFlightEffectSpec("유저인터페이스/item/돌.bmp")
}

func poisonAttackEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2800 * time.Millisecond,
		SFX:      []string{"effect\\ef_detoxication.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			Color:          color.RGBA{R: 255, G: 178, B: 255, A: 255},
			TextureFile:    "effect/pok1.tga",
			Duration:       1000 * time.Millisecond,
			AlphaMax:       1,
			FadeIn:         true,
			FadeOut:        true,
			PosXRand:       1,
			PosYRand:       1,
			PosZEnd:        5,
			SizeStart:      100 * EffectPixelRatio,
			SizeEnd:        100 * EffectPixelRatio,
			SizeRand:       20 * EffectPixelRatio,
			Duplicate:      10,
			DuplicateDelay: 200 * time.Millisecond,
		}},
	}
}

func invenomEffectSpec() EffectSpec {
	return strEffectSpecAttached("invenom", "effect\\thief_invenom.wav", false)
}
