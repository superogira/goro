package db

import (
	"image/color"
	"strconv"
	"time"
)

func teiHitEffectSpec(asura bool) EffectSpec {
	spec := funcEffectSpec("MonkRays", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.Components[0].Duplicate = 80
	if asura {
		spec.Components[0].TextureFile = "effect/lens1.tga"
		spec.Components[0].Duplicate = 96
	}
	return spec
}

func teiHit2EffectSpec() EffectSpec {
	spec := funcEffectSpec("ImpactParticles", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].Duplicate = 20
	return spec
}

func tanjiEffectSpec() EffectSpec {
	spec := funcEffectSpec("Tanji", 500*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/blue_ivy.bmp"
	spec.SFX = []string{"effect\\mon_탄지신통.wav"}
	return spec
}

func tripleAttackEffectSpec() EffectSpec {
	spec := funcEffectSpec("TripleAttack", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/cloud11.tga"
	spec.SFX = []string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"}
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration, 20 * EffectFrameDuration, 30 * EffectFrameDuration}
	return spec
}

func hitLineEffectSpec() EffectSpec {
	spec := funcEffectSpec("HitLine", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/white02.bmp"
	return spec
}

func gumgangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 99999 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Gumgang",
			TextureFiles:   []string{"effect/super1.bmp", "effect/super2.bmp", "effect/super3.bmp", "effect/super4.bmp", "effect/super5.bmp"},
			Color:          color.RGBA{R: 100, G: 100, B: 255, A: 255},
			AttachedEntity: true,
		}},
	}
}

func chimtoEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 16 * EffectFrameDuration,
		SFX:      []string{"effect\\mon_침투경.wav"},
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			SpriteFile:  "침투경_", // misc/chimto in the original source's resource aliases
			SpriteDelay: 2 * EffectFrameDuration,
			Duration:    16 * EffectFrameDuration,
			AlphaMax:    200.0 / 255,
			SizeStart:   effectTableSize(250),
			SizeEnd:     effectTableSize(250),
		}},
	}
}

func absorbSpiritsEffectSpec() EffectSpec {
	tint := color.RGBA{R: 77, G: 77, B: 255, A: 255}
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"effect\\흡기.wav"},
		Components: []EffectComponent{
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 1.1, 15, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1, 1, 13, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 3, 2, true, true, true, 2),
			absorbSpiritParticle(tint, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, true),
			absorbSpiritParticle(tint, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			absorbSpiritParticle(tint, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, false),
		},
	}
}

func absorbSpiritParticle(tint color.RGBA, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:             EffectComponent3D,
		TextureFile:      "effect/pok3.tga",
		Color:            tint,
		Duration:         duration,
		Delay:            delay,
		Duplicate:        duplicate,
		DuplicateDelay:   duplicateDelay,
		AlphaMax:         0.8,
		FadeIn:           true,
		FadeOut:          true,
		PosXRand:         posXRand,
		PosYRand:         posYRand,
		PosZStartRand:    posZStartRand,
		PosZStartMiddle:  posZStartMiddle,
		PosZEnd:          posZEnd,
		PosZEndRand:      posZEndRand,
		PosZEndMiddle:    posZEndMiddle,
		SizeStart:        effectTableSize(9),
		SizeEnd:          effectTableSize(9),
		SizeRand:         effectTableSize(2),
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Sparkling:        sparkling,
		RotateWithCamera: false,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func gumgangRingEffectSpec(duration time.Duration, alphaMax, bottomSize, topSize float64, wav string) EffectSpec {
	component := robrCylinderBlendComponent("ring_yellow", color.RGBA{}, duration, alphaMax, 4, bottomSize, topSize, 2, true, true, true, 8)
	component.Duplicate = 4
	component.DuplicateDelay = 100 * time.Millisecond
	spec := EffectSpec{
		Duration:   duration + 300*time.Millisecond,
		Components: []EffectComponent{component},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func beginAsuraEffectSpec(champion bool) EffectSpec {
	spec := blueCastingEffectSpec()
	spec.Duration = 350 * EffectFrameDuration
	spec.Components[0].TextureFile = "effect/ring_white.tga"
	glyphs := EffectComponent{Kind: EffectComponentFUNC, FuncName: "AsuraGlyphs"}
	for i := 1; i <= 6; i++ {
		index := i
		if champion {
			index += 10
		}
		glyphs.TextureFiles = append(glyphs.TextureFiles, "effect/asura"+strconv.Itoa(index)+".tga")
	}
	spec.Components = append(spec.Components, glyphs)
	return spec
}

func steelBodyEffectSpec() EffectSpec {
	spec := gumgangEffectSpec()
	// A packet effect is finite; the Steel Body status keeps its aura alive.
	spec.Duration = 2 * time.Second
	spec.Components[0].Duplicate = 4
	spec.SFX = []string{"effect\\mon_금강불괴.wav"}
	return spec
}

func steelBodyRingEffectSpec() EffectSpec {
	return gumgangRingEffectSpec(1500*time.Millisecond, 0.5, 2, 5, "")
}

func gumgang2EffectSpec() EffectSpec {
	return gumgangRingEffectSpec(1500*time.Millisecond, 0.5, 2, 5, "")
}

func teiHit1EffectSpec() EffectSpec {
	return teiHitEffectSpec(false)
}

func gumgang3EffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_yellow", 300)
}

func teiHit1XEffectSpec() EffectSpec {
	return teiHitEffectSpec(true)
}

func chainComboEffectSpec() EffectSpec {
	return strEffectSpecAttached("연환", "effect\\mon_연환.wav", false)
}

func beginAsuraCastEffectSpec() EffectSpec {
	return beginAsuraEffectSpec(false)
}

func teiHit1TEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 350 * time.Millisecond,
		SFX:      []string{"effect\\mon_아수라 패황권.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/lens1.tga",
			Color:            effectTableColor(0.1, 0.1, 1),
			Duration:         250 * time.Millisecond,
			Delay:            100 * time.Millisecond,
			Duplicate:        24,
			DuplicateDelay:   0,
			AlphaMax:         0.8,
			BlendMode:        2,
			BlendAdditive:    true,
			FadeIn:           true,
			FadeOut:          true,
			AttachedEntity:   true,
			PosXEndRand:      40,
			PosYEndRand:      40,
			SizeStartX:       effectTableSize(10),
			SizeStartY:       effectTableSize(150),
			SizeEndX:         effectTableSize(10),
			SizeEndY:         effectTableSize(150),
			Overlay:          true,
			RotateToTarget:   true,
			RotateWithCamera: true,
		}},
	}
}
