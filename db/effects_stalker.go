package db

import "time"

func preserveEffectSpec() EffectSpec {
	spec := guardEffectSpec()
	spec.Duration = 200 * EffectFrameDuration
	spec.SFX = []string{"effect\\black_maximize_power_sword_bic.wav"}
	spec.Components[0].TextureFile = "effect/a01.bmp"
	spec.Components[0].TextureFiles = []string{"effect/a01.bmp"}
	return spec
}

func fullStripEffectSpec() EffectSpec {
	spec := funcEffectSpec("FullStrip", 300*EffectFrameDuration, false)
	spec.Components[0].TextureFiles = []string{"effect/shield.bmp", "effect/sword.bmp"}
	for _, tick := range []int{1, 11, 21, 31} {
		spec.SFX = append(spec.SFX, "effect\\복호격.wav")
		spec.SFXDelays = append(spec.SFXDelays, time.Duration(tick)*EffectFrameDuration)
	}
	return spec
}

func chaseWalkSmokeEffectSpec() EffectSpec {
	return EffectSpec{Components: []EffectComponent{{Kind: EffectComponentSPR, SpriteFile: "smoke", SpriteDelay: 5 * EffectFrameDuration / 6, WorldSizedSprite: true}}}
}

func rejectSwordEffectSpec() EffectSpec {
	return strEffectSpecAttached("sword", "effect\\kyrie_guard.wav", false)
}
