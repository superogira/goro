package db

func hfliMoon1EffectSpec() EffectSpec {
	return strEffectSpecAttached("moonlight_1", "effect\\h_moonlight_1.wav", false)
}

func hfliMoon2EffectSpec() EffectSpec {
	return strEffectSpecAttached("moonlight_2", "effect\\h_moonlight_2.wav", false)
}

func hfliMoon3EffectSpec() EffectSpec {
	return strEffectSpecAttached("moonlight_3", "effect\\h_moonlight_3.wav", false)
}

func hoUpEffectSpec() EffectSpec {
	return strEffectSpecAttached("h_levelup", "", false)
}

func hamiDefenceEffectSpec() EffectSpec {
	return strEffectSpecAttached("defense", "", false)
}

func hamiCastleEffectSpec() EffectSpec {
	return sprEffectSpec("캐슬링", "", true, false)
}

func hamiBloodEffectSpec() EffectSpec {
	return sprEffectSpec("블러드러스트", "", true, false)
}

func poisonMistEffectSpec() EffectSpec {
	return strEffectSpecAttached("poison_mist", "", false)
}

func eraserCutterEffectSpec() EffectSpec {
	return strEffectSpecAttached("eraser_cutter", "", false)
}

func lavaSlideEffectSpec() EffectSpec {
	return strEffectSpecAttached("lava_slide", "", false)
}

func sonicClawEffectSpec() EffectSpec {
	return strEffectSpecAttached("sonic_claw", "", false)
}

func tinderBreakerEffectSpec() EffectSpec {
	return strEffectSpecAttached("tinder", "", false)
}

func midnightFrenzyEffectSpec() EffectSpec {
	return strEffectSpecAttached("mid_frenzy", "", false)
}

func volcanicAshEffectSpec() EffectSpec {
	return strEffectSpecAttached("vash00", "", false)
}
