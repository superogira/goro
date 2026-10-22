package db

import "time"

func concentrationBeginEffectSpec() EffectSpec {
	spec := agilityEffectSpec(false)
	spec.SFX = []string{"effect\\ef_incagidex.wav"}
	spec.Components[0].TextureFile = "effect/dex_agi_up.bmp"
	return spec
}

func arrowShowerEffectSpec() EffectSpec {
	spec := funcEffectSpec("ArrowShower", 8*24*time.Millisecond, false)
	spec.Components[0].SpriteFile = "data/sprite/몬스터/skel_archer_arrow"
	return spec
}

func arrowShotEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 140 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/npc/skel_archer_arrow",
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Duration:         140 * time.Millisecond,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			PosZ:             1,
			SizeStart:        effectTableSize(100),
			SizeEnd:          effectTableSize(100),
			AngleStart:       180,
		}},
	}
}

func arrowShotROEffectSpec() EffectSpec {
	return strEffectSpecAttached("arrowshot", "", false)
}

func concentrationEffectSpec() EffectSpec {
	return strEffectSpecAttached("concentration", "effect\\ac_concentration.wav", false)
}
