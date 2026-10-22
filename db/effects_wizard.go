package db

import "time"

func iceWallEffectSpec() EffectSpec {
	spec := funcEffectSpec("WizardSpikes", 9999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/ice.tga"
	spec.SFX = []string{"effect\\wizard_icewall.wav"}
	return spec
}

func earthSpikeEffectSpec() EffectSpec {
	spec := funcEffectSpec("WizardSpikes", 240*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/stone.bmp"
	spec.SFX = []string{"effect\\wizard_earthspike.wav"}
	spec.CameraShake = 200 * time.Millisecond
	return spec
}

func heavenDriveEffectSpec() EffectSpec {
	spec := earthSpikeEffectSpec()
	spec.Duration = 300 * EffectFrameDuration
	spec.Components[0].AttachedEntity = false
	return spec
}

func stormGustEffectSpec() EffectSpec {
	spec := strEffectSpecAttachedMin("stormgust", "storm_min", "effect\\wizard_stormgust.wav", false)
	spikes := funcEffectSpec("WizardSpikes", 215*EffectFrameDuration, true).Components[0]
	spikes.Duration = 215 * EffectFrameDuration
	spikes.TextureFile = "effect/ice.tga"
	spec.Components = append(spec.Components, spikes)
	return spec
}

func sightTrasherEffectSpec() EffectSpec {
	spec := funcEffectSpec("SightTrasher", 85*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "sight"
	spec.Components[0].TextureFile = "effect/ring_yellow.tga"
	shadow := spec.Components[0]
	shadow.SpriteFile = "data/sprite/shadow"
	shadow.TextureFile = ""
	shadow.ShadowTexture = true
	spec.Components = append(spec.Components, shadow)
	spec.SFX = []string{"effect\\wizard_sightrasher.wav"}
	return spec
}

func sightBlasterEffectSpec() EffectSpec {
	spec := mageSightEffectSpec()
	spec.Components = spec.Components[:1]
	spec.SFX = nil
	return spec
}

func jupitelThunderEffectSpec() EffectSpec {
	spec := funcEffectSpec("Jupitel", 20*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\hunter_shockwavetrap.wav"}
	spec.Components[0].TextureFile = "effect/thunder_center.bmp"
	spec.Components[0].TextureFiles = []string{
		"effect/thunder_ball_a.bmp", "effect/thunder_ball_b.bmp", "effect/thunder_ball_c.bmp",
		"effect/thunder_ball_d.bmp", "effect/thunder_ball_e.bmp", "effect/thunder_ball_f.bmp",
	}
	return spec
}

func jupitelHitEffectSpec() EffectSpec {
	spec := funcEffectSpec("Jupitel", 250*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/thunder_pang.bmp"
	spec.Components[0].TextureFiles = []string{
		"effect/thunder_plazma_blast_a.bmp", "effect/thunder_plazma_blast_b.bmp",
		"effect/thunder_ball_d.bmp", "effect/thunder_ball_e.bmp", "effect/thunder_ball_f.bmp",
	}
	return spec
}

func waterBall2EffectSpec() EffectSpec {
	spec := funcEffectSpec("WaterBall", 40*EffectFrameDuration, true)
	spec.Components[0].SpriteFile = "waterball"
	spec.SFX = []string{"effect\\wizard_waterball_chulung.wav"}
	return spec
}

func firePillarEffectSpec() EffectSpec {
	spec := strEffectSpec("firepillar", "effect\\wizard_fire_pillar_a.wav")
	spec.SFXDelays = []time.Duration{75 * EffectFrameDuration}
	return spec
}

func firePillarOnEffectSpec() EffectSpec {
	spec := funcEffectSpec("FirePillar", 8000*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/magic_red.tga"
	return spec
}

func waterBallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\wizard_waterball_chulung.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponent3D,
			TextureFiles: []string{
				"effect/water_out_a.bmp",
				"effect/water_out_b.bmp",
				"effect/water_out_c.bmp",
			},
			FrameDelay:       10 * time.Millisecond,
			Duration:         500 * time.Millisecond,
			FadeOut:          true,
			PosXRand:         1.5,
			PosZRand:         1.5,
			PosY:             0,
			PosYEnd:          3,
			PosYSmooth:       true,
			SizeStart:        effectTableSize(30.5),
			SizeEnd:          effectTableSize(30.5),
			RotateWithCamera: true,
			BlendMode:        2,
			BlendAdditive:    true,
			AttachedEntity:   true,
		}},
	}
}

func lordVermilionEffectSpec() EffectSpec {
	return strEffectSpecAttached("lord", "effect\\wizard_fire_ivy.wav", false)
}

func meteorStormEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake:      650 * time.Millisecond,
		CameraShakeDelay: 40 * EffectFrameDuration,
		SFX:              []string{"effect\\wizard_meteor.wav"},
		SFXRandMin:       1,
		SFXRandMax:       4,
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "meteor%d",
			STRRandMin:     1,
			STRRandMax:     4,
			AttachedEntity: true,
		}},
	}
}

func quagmireEffectSpec() EffectSpec {
	return strEffectSpec("quagmire", "effect\\wizard_quagmire.wav")
}

func firePillarBombEffectSpec() EffectSpec {
	return strEffectSpec("firepillarbomb", "effect\\wizard_fire_pillar_b.wav")
}

func stormMinEffectSpec() EffectSpec {
	return strEffectSpecAttached("storm_min", "effect\\wizard_stormgust.wav", false)
}
