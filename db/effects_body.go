package db

import (
	"image/color"
	"time"
)

func redLightBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 99999999 * EffectFrameDuration}
}

func enhanceEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2000 * time.Millisecond,
		Components: []EffectComponent{
			robrCylinderComponent("alpha_down", color.RGBA{}, 2000*time.Millisecond, 0.2, 1, 0.95, 0.95, 10, true, false, true, true),
			incAgilityParticleComponent(0.8, 500*time.Millisecond, 7),
			incAgilityParticleComponent(0.8, 400*time.Millisecond, 3),
		},
	}
}

func makeBlurEffectSpec() EffectSpec {
	return funcEffectSpec("MakeBlur", 2*time.Second, false)
}

func lightBladeEffectSpec() EffectSpec {
	return funcEffectSpec("LightBlade", 99999999*EffectFrameDuration, true)
}

func pinkBodyEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 99999999*EffectFrameDuration, true)
}

func chemicalBodyEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 120*EffectFrameDuration, true)
}

func spinedBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 50 * EffectFrameDuration}
}

func babyBodyBackEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("BabyBodyBack", "", 15)
}

func babyBody2EffectSpec() EffectSpec {
	return bodySizeEffectSpec("EffectSmall", 5*time.Minute, 2.5)
}

func giantBodyEffectSpec() EffectSpec {
	return bodySizeEffectSpec("EffectBigTransition", 300*time.Millisecond, 7.5)
}

func giantBody2EffectSpec() EffectSpec {
	return bodySizeEffectSpec("EffectBig", 5*time.Minute, 7.5)
}

func quakeBodyEffectSpec() EffectSpec {
	return EffectSpec{Duration: 14 * EffectFrameDuration}
}

func castSpinEffectSpec() EffectSpec {
	return EffectSpec{Duration: 100 * EffectFrameDuration, SFX: []string{"effect\\EF_BeginSpell.wav"}}
}

func allRayProtectEffectSpec() EffectSpec {
	return strEffectSpecAttached("guardian", "", false)
}

func invincibleOff2EffectSpec() EffectSpec {
	return strEffectSpecAttached("invincibleoff2", "", false)
}

func allFullThrottleEffectSpec() EffectSpec {
	return strEffectSpecAttached("all_full_throttle", "effect\\all_full_throttle.wav", false)
}

func bodyColorEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 300*time.Millisecond, true)
}

func bodySizeEffectSpec(name string, duration time.Duration, targetSize float64) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       name,
			Duration:       duration,
			SizeEnd:        targetSize,
			AttachedEntity: true,
		}},
	}
}

func transBlueBodyEffectSpec() EffectSpec {
	return funcEffectSpec("TransBlueBody", 200*EffectFrameDuration, true)
}

func bodyTransformEffectSpec(name, sound string, frames int) EffectSpec {
	spec := funcEffectSpec(name, time.Duration(frames)*EffectFrameDuration, true)
	if sound != "" {
		spec.SFX = []string{sound}
	}
	return spec
}

func taeReadyEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("TaeReady", "", 60)
}
