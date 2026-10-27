package db

import "time"

func bottomBasilicaEffectSpec() EffectSpec {
	return groundWallsEffectSpec("effect/alpha_down.tga", 29999)
}

func assumptioEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 2*time.Second, true)
}

func assumptio2EffectSpec() EffectSpec {
	return strEffectSpecAttached("asum", "effect\\아숨프티오.wav", false)
}
