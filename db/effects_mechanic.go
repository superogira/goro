package db

func crashAxeEffectSpec() EffectSpec {
	return strEffectSpecAttached("powerswing", "", false)
}

func ncMagmaEruptionEffectSpec() EffectSpec {
	return strEffectSpecAttached("nc_magma_eruption", "effect\\nc_magma_eruption.wav", false)
}
