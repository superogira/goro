package db

import (
	"image/color"
	"time"
)

func hatedEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SoulLinkerParticles",
			TextureFiles: []string{"effect/pok1.tga", "effect/pok3.tga"},
			Color:        color.RGBA{150, 150, 250, 255}, AttachedEntity: true,
		}},
	}
}

func healParticleComponent(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/pok3.tga",
		Duration:        duration,
		Delay:           delay,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		Sparkling:       sparkling,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       9 * EffectPixelRatio,
		SizeEnd:         9 * EffectPixelRatio,
		SizeRand:        2 * EffectPixelRatio,
		Duplicate:       duplicate,
		BlendAdditive:   true,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func incAgilityParticleComponent(alpha float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/ac_center2.tga",
		Duration:        1000 * time.Millisecond,
		Delay:           delay,
		DuplicateDelay:  200 * time.Millisecond,
		AlphaMax:        alpha,
		FadeOut:         true,
		PosXRand:        1.5,
		PosYRand:        1,
		PosZStartRand:   1,
		PosZStartMiddle: 1,
		PosZEndRand:     1,
		PosZEndMiddle:   6,
		SizeStartX:      2.5 * EffectPixelRatio,
		SizeEndX:        2.5 * EffectPixelRatio,
		SizeRandY:       15 * EffectPixelRatio,
		SizeRandYMiddle: 45 * EffectPixelRatio,
		Duplicate:       duplicate,
	}
}
