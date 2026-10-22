package db

func electric4EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\sr_earthshaker.wav")
}

func srFlashComboEffectSpec() EffectSpec {
	return strEffectSpecAttached("sr_flashcombo", "effect\\sr_flashcombo.wav", false)
}
