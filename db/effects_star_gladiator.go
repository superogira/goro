package db

import (
	"image/color"
	"time"
)

func starWarmthEffectSpec() EffectSpec {
	spec := gumgangEffectSpec()
	spec.Components[0].Duplicate = 2
	return spec
}

func starWarmthParticlesEffectSpec() EffectSpec {
	spec := hatedEffectSpec()
	spec.Duration = 300 * EffectFrameDuration
	spec.SFX = []string{"effect\\T_따듯한마법.wav"}
	spec.SFXDelays = []time.Duration{10 * EffectFrameDuration}
	spec.Components[0].Color = color.RGBA{250, 100, 100, 255}
	return spec
}

func starComfortEffectSpec() EffectSpec {
	return EffectSpec{Duration: 400 * EffectFrameDuration,
		SFX: []string{"effect\\T_안락한마법.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration},
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "StarRings", TextureFile: "effect/ring_blue.tga", AttachedEntity: true}},
	}
}

func starBlackDevilEffectSpec() EffectSpec {
	return EffectSpec{Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "StarRings", TextureFile: "effect/ring_black.tga", AttachedEntity: true}},
	}
}

func starBlindEffectSpec() EffectSpec {
	return EffectSpec{Duration: 99999999 * EffectFrameDuration,
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "StarBlind", TextureFiles: []string{"effect/fullb.tga", "effect/white02.bmp"}, AttachedEntity: true}},
	}
}
