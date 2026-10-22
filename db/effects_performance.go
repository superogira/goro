package db

func songMusicEffectSpec(texture string, variant int) EffectSpec {
	spec := funcEffectSpec("SongMusic", 29999*EffectFrameDuration, false)
	c := &spec.Components[0]
	c.TextureFile, c.Animation = texture, variant
	switch variant {
	case 2:
		c.TextureFiles = []string{"effect/melody_a.bmp", "effect/melody_b.bmp"}
	case 5:
		c.TextureFiles = []string{"effect/레드젬스톤.bmp", "effect/블루젬스톤.bmp", "effect/옐로우젬스톤.bmp"}
	case 8:
		c.TextureFiles = []string{"effect/spell_01.bmp", "effect/spell_02.bmp", "effect/spell_03.bmp", "effect/spell_04.bmp", "effect/spell_05.bmp", "effect/spell_06.bmp", "effect/spell_07.bmp", "effect/spell_08.bmp"}
	}
	return spec
}

func songVerticalEffectSpec(texture string, variant int) EffectSpec {
	spec := funcEffectSpec("SongVertical", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = texture
	spec.Components[0].Animation = variant
	return spec
}

func songLightEffectSpec(variant int) EffectSpec {
	spec := funcEffectSpec("SongLight", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/twirl.bmp"
	spec.Components[0].Animation = variant
	return spec
}

func songOutEffectSpec() EffectSpec {
	spec := funcEffectSpec("SongOut", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/safeline.bmp"
	return spec
}

func bottomLullabyEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/zz.bmp", 1)
}

func bottomRichKimEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/pocket.bmp", 3)
}

func bottomChaosEffectSpec() EffectSpec {
	return songLightEffectSpec(8)
}

func bottomDrumEffectSpec() EffectSpec {
	return songMusicEffectSpec("", 2)
}

func bottomNibelungEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/twirl.bmp", 7)
}

func bottomAbyssEffectSpec() EffectSpec {
	return songMusicEffectSpec("", 5)
}

func bottomSiegEffectSpec() EffectSpec {
	return songLightEffectSpec(4)
}
