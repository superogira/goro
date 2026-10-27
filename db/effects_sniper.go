package db

import "time"

func trueSightEffectSpec() EffectSpec {
	spec := funcEffectSpec("TrueSight", 250*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\hunter_detecting.wav"}
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	return spec
}

func sharpShootingEffectSpec() EffectSpec {
	spec := funcEffectSpec("SharpShooting", 200*EffectFrameDuration, false)
	spec.SFX = []string{"effect\\샤프슈팅.wav"}
	spec.SFXDelays = []time.Duration{30 * EffectFrameDuration}
	spec.Components[0].TextureFile = "effect/alpha_center.tga"
	return spec
}

func windWalkEffectSpec() EffectSpec {
	spec := funcEffectSpec("WindWalk", 200*EffectFrameDuration, false)
	spec.SFX = []string{"effect\\윈드워크.wav"}
	spec.Components[0].TextureFile = "effect/cloud11.tga"
	return spec
}

func falconAssaultEffectSpec() EffectSpec {
	spec := funcEffectSpec("CameraQuake", 200*EffectFrameDuration, true)
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 30 * EffectFrameDuration
	spec.Components[0].Delay = spec.CameraShakeDelay
	spec.Components[0].Duration = spec.Duration - spec.CameraShakeDelay
	return spec
}

func sharpShootingCastEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 10 * time.Second,
		Components: []EffectComponent{{
			Kind:             EffectComponentFUNC,
			FuncName:         "CastRing",
			TextureName:      "ring_jadu",
			AlphaMax:         0.9,
			Fade:             true,
			Rotate:           true,
			BottomSize:       0.8,
			TopSize:          2.45,
			Height:           2.8,
			PosZ:             0.08,
			TotalCircleSides: 20,
			CircleSides:      20,
			AttachedEntity:   true,
		}},
	}
}
