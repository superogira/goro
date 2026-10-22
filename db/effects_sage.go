package db

import "time"

func landProtectorGroundEffectSpec() EffectSpec {
	spec := funcEffectSpec("LandProtectorGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/aaa copy.bmp"
	return spec
}

func sageEndowEffectSpec(file string) EffectSpec {
	spec := strEffectSpecAttached(file, "_enemy_hit_wind1.wav", false)
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration}
	return spec
}

func spellBreakerEffectSpec() EffectSpec {
	return strEffectSpecAttached("spell", "effect\\sage_spell breake.wav", false)
}

func dispellEffectSpec() EffectSpec {
	return strEffectSpecAttached("디스펠", "", false)
}

func volcanoEffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_red", 300)
}

func delugeEffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_blue", 999)
}

func violentGaleEffectSpec() EffectSpec {
	return groundCastEffectSpec("magic_green", 999)
}

func landProtectorEffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_white", 999)
}

func bottomVolcanoEffectSpec() EffectSpec {
	return propertyGroundEffectSpec("ring_red")
}

func bottomDelugeEffectSpec() EffectSpec {
	return propertyGroundEffectSpec("ring_blue")
}

func bottomViolentEffectSpec() EffectSpec {
	return propertyGroundEffectSpec("magic_green")
}

func magicRodEffectSpec() EffectSpec {
	return strEffectSpecAttached("매직로드", "effect\\sage_magic rod.wav", false)
}

func flameLauncherEffectSpec() EffectSpec {
	return sageEndowEffectSpec("enc_fire")
}

func frostWeaponEffectSpec() EffectSpec {
	return sageEndowEffectSpec("enc_ice")
}

func lightningLoadEffectSpec() EffectSpec {
	return sageEndowEffectSpec("enc_wind")
}

func seismicWeaponEffectSpec() EffectSpec {
	return sageEndowEffectSpec("enc_earth")
}
