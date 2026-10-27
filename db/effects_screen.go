package db

import "time"

func screenQuakeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    200 * time.Millisecond,
		CameraShake: 200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Duration:       200 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func chemical2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         500 * time.Millisecond,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 132 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			AttachedEntity: true,
		}},
	}
}

func whitePulseEffectSpec() EffectSpec {
	return EffectSpec{Duration: 500 * time.Millisecond}
}

func quakeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    650 * time.Millisecond,
		CameraShake: 650 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Duration:       650 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}
