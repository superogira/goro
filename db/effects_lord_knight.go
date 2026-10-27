package db

import "time"

func auraBladeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 60 * EffectFrameDuration,
		SFX:      []string{"effect\\오라 블레이드.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "BlueCasting",
			TextureFile: "effect/ring_white.tga", TextureFiles: []string{"effect/ring_yellow.tga"},
			AttachedEntity: true,
		}},
	}
}

func magnum2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  100 * EffectFrameDuration,
		SFX:       []string{"permeter_attack.wav", "effect\\ef_magnumbreak.wav"},
		SFXDelays: []time.Duration{20 * EffectFrameDuration, 30 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SpiralPierce", TextureFile: "effect/ring_yellow.tga",
			AttachedEntity: true,
		}},
	}
}

func redBodyEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 99999*EffectFrameDuration, true)
}

func pierceBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 200 * EffectFrameDuration, SFX: []string{"effect\\ef_beginspell.wav"}, Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "EffectBodyColor", AttachedEntity: true}}}
}

func lkConcentrationEffectSpec() EffectSpec {
	return strEffectSpecAttached("twohand", "", true)
}

func bash3D3EffectSpec() EffectSpec {
	return bashRayEffectSpec("effect\\헤드 크러쉬.wav", 0)
}

func bash3D4EffectSpec() EffectSpec {
	return bashRayEffectSpec("effect\\비트 조인트.wav", 22*EffectFrameDuration)
}
