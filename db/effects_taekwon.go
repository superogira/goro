package db

import "time"

func mildWindEffectSpec(texture string) EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "AsuraGlyphs", TextureFiles: []string{texture},
		AttachedEntity: true,
	}}}
}

func taekwonStormEffectSpec(kick bool) EffectSpec {
	spec := EffectSpec{Duration: 100 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonStorm",
		TextureFiles:   []string{"effect/storm2.tga", "effect/cloud11.tga"},
		AttachedEntity: true,
	}}}
	if kick {
		spec.SFX = []string{"effect\\T_회오리차기.wav"}
		spec.SFXDelays = []time.Duration{5 * EffectFrameDuration}
	}
	return spec
}

func taekwonStopEffectSpec() EffectSpec {
	return EffectSpec{Duration: 100 * EffectFrameDuration,
		SFX: []string{"effect\\T_효과음1.wav"}, Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "TaekwonStop", TextureFile: "effect/magic_blue.tga", AttachedEntity: true,
		}}}
}

func taekwonDashEffectSpec() EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonDash", TextureFile: "effect/line3.tga",
	}}}
}

func taekwonJumpEffectSpec() EffectSpec {
	return EffectSpec{Duration: 300 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonJump",
		TextureFiles: []string{"effect/whitelight.tga", "effect/peong1.tga", "effect/peong2.tga", "effect/peong3.tga"},
	}}}
}

func taekwonFootEffectSpec(texture string) EffectSpec {
	return EffectSpec{Duration: 340 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "TaekwonFoot", TextureFile: texture,
	}}}
}

func pressedBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 50 * EffectFrameDuration, SFX: []string{"effect\\T_내려찍기.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration}}
}

func spinedBody2EffectSpec() EffectSpec {
	return EffectSpec{Duration: 50 * EffectFrameDuration, SFX: []string{"effect\\T_돌려차기.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration}}
}

func kickedBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 100 * EffectFrameDuration, SFX: []string{"effect\\EF_hit2.wav", "effect\\EF_hit4.wav"}, SFXDelays: []time.Duration{17 * EffectFrameDuration, 25 * EffectFrameDuration}}
}

func runEffectSpec() EffectSpec {
	return EffectSpec{Duration: 9999 * EffectFrameDuration}
}

func stopRunEffectSpec() EffectSpec {
	return EffectSpec{Duration: 30 * EffectFrameDuration}
}

func quakeBody2EffectSpec() EffectSpec {
	return EffectSpec{Duration: 35 * EffectFrameDuration}
}

func stormKickEffectSpec() EffectSpec {
	return taekwonStormEffectSpec(true)
}

func stormKick3EffectSpec() EffectSpec {
	return taekwonStormEffectSpec(false)
}

func foot3EffectSpec() EffectSpec {
	return taekwonFootEffectSpec("effect/foot_l2.tga")
}

func foot4EffectSpec() EffectSpec {
	return taekwonFootEffectSpec("effect/foot_r2.tga")
}

func jumpBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 500 * EffectFrameDuration, SFX: []string{"effect\\T_회피2.wav"}, SFXDelays: []time.Duration{30 * EffectFrameDuration}}
}

func landBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 35 * EffectFrameDuration}
}

func jumpKickEffectSpec() EffectSpec {
	return EffectSpec{Duration: 100 * EffectFrameDuration, SFX: []string{"effect\\T_날라차기.wav"}, SFXDelays: []time.Duration{5 * EffectFrameDuration}}
}

func beginAsura1EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon1.tga")
}

func beginAsura2EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon2.tga")
}

func beginAsura3EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon3.tga")
}

func beginAsura4EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon4.tga")
}

func beginAsura5EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon7.tga")
}

func beginAsura6EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon5.tga")
}

func beginAsura7EffectSpec() EffectSpec {
	return mildWindEffectSpec("effect/hanmoon6.tga")
}
