package db

import (
	"image/color"
	"time"
)

func healSPEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2000 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{R: 25, G: 128, B: 255, A: 255}, 2000*time.Millisecond, 0.2, 1, 0.95, 0.95, 10, true, true, true, true),
			tintedEffectComponent(healParticleComponent(0.8, 1000*time.Millisecond, 400*time.Millisecond, 80*time.Millisecond, 6, 1.5, 1.5, 0, 0, 0, 3, 6, true), color.RGBA{R: 25, G: 128, B: 255, A: 255}),
			tintedEffectComponent(healParticleComponent(0.8, 900*time.Millisecond, 200*time.Millisecond, 200*time.Millisecond, 3, 1, 1, 1, 0, 6, 0, 0, true), color.RGBA{R: 25, G: 128, B: 255, A: 255}),
		},
	}
}

func hpTimeEffectSpec() EffectSpec {
	return naturalRecoveryEffectSpec(color.RGBA{R: 230, G: 255, B: 230, A: 255}, "_heal_effect.wav")
}

func spTimeEffectSpec() EffectSpec {
	return naturalRecoveryEffectSpec(color.RGBA{R: 230, G: 230, B: 255, A: 255}, "effect\\흡기.wav")
}

func naturalRecoveryEffectSpec(tint color.RGBA, wav string) EffectSpec {
	return EffectSpec{
		Duration: 1110 * time.Millisecond,
		SFX:      []string{wav},
		Components: []EffectComponent{{
			Kind:            EffectComponent3D,
			TextureFile:     "effect/pok1.tga",
			Color:           tint,
			Duration:        500 * time.Millisecond,
			Delay:           500 * time.Millisecond,
			Duplicate:       12,
			DuplicateDelay:  10 * time.Millisecond,
			AlphaMax:        0.8,
			FadeIn:          true,
			FadeOut:         true,
			SizeStart:       effectTableSize(30),
			SizeEnd:         effectTableSize(30),
			SizeRand:        effectTableSize(20),
			BlendMode:       2,
			BlendAdditive:   true,
			PosXRand:        0.6,
			PosYRand:        0.6,
			PosZStartRand:   1.5,
			PosZStartMiddle: 2,
			PosZEndRand:     1,
			PosZEndMiddle:   5,
			Sparkling:       true,
			SparkNumber:     3,
			AttachedEntity:  true,
		}},
	}
}
