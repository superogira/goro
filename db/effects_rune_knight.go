package db

func m05EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef05", "dragon_breath.wav", true, false)
}

func ignitionBreakEffectSpec() EffectSpec {
	return strEffectSpecAttached("이그니션브레이크", "effect\\wl_jackfrost.wav", false)
}

func dragonHowlingEffectSpec() EffectSpec {
	return strEffectSpecAttached("dragon_h", "dragon_h.wav", false)
}

func millenniumShieldEffectSpec() EffectSpec {
	return strEffectSpecAttached("mil_shield", "", false)
}

func berserkPotion2EffectSpec() EffectSpec {
	return strEffectSpecAttached("버서크", "", false)
}

func healNEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\기공포.wav")
}

func runMakeOKEffectSpec() EffectSpec {
	return strEffectSpecAttached("rune_success", "", false)
}

func runMakeFailureEffectSpec() EffectSpec {
	return strEffectSpecAttached("rune_fail", "", false)
}

func rkLuxAnimaEffectSpec() EffectSpec {
	return strEffectSpecAttached("rk_luxanima", "", false)
}
