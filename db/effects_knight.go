package db

import "time"

func pierceEffectSpec() EffectSpec {
	// Am_Attacked gives EF_PIERCE a count of one, so only its first ring runs.
	return EffectSpec{
		Duration:   15 * EffectFrameDuration,
		Components: []EffectComponent{hitRingComponent("ring_yellow", 15, 6, 11, 2.5, .7, 8)},
	}
}

func bowlingBashEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 50 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "BowlingBash",
			TextureFile: "effect/ring_blue.tga", TextureFiles: []string{"effect/ring_yellow.tga"},
		}},
	}
}

func spearBoomerangEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  28 * EffectFrameDuration,
		SFX:       []string{"effect\\ef_fireball.wav", "effect\\ef_fireball.wav", "effect\\ef_fireball.wav", "effect\\ef_fireball.wav"},
		SFXDelays: []time.Duration{0, 4 * EffectFrameDuration, 8 * EffectFrameDuration, 12 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SpearBoomerang", SpriteFile: "창",
		}},
	}
}

func brandishSpearEffectSpec() EffectSpec {
	return strEffectSpecAttached("brandish", "", false)
}

func twoHandQuickenEffectSpec() EffectSpec {
	return strEffectSpecAttached("twohand", "effect\\knight_twohandquicken.wav", false)
}

func autoCounterEffectSpec() EffectSpec {
	return strEffectSpecAttached("autocounter", "effect\\knight_autocounter.wav", false)
}

func brandishSpear2EffectSpec() EffectSpec {
	return strEffectSpecAttached("brandish2", "", false)
}

func pierceSelfEffectSpec() EffectSpec {
	return strEffectSpecAttached("pierce", "", false)
}

func bowlingSelfEffectSpec() EffectSpec {
	return strEffectSpecAttached("bowling", "", false)
}

func spearStabSelfEffectSpec() EffectSpec {
	return strEffectSpecAttached("spearstab", "", false)
}

func spearBmrSelfEffectSpec() EffectSpec {
	return strEffectSpecAttached("spearboomerang", "", false)
}
