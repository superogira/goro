package db

import "time"

func detectingEffectSpec() EffectSpec {
	spec := funcEffectSpec("Detecting", 57*EffectFrameDuration, false)
	spec.SFX = []string{"effect\\hunter_detecting.wav"}
	spec.Components[0].TextureFile = "effect/fashasha.tga"
	return spec
}

func removeTrapEffectSpec() EffectSpec {
	spec := funcEffectSpec("RemoveTrap", 57*EffectFrameDuration, false)
	spec.SFX = []string{"effect\\hunter_removetrap.wav"}
	spec.Components[0].SpriteFile = "particle1"
	return spec
}

func flasherEffectSpec() EffectSpec {
	spec := funcEffectSpec("Flasher", 70*EffectFrameDuration, false)
	spec.SFX = []string{"effect\\hunter_flasher.wav"}
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].TextureFiles = []string{"effect/alpha_center.tga"}
	return spec
}

func blitzBeatEffectSpec() EffectSpec {
	spec := funcEffectSpec("BlitzBeat", 250*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\hunter_blitzbeat_1st.wav", "effect\\hunter_blitzbeat.wav"}
	spec.SFXDelays = []time.Duration{5 * EffectFrameDuration, 30 * EffectFrameDuration}
	spec.Components[0].TextureFile = "effect/ac_center2.tga"
	spec.Components[0].TextureFiles = []string{"effect/magic_green.tga"}
	return spec
}

func blastMineEffectSpec() EffectSpec {
	spec := strEffectSpec("blastmine", "effect\\hunter_blastmine.wav")
	spec.SFXDelays = []time.Duration{20 * EffectFrameDuration}
	return spec
}

func skidTrapEffectSpec() EffectSpec {
	return strEffectSpec("skidtrap", "effect\\hunter_skidtrap.wav")
}

func blastMineSoundEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\hun_anklesnare.wav")
}

func claymoreEffectSpec() EffectSpec {
	return strEffectSpec("claymore", "effect\\hunter_claymoretrap.wav")
}

func freezingTrapEffectSpec() EffectSpec {
	return strEffectSpec("freezing", "effect\\hunter_freezingtrap.wav")
}

func springTrapEffectSpec() EffectSpec {
	return strEffectSpec("spring", "effect\\hunter_springtrap.wav")
}

func sandmanEffectSpec() EffectSpec {
	return strEffectSpec("sandman", "effect\\hunter_sandman.wav")
}

func shockwaveEffectSpec() EffectSpec {
	return strEffectSpec("shockwave", "")
}

func shockwaveHitEffectSpec() EffectSpec {
	return strEffectSpecAttached("shockwavehit", "", false)
}
