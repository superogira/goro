package db

import "time"

func bottomHummingEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/melody_a.bmp", 0)
}

func bottomForgetEffectSpec() EffectSpec {
	return songVerticalEffectSpec("effect/magic_green.tga", 2)
}

func bottomFortuneEffectSpec() EffectSpec {
	return songMusicEffectSpec("effect/kiss.bmp", 4)
}

func bottomServiceEffectSpec() EffectSpec {
	return songVerticalEffectSpec("effect/safeline.bmp", 4)
}

func talkScreamEffectSpec() EffectSpec {
	return funcEffectSpec("ScreamTalk", 500*time.Millisecond, true)
}

func winkEffectSpec() EffectSpec {
	return directedSpriteEffectSpec("wink")
}

func bottomUglyDanceEffectSpec() EffectSpec {
	return songVerticalEffectSpec("effect/ring_red.tga", 0)
}
