package db

import "time"

func hpConversionEffectSpec() EffectSpec {
	spec := funcEffectSpec("HPConversion", 200*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "particle1"
	return spec
}

func soulChangeLinkEffectSpec() EffectSpec {
	spec := funcEffectSpec("SoulChangeLink", 200*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\소울 체인지.wav"}
	return spec
}

func magicCrasher2EffectSpec() EffectSpec {
	spec := funcEffectSpec("EffectBodyColor", 100*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\swordman_provoke.wav"}
	spec.SFXDelays = []time.Duration{25 * EffectFrameDuration}
	return spec
}

func memorizeEffectSpec() EffectSpec {
	spec := funcEffectSpec("EffectBodyColor", 200*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\priest_suffragium.wav"}
	return spec
}

func spiderWebEffectSpec() EffectSpec {
	spec := landProtectorGroundEffectSpec()
	spec.Components[0].TextureFile = "effect/spiderweb.tga"
	return spec
}

func fogWallGroundEffectSpec() EffectSpec {
	spec := funcEffectSpec("PriestGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/ring_white.tga"
	return spec
}

func linkLightEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 200*EffectFrameDuration, true)
}

func soulBurnEffectSpec() EffectSpec {
	return strEffectSpecAttached("소울번", "", false)
}

func soulChangeEffectSpec() EffectSpec {
	return strEffectSpecAttached("사람효과", "", false)
}

func doubleCastBodyEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 120*EffectFrameDuration, true)
}
