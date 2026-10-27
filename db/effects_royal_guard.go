package db

func pressure2EffectSpec() EffectSpec {
	return pressureDropEffectSpec("effect/shield.bmp", "effect\\lg_shieldpress.wav")
}

func primeCharge2EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\lg_prestige.wav")
}

func primeCharge3EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\lg_banding.wav")
}

func primeCharge4EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\lg_inspiration.wav")
}
