package db

func fireWall2EffectSpec() EffectSpec {
	return strEffectSpecAttached("firewall_per", "", false)
}

func sprPlant10EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\s사이킥웨이브.wav")
}

func shockwave2EffectSpec() EffectSpec {
	return strEffectSpecAttached("hunter_shockwave_blue", "", false)
}

func coldThrow2EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\wl_jackfrost.wav")
}

func demonicFire4EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\s워머.wav")
}

func pressure3EffectSpec() EffectSpec {
	return pressureDropEffectSpec("effect/cross1.bmp", "")
}

func miResultMakeOKEffectSpec() EffectSpec {
	return strEffectSpecAttached("changematerial_su", "", false)
}

func miResultMakeFailEffectSpec() EffectSpec {
	return strEffectSpecAttached("changematerial_fa", "", false)
}

func soElemShieldEffectSpec() EffectSpec {
	return strEffectSpecAttached("so_elemental_shield", "effect\\so_elemental_shield.wav", false)
}
