package db

import "time"

func arrowVulcanEffectSpec() EffectSpec {
	spec := funcEffectSpec("ArrowVulcan", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\애로우 발칸.wav"}
	spec.SFXDelays = []time.Duration{30 * EffectFrameDuration}
	return spec
}

func moonlitEffectSpec() EffectSpec {
	spec := funcEffectSpec("Moonlit", 99999*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/fire_ivy.bmp"
	return spec
}

func marionetteEffectSpec() EffectSpec {
	spec := funcEffectSpec("Marionette", 99999999*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	spec.SFX = []string{"effect\\소울 체인지.wav"}
	return spec
}

func hermodeEffectSpec() EffectSpec {
	spec := funcEffectSpec("Hermode", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/white02.bmp"
	return spec
}

func tarotCardEffectSpec(index int) EffectSpec {
	spec := funcEffectSpec("TarotCard", 300*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/tarot" + twoDigitString(index) + ".tga"
	return spec
}

func hermodeMusicEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\헤르모드의 지팡이.wav")
}

func tarotCard1EffectSpec() EffectSpec {
	return tarotCardEffectSpec(1)
}

func tarotCard2EffectSpec() EffectSpec {
	return tarotCardEffectSpec(2)
}

func tarotCard3EffectSpec() EffectSpec {
	return tarotCardEffectSpec(3)
}

func tarotCard4EffectSpec() EffectSpec {
	return tarotCardEffectSpec(4)
}

func tarotCard5EffectSpec() EffectSpec {
	return tarotCardEffectSpec(5)
}

func tarotCard6EffectSpec() EffectSpec {
	return tarotCardEffectSpec(6)
}

func tarotCard7EffectSpec() EffectSpec {
	return tarotCardEffectSpec(7)
}

func tarotCard8EffectSpec() EffectSpec {
	return tarotCardEffectSpec(8)
}

func tarotCard9EffectSpec() EffectSpec {
	return tarotCardEffectSpec(9)
}

func tarotCard10EffectSpec() EffectSpec {
	return tarotCardEffectSpec(10)
}

func tarotCard11EffectSpec() EffectSpec {
	return tarotCardEffectSpec(11)
}

func tarotCard12EffectSpec() EffectSpec {
	return tarotCardEffectSpec(12)
}

func tarotCard13EffectSpec() EffectSpec {
	return tarotCardEffectSpec(13)
}

func tarotCard14EffectSpec() EffectSpec {
	return tarotCardEffectSpec(14)
}
