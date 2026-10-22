package db

import "time"

func bashHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 50 * EffectFrameDuration,
		SFX:      []string{"effect\\ef_hit2.wav"},
		Components: []EffectComponent{{
			Kind:         EffectComponentFUNC,
			FuncName:     "BashHit",
			TextureFile:  "effect/lens1.tga",
			TextureFiles: []string{"effect/lens2.tga"},
		}},
	}
}

func endureEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 80 * EffectFrameDuration,
		SFX:      []string{"effect\\ef_endure.wav"},
		Components: []EffectComponent{{
			Kind:         EffectComponentFUNC,
			FuncName:     "Endure",
			TextureFile:  "effect/endure.tga",
			TextureFiles: []string{"effect/alpha_down.tga"},
		}},
	}
}

func bashBeginEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 40 * EffectFrameDuration,
		SFX:      []string{"effect\\ef_bash.wav"},
		Components: []EffectComponent{{
			Kind:         EffectComponentFUNC,
			FuncName:     "Bash",
			TextureFile:  "effect/alpha_down.tga",
			TextureFiles: []string{"effect/alpha_center.tga"},
		}},
	}
}

func provokeEffectSpec() EffectSpec {
	return EffectSpec{
		SFX:       []string{"effect\\swordman_provoke.wav", "effect\\swordman_provoke.wav"},
		SFXDelays: []time.Duration{5 * EffectFrameDuration, 30 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind:    EffectComponentSTR,
			STRFile: "provoke",
		}},
	}
}

func magnumBreakEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 30 * EffectFrameDuration,
		SFX:      []string{"effect\\ef_magnumbreak.wav"},
		Components: []EffectComponent{{
			Kind:         EffectComponentFUNC,
			FuncName:     "MagnumBreak",
			TextureFile:  "effect/ring_yellow.tga",
			TextureFiles: []string{"effect/대폭발.tga"},
		}},
	}
}

func quakeMagnumEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    50 * time.Millisecond,
		CameraShake: 50 * time.Millisecond,
	}
}
