package db

import (
	"image/color"
	"time"
)

func agilityEffectSpec(decrease bool) EffectSpec {
	spec := funcEffectSpec("Agility", 100*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_incagility.wav"}
	spec.Components[0].TextureFile = "effect/agi_up.bmp"
	spec.Components[0].TextureFiles = []string{"effect/ac_center2.tga"}
	if decrease {
		spec.SFX = []string{"effect\\ef_decagility.wav"}
		spec.Components[0].TextureFile = "effect/slow.bmp"
	}
	return spec
}

func healCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 178, G: 255, B: 178, A: 255},
		TextureName:      "ring_white",
		Duration:         1500 * time.Millisecond,
		AlphaMax:         0.2,
		Fade:             true,
		Rotate:           true,
		Animation:        1,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendAdditive:    true,
	}
}

func healOffensiveCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	component := healCylinderComponent(bottomSize, topSize, height)
	component.Duration = time.Second
	component.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	component.BlendAdditive = true
	return component
}

// Acolyte Heal selects these larger variants by heal amount; Arch Bishop also reuses them.
func heal2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 15),
			heal2CylinderComponent(1, 1, 13),
			heal2CylinderComponent(1.1, 3, 2),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 2, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, 9, 2, 2, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, 9, 2, 0, false),
		},
	}
}

func heal4EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2 * time.Second,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 18),
			heal2CylinderComponent(1, 1, 15),
			heal2CylinderComponent(1.1, 3, 3),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 7, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 3, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 25, 1.5, 1.5, 0, 0, 0, 3, 6, 10, 5, 3, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 15, 1, 1, 1, 0, 6, 0, 0, 11, 2, 0, false),
		},
	}
}

func heal2CylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return robrCylinderBlendComponent("ring_white", color.RGBA{R: 178, G: 255, B: 178, A: 255}, 1500*time.Millisecond, 0.3, 1, bottomSize, topSize, height, true, true, true, 2)
}

func healSparkParticle(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle, size, sizeRand float64, sparkNumber int, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		TextureFile:     "effect/pok3.tga",
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Duration:        duration,
		Delay:           delay,
		Duplicate:       duplicate,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       effectTableSize(size),
		SizeEnd:         effectTableSize(size),
		SizeRand:        effectTableSize(sizeRand),
		BlendMode:       2,
		BlendAdditive:   true,
		AttachedEntity:  true,
		Sparkling:       sparkling,
		SparkNumber:     sparkNumber,
	}
	return component
}

func pneumaEffectSpec() EffectSpec {
	return strEffectSpecRandom("pneuma%d", "", 1, 3)
}

func ruwachEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 12200 * time.Millisecond,
		SFX:      []string{"effect\\ef_sight.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "data\\sprite\\shadow",
				ShadowTexture:   true,
				Duration:        12200 * time.Millisecond,
				AlphaMax:        0.5,
				PosX:            -2,
				SizeStart:       effectTableSize(0.4),
				SizeEnd:         effectTableSize(0.4),
				SizeDelta:       0.15,
				Duplicate:       8,
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  8,
				OrbitPhase:      0.7,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
			},
			{
				Kind:            EffectComponent3D,
				Color:           color.RGBA{G: 255, B: 255, A: 255},
				SpriteFile:      "particle2",
				Duration:        12200 * time.Millisecond,
				AlphaMax:        0.07,
				AlphaMaxDelta:   0.07,
				BlendAdditive:   true,
				PosX:            -2,
				PosZ:            2,
				SizeStart:       effectTableSize(80),
				SizeEnd:         effectTableSize(80),
				SizeDelta:       30,
				Duplicate:       10,
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  8,
				OrbitPhase:      0.7,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
			},
		},
	}
}

func incAgilityEffectSpec() EffectSpec {
	return agilityEffectSpec(false)
}

func decAgilityEffectSpec() EffectSpec {
	return agilityEffectSpec(true)
}

func aquaEffectSpec() EffectSpec {
	return EffectSpec{
		SFX: []string{"effect\\ef_aqua.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentSPR,
			SpriteFile:       "성수뜨기",
			SpriteDelay:      2 * EffectFrameDuration,
			PosZ:             4,
			AttachedEntity:   true,
			WorldSizedSprite: true,
		}},
	}
}

func signumEffectSpec() EffectSpec {
	return strEffectSpecAttached("cross", "effect\\ef_signum.wav", false)
}

func angelusEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("angelus", "jong_mini", "effect\\ef_angelus.wav", true)
}

func blessingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2500 * time.Millisecond,
		SFX:      []string{"effect\\ef_blessing.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentSPR,
				SpriteFile:       "축복",
				Duration:         1500 * time.Millisecond,
				SpriteDelay:      30 * time.Millisecond,
				SpriteRepeat:     true,
				SpriteHead:       true,
				SpriteYOffset:    -120,
				WorldSizedSprite: true,
			},
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "particle6",
				Duration:        1200 * time.Millisecond,
				Delay:           300 * time.Millisecond,
				Duplicate:       6,
				DuplicateDelay:  0,
				AlphaMax:        1,
				Sparkling:       true,
				SparkNumber:     2,
				FadeIn:          true,
				FadeOut:         true,
				PosXRand:        1.2,
				PosYRand:        1,
				PosZStartRand:   2,
				PosZStartMiddle: 5.5,
				PosZEndRand:     0.5,
				PosZEndMiddle:   1,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
			},
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "particle6",
				Duration:        1200 * time.Millisecond,
				Delay:           400 * time.Millisecond,
				Duplicate:       6,
				DuplicateDelay:  0,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				PosXRand:        1.4,
				PosYRand:        1.1,
				PosZStartRand:   2,
				PosZStartMiddle: 5.5,
				PosZEndRand:     0.5,
				PosZEndMiddle:   1,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "BlessingCircle",
				TextureName:    "alpha_down",
				Color:          color.RGBA{R: 0x20, G: 0xb0, B: 0xe8, A: 255},
				Duration:       150 * EffectFrameDuration,
				FrameDelay:     EffectFrameDuration,
				AlphaMax:       100.0 / 255,
				SizeStart:      2,
				CircleSides:    10,
				Overlay:        true,
				AttachedEntity: true,
			},
		},
	}
}

func cureEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("cure", "cure_min", "effect\\acolyte_cure.wav", false)
}

func holyLightEffectSpec() EffectSpec {
	return strEffectSpecAttached("holyhit", "", false)
}

func healEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1840 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			healCylinderComponent(0.95, 0.95, 8),
			healCylinderComponent(1.0, 1.0, 8),
			healParticleComponent(0.6, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 15, 1.5, 1.5, 0, 0, 0, 2, 6, false),
			healParticleComponent(0.6, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 7, 1, 1, 1, 0, 5, 0, 0, false),
		},
	}
}

func healOffensiveEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1490 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			healOffensiveCylinderComponent(0.95, 0.95, 10),
			healOffensiveCylinderComponent(1.0, 1.0, 9),
			healParticleComponent(0.8, 1000*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 10, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			healParticleComponent(0.8, 900*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 5, 1, 1, 1, 0, 6, 0, 0, true),
		},
	}
}
