package db

import "time"

// EffectHitRing holds the Prim3DCylinder motion in cells per original-client tick.
type EffectHitRing struct {
	Speed, HeightSpeed, HeightAcceleration float64
	InitialDistance                        float64
	FadeOutFrame                           int
	RandomImpactOffset                     bool
}

// EffectHitParticles uses the original random speed range, in hundredths of
// a native unit per tick. One cell is five native units.
type EffectHitParticles struct {
	SpeedMin, SpeedRange int
}

func hitRingComponent(texture string, frames int, inner, outer, height, speed float64, fadeOutFrame int) EffectComponent {
	return EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitRing", TextureName: texture,
		Duration: time.Duration(frames) * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		AlphaMax: 254.0 / 255, BottomSize: inner / 5, TopSize: outer / 5,
		Height: height / 5, PosZ: 2, AngleX: 90, CircleSides: 10, AttachedEntity: true,
		HitRing: &EffectHitRing{Speed: speed / 5, FadeOutFrame: fadeOutFrame},
	}
}

// CRagEffect::Hit3/Hit4: growing cylinders followed by three-segment SPR trails.
func legacyHitEffectSpec(arrow bool) EffectSpec {
	inner, count, speedMin, speedRange := 1.5, 8, 80, 120
	heightAccel, sound := 0.2, "effect\\ef_hit3.wav"
	if arrow {
		inner, count, speedMin, speedRange = 0.5, 5, 60, 90
		heightAccel, sound = 0.15, "effect\\ef_hit4.wav"
	}
	ring := hitRingComponent("lens2", 15, inner, 4, 0, 0.7, 10)
	ring.HitRing.HeightSpeed = 0.25 / 5
	ring.HitRing.HeightAcceleration = heightAccel / 5
	components := []EffectComponent{}
	if !arrow {
		core := hitRingComponent("lens2", 15, 1.5, 1.5, 0, 0.7, 10)
		core.HitRing.HeightSpeed = 0.5 / 5
		core.HitRing.HeightAcceleration = 0.2 / 5
		components = append(components, core)
	}
	components = append(components, ring, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "HitParticles", SpriteFile: "particle1",
		Duration: 30 * EffectFrameDuration, FrameDelay: EffectFrameDuration,
		Duplicate: count, PosZ: 2, AttachedEntity: true,
		HitParticles: &EffectHitParticles{SpeedMin: speedMin, SpeedRange: speedRange},
	})
	return EffectSpec{Duration: 30 * EffectFrameDuration, SFX: []string{sound}, Components: components}
}

func bash3DEffectSpec(funcName, wav string, duration, delay time.Duration, duplicate int) EffectSpec {
	return EffectSpec{
		Duration: duration,
		SFX:      []string{wav},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       funcName,
				AttachedEntity: true,
			},
			bash3DCylinderComponent(4.5, delay, duplicate),
			bash3DCylinderComponent(7.2, delay, duplicate),
		},
	}
}

func bash3DCylinderComponent(topSize float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_center",
		Duration:         175 * time.Millisecond,
		Delay:            delay,
		Duplicate:        duplicate,
		AlphaMax:         0.6,
		Fade:             true,
		AngleX:           -90,
		AngleZRandom:     360,
		FixedPerspective: true,
		PosZ:             1.5,
		Height:           0,
		BottomSize:       0.01,
		TopSize:          topSize,
		Animation:        2,
		AttachedEntity:   true,
		TotalCircleSides: 30,
		CircleSides:      1,
	}
}

func bash3D6EffectSpec() EffectSpec {
	spec := bash3DEffectSpec("Bash3D6", "effect\\bash3d.wav", 500*time.Millisecond, 200*time.Millisecond, 5)
	tint := effectTableColor(0.3, 0.5, 1)
	for i := 1; i < len(spec.Components); i++ {
		spec.Components[i].Color = tint
	}
	return spec
}

// The BASH3D ray burst is shared by Lord Knight attacks and Gunslinger Dust.
func bashRayEffectSpec(sound string, delay time.Duration) EffectSpec {
	return EffectSpec{
		Duration: 51 * EffectFrameDuration,
		SFX:      []string{sound}, SFXDelays: []time.Duration{delay},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Sacrifice", TextureFile: "effect/alpha_center.tga",
		}},
	}
}

func hit1EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			TextureFile: "effect/pok3.tga",
			Duration:    300 * time.Millisecond,
			Duplicate:   4,
			AlphaMax:    0.8,
			FadeIn:      true,
			FadeOut:     true,
			Sparkling:   true,
			PosZ:        1,
			PosXEndRand: 2,
			PosYEndRand: 2,
			PosZEndRand: 2,
			SizeStart:   effectTableSize(10),
			SizeEnd:     effectTableSize(10),
			SizeRand:    effectTableSize(20),
			SizeSmooth:  true,
		}, hitRingComponent("ring_blue", 10, 5, 10, 3.5, 0.7, 5)},
	}
}

func hit3EffectSpec() EffectSpec {
	return legacyHitEffectSpec(false)
}

func hit4EffectSpec() EffectSpec {
	return legacyHitEffectSpec(true)
}

func hit5EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 400 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit5.wav"},
		Components: []EffectComponent{
			hitSlashComponent(EffectComponent3D, effectTableSize(15), effectTableSize(200), 90, 0, false),
			hitSlashComponent(EffectComponent3D, effectTableSize(15), effectTableSize(200), 180, 90, false),
		},
	}
}

func hit6EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 400 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit6.wav"},
		Components: []EffectComponent{
			hitSlashComponent(EffectComponent2D, effectTableSize(10), effectTableSize(150), 90, 0, true),
			hitSlashComponent(EffectComponent2D, effectTableSize(10), effectTableSize(150), 180, 90, true),
		},
	}
}

func fireHitEffectSpec() EffectSpec {
	return strEffectSpecRandomAttached("firehit%d", "effect\\ef_firehit.wav", 1, 3, true, false)
}

func fireSplashHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponent2D,
			TextureFile: "effect/firering.tga",
			Duration:    500 * time.Millisecond,
			FadeOut:     true,
			PosZ:        1,
			SizeStart:   10 * EffectPixelRatio,
			SizeEnd:     300 * EffectPixelRatio,
			AngleStart:  0,
			AngleEnd:    -360,
		}},
	}
}

func coldHitEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("_hit_fist3.wav", "_hit_fist4.wav")
}

func windHitEffectSpec() EffectSpec {
	return strEffectSpecRandomAttached("windhit%d", "_hit_fist%d.wav", 1, 3, true, false)
}

func poisonHitEffectSpec() EffectSpec {
	return EffectSpec{
		SFX: []string{"effect\\ef_poisonattack.wav"},
		Components: []EffectComponent{{
			Kind:       EffectComponentSPR,
			SpriteFile: "poisonhit",
		}},
	}
}

func earthHitEffectSpec() EffectSpec {
	return strEffectSpecAttached("earthhit", "", false)
}

func spearHitSoundEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("_hit_spear.wav")
}

func enemyHitNormal1EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("_enemy_hit_normal1.wav")
}

func magicalAtkHitEffectSpec() EffectSpec {
	return strEffectSpecAttached("magical", "", false)
}

func deathEffectSpec() EffectSpec {
	return strEffectSpecAttached("devil", "", false)
}

func fireHit2EffectSpec() EffectSpec {
	return strEffectSpecRandomAttached("firehit%d", "", 1, 3, true, false)
}

func hitSlashComponent(kind EffectComponentKind, sizeX, sizeEndY, angleStart, angleEnd float64, overlay bool) EffectComponent {
	return EffectComponent{
		Kind:        kind,
		TextureFile: "effect/lens2.tga",
		Duration:    400 * time.Millisecond,
		AlphaMax:    1,
		FadeOut:     true,
		Rotate:      true,
		PosZ:        1,
		SizeStartX:  sizeX,
		SizeEndX:    sizeX,
		SizeStartY:  effectTableSize(10),
		SizeEndY:    sizeEndY,
		AngleStart:  angleStart,
		AngleEnd:    angleEnd,
		Overlay:     overlay,
	}
}

func hitLineBurstEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SoulLinkerHitLines",
			TextureFile: "effect/white02.bmp", AttachedEntity: true,
		}},
	}
}
