package db

import "time"

func soulDrainEffectSpec() EffectSpec {
	spec := funcEffectSpec("SoulDrain", 200*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "particle1"
	return spec
}

func napalmVulcanEffectSpec() EffectSpec {
	spec := EffectSpec{Duration: 200 * EffectFrameDuration}
	// Hit2 only creates particles at stateCnt == 0. The original calls it at
	// these later ticks, so only the five Napalm Beat sounds are produced.
	for tick := 20; tick <= 60; tick += 10 {
		spec.SFX = append(spec.SFX, "effect\\EF_NapalmBeat.wav")
		spec.SFXDelays = append(spec.SFXDelays, time.Duration(tick)*EffectFrameDuration)
	}
	return spec
}

func gravitationEffectSpec() EffectSpec {
	spec := funcEffectSpec("Gravitation", 2000*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/stone.bmp"
	spec.Components[0].TextureFiles = []string{"effect/ice.tga"}
	spec.CameraShake = 200 * time.Millisecond
	return spec
}

func magicCrasherEffectSpec() EffectSpec {
	spec := funcEffectSpec("MagicCrasherBodyColor", 100*EffectFrameDuration, true)
	spec.CameraShake = 650 * time.Millisecond
	spec.CameraShakeDelay = 30 * EffectFrameDuration
	spec.SFX = []string{"effect\\매직 크래쉬.wav"}
	spec.SFXDelays = []time.Duration{25 * EffectFrameDuration}
	return spec
}

func ganbanteinEffectSpec() EffectSpec {
	return groundCastEffectSpec("ring_white", 300)
}

func magicPowerEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\마법력 증폭.wav")
}
