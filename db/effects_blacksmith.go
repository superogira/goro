package db

import "time"

func hasteUpEffectSpec() EffectSpec {
	spec := funcEffectSpec("HasteUp", 100*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.Components[0].SpriteFile = "particle1"
	spec.SFX = []string{"effect\\black_adrenalinerush_a.wav", "effect\\black_adrenalinerush_b.wav"}
	spec.SFXDelays = []time.Duration{0, 30 * EffectFrameDuration}
	return spec
}

func overthrustEffectSpec() EffectSpec {
	spec := funcEffectSpec("Overthrust", 40*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\black_overthrust.wav", "effect\\EF_StoneCurse.wav"}
	return spec
}

func repairWeaponEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("repairweapon", "", false)
	spec.SFX = []string{"effect\\black_weapon_repair_a.wav", "effect\\black_weapon_repair_a.wav"}
	spec.SFXDelays = []time.Duration{20 * EffectFrameDuration, 55 * EffectFrameDuration}
	return spec
}

func crashEarthEffectSpec() EffectSpec {
	spec := strEffectSpec("crashearth", "effect\\wizard_fire_pillar_b.wav")
	spec.SFXDelays = []time.Duration{30 * EffectFrameDuration}
	spec.CameraShake = 650 * time.Millisecond
	spec.CameraShakeDelay = 35 * EffectFrameDuration
	return spec
}

func maximizePowerEffectSpec() EffectSpec {
	spec := strEffectSpecAttachedMin("maximizepower", "maximize_min", "", false)
	spec.SFX = []string{
		"effect\\black_maximize_power_circle.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword_bic.wav",
	}
	spec.SFXDelays = []time.Duration{0, 40 * EffectFrameDuration, 47 * EffectFrameDuration, 65 * EffectFrameDuration}
	return spec
}

func adrenalineCastEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\black_adrenalinerush_a.wav")
}

func weaponPerfectEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("weaponperfection", "weaponperfection_min", "effect\\black_weapon_perfection.wav", false)
}

func maximizeSoundsEffectSpec() EffectSpec {
	return delayedSoundEffectSpecDuration(950*time.Millisecond,
		[]string{
			"effect\\black_maximize_power_circle.wav",
			"effect\\black_maximize_power_sword.wav",
			"effect\\black_maximize_power_sword.wav",
			"effect\\black_maximize_power_sword_bic.wav",
		},
		[]time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond},
	)
}

func greedSoundEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\ef_entry.wav")
}

func refineOKEffectSpec() EffectSpec {
	return strEffectSpecAttached("bs_refinesuccess", "effect\\bs_refinesuccess.wav", false)
}

func refineFailEffectSpec() EffectSpec {
	return strEffectSpecAttached("bs_refinefailed", "effect\\bs_refinefailed.wav", false)
}
