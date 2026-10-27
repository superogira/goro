package db

import (
	"image/color"
	"time"
)

func suiExplosionEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\ef_hit2.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "sui_explosion",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func drainEffectSpec(tint color.RGBA, withBodyColor bool) EffectSpec {
	spec := EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/이팩트/particle1",
			SpriteRepeat:     true,
			Duration:         600 * time.Millisecond,
			Duplicate:        5,
			DuplicateDelay:   0,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			AlphaMax:         1,
			Color:            tint,
			SizeStart:        effectTableSize(150),
			SizeEnd:          effectTableSize(180),
			PosZ:             5,
			Arc:              3,
			Retreat:          3,
		}},
	}
	if withBodyColor {
		spec.Components = append(spec.Components,
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainOwnerBodyColor", AttachedEntity: true, Delay: 500 * time.Millisecond, Duration: 400 * time.Millisecond},
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainTargetBodyColor", AttachedEntity: true, Duration: 400 * time.Millisecond},
		)
	}
	return spec
}

func npcEarthquakeEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\earth_quake.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSPR,
				SpriteFile:     "어스퀘이크",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
				Duplicate:      3,
				DuplicateDelay: 35 * time.Millisecond,
			},
		},
	}
}

func dragonFearEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\dragonfear.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "dragon_h",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func evilLandEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:      EffectComponentFUNC,
				FuncName:  "FlatColorTile",
				Color:     color.RGBA{R: 160, G: 160, B: 160, A: 51},
				SizeStart: 1,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/curse.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      1,
				SizeEnd:        1,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				AttachedEntity: false,
			},
		},
	}
}

func darkSoulStrikeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 450 * time.Millisecond,
		SFX:      []string{"effect\\ef_soulstrike.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				TextureFile:     "effect/pok3.tga",
				Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
				Duration:        200 * time.Millisecond,
				Delay:           250 * time.Millisecond,
				DuplicateDelay:  150 * time.Millisecond,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				ToSrc:           true,
				PosZEnd:         1,
				PosZSmooth:      true,
				PosZStartRand:   5,
				PosZStartMiddle: 6,
				SizeStart:       effectTableSize(50),
				SizeEnd:         effectTableSize(50),
				AttachedEntity:  true,
			},
			{
				Kind:           EffectComponent3D,
				SpriteFile:     "data/sprite/이팩트/particle5",
				SpriteRepeat:   true,
				Duration:       250 * time.Millisecond,
				Duplicate:      5,
				DuplicateDelay: 20 * time.Millisecond,
				ToSrc:          true,
				RotateToTarget: true,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(500),
				PosZ:           3,
				Arc:            4,
				Retreat:        4,
			},
		},
	}
}

func darkJupitelHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pokjuk_d.bmp",
				Duration:       100 * time.Millisecond,
				SizeStart:      0,
				SizeEnd:        effectTableSize(25),
				BlendMode:      2,
				BlendAdditive:  true,
				RotateToTarget: true,
				FadeOut:        true,
				Overlay:        true,
				AttachedEntity: true,
			},
			{
				Kind: EffectComponent3D,
				TextureFiles: []string{
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_b.bmp",
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_c.bmp",
					"effect/twirl_soft.bmp",
				},
				FrameDelay:     10 * time.Millisecond,
				Duration:       300 * time.Millisecond,
				SizeStart:      effectTableSize(75),
				SizeEnd:        effectTableSize(75),
				BlendMode:      2,
				BlendAdditive:  true,
				Overlay:        true,
				AttachedEntity: true,
			},
		},
	}
}

func summonSlaveEffectSpec() EffectSpec {
	return EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "smoke",
			AttachedEntity: true,
		}},
	}
}

func gasPushEffectSpec() EffectSpec {
	return strEffectSpec("gaspush", "")
}

func mentalBreakEffectSpec() EffectSpec {
	return strEffectSpecAttached("mentalbreak", "", false)
}

func suicideEffectSpec() EffectSpec {
	return strEffectSpecAttached("suicide", "", false)
}

func comboAttack1EffectSpec() EffectSpec {
	return strEffectSpecAttached("yunta_1", "", false)
}

func comboAttack2EffectSpec() EffectSpec {
	return strEffectSpecAttached("yunta_2", "", false)
}

func comboAttack3EffectSpec() EffectSpec {
	return strEffectSpecAttached("yunta_3", "", false)
}

func comboAttack4EffectSpec() EffectSpec {
	return strEffectSpecAttached("yunta_4", "", false)
}

func comboAttack5EffectSpec() EffectSpec {
	return strEffectSpecAttached("yunta_5", "", false)
}

func guidedAttackEffectSpec() EffectSpec {
	return strEffectSpecAttached("homing", "", false)
}

func poisonAttack2EffectSpec() EffectSpec {
	return strEffectSpecAttached("poison", "", false)
}

func silenceAttackEffectSpec() EffectSpec {
	return strEffectSpecAttached("silence", "", false)
}

func stunAttackEffectSpec() EffectSpec {
	return strEffectSpecAttached("stun", "", false)
}

func petrifyAttackEffectSpec() EffectSpec {
	return strEffectSpecAttached("stonecurse", "", false)
}

func sleepAttackEffectSpec() EffectSpec {
	return strEffectSpecAttached("sleep", "", false)
}

func darkBreathEffectSpec() EffectSpec {
	return EffectSpec{Components: []EffectComponent{{Kind: EffectComponentSPR, SpriteFile: "darkbreath", SpriteHead: true, AttachedEntity: true}}}
}

func defenderEffectSpec() EffectSpec {
	return strEffectSpecAttached("deffender", "", false)
}

func keepingEffectSpec() EffectSpec {
	return strEffectSpecAttached("keeping", "", false)
}

func bloodDrainEffectSpec() EffectSpec {
	return drainEffectSpec(color.RGBA{R: 255, G: 102, B: 102, A: 255}, false)
}

func energyDrainEffectSpec() EffectSpec {
	return drainEffectSpec(color.RGBA{R: 102, G: 102, B: 255, A: 255}, true)
}

func darkGrandCrossEffectSpec() EffectSpec {
	return EffectSpec{}
}

func npcStopEffectSpec() EffectSpec {
	return sprEffectSpec("스톱", "", true, false)
}

func wideBleedingEffectSpec() EffectSpec {
	return strEffectSpecAttached("wideb", "effect\\wideb.wav", false)
}

func wideConfuseEffectSpec() EffectSpec {
	return strEffectSpecAttached("dfear", "effect\\dragonfear.wav", false)
}

func bottomRunnerEffectSpec() EffectSpec {
	return groundTextureEffectSpec("effect/hanmoon1.tga")
}

func bottomTransferEffectSpec() EffectSpec {
	return groundTextureEffectSpec("effect/hanmoon2.tga")
}

func guard3EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\kyrie_guard.wav")
}

func criticalWoundEffectSpec() EffectSpec {
	return strEffectSpecAttached("cwound", "", false)
}

func venomFogEffectSpec() EffectSpec {
	return strEffectSpecRandomAttached("bubble%d_1", "", 1, 4, true, false)
}

func dustStormEffectSpec() EffectSpec {
	return strEffectSpecAttached("dust", "", false)
}

func danceBladeAtkEffectSpec() EffectSpec {
	return strEffectSpecAttached("dancingblade", "", false)
}

func deathSummonEffectSpec() EffectSpec {
	return strEffectSpecAttached("devil", "", false)
}

func npcChillEffectSpec() EffectSpec {
	return strEffectSpecAttached("chill", "", false)
}
