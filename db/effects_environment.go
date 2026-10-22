package db

import (
	"image/color"
	"math"
	"time"
)

func sakuraEffectSpec() EffectSpec {
	return weatherLeafEffectSpec("data/sprite/이팩트/sakura01", color.RGBA{R: 255, G: 210, B: 225, A: 255})
}

func mapleEffectSpec() EffectSpec {
	return weatherLeafEffectSpec("data/sprite/이팩트/단풍", color.RGBA{R: 255, G: 170, B: 80, A: 255})
}

func cloudEffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.18, 35, -10, 20*time.Second)
}

func cloud2EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 20*time.Second)
}

func cloud3EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"}, color.RGBA{R: 120, G: 110, B: 100, A: 255}, 0.78, 45, -10, 34*time.Second)
}

func cloud5EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 225, G: 212, B: 194, A: 255}, 0.70, 50, 4, 8*time.Second)
}

func cloud6EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 28*time.Second)
}

func cloud7EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 51, G: 79, B: 161, A: 255}, 0.55, 35, 4, 20*time.Second)
}

func cloud8EffectSpec() EffectSpec {
	return weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 140, B: 51, A: 255}, 0.62, 35, 4, 20*time.Second)
}

func smokeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 10 * time.Second,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "굴뚝연기",
			SpriteRepeat:     true,
			Duration:         10 * time.Second,
			Duplicate:        10,
			DuplicateDelay:   time.Second,
			AlphaMax:         0.8,
			FadeOut:          true,
			BlendAdditive:    true,
			PosZ:             0,
			PosZEnd:          20,
			PosXEndRand:      3,
			PosXSmooth:       true,
			SizeStart:        effectTableSize(70),
			SizeEnd:          effectTableSize(300),
			SizeSmooth:       true,
			AngleStart:       -90,
			AngleEnd:         0,
			Rotate:           true,
			RotateWithCamera: true,
			Color:            color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}},
	}
}

func fireflyEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 3 * time.Second,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "data/sprite/이팩트/particle1",
			SpriteRepeat:   true,
			Duration:       3 * time.Second,
			Duplicate:      3,
			DuplicateDelay: 420 * time.Millisecond,
			PosXRand:       0.18,
			PosYRand:       0.18,
			PosZ:           0.25,
			PosZEnd:        1.0,
			PosZStartRand:  0.2,
			PosZEndRand:    0.25,
			SizeStart:      effectTableSize(70),
			SizeEnd:        effectTableSize(110),
			AlphaMax:       0.22,
			FadeIn:         true,
			FadeOut:        true,
			Color:          color.RGBA{R: 230, G: 250, B: 255, A: 255},
		}},
	}
}

func torchEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "torch_01",
			SpriteRepeat:   true,
			Duration:       600 * time.Millisecond,
			SpriteDelay:    100 * time.Millisecond,
			PosX:           0.1,
			PosZ:           0.8,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(100),
			AngleStart:     270,
			AngleEnd:       270,
			RotateToTarget: true,
			AlphaMax:       1,
			Color:          color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}},
	}
}

func torchRedEffectSpec() EffectSpec {
	return coloredTorchEffectSpec(color.RGBA{R: 255, G: 110, B: 75, A: 255})
}

func torchGreenEffectSpec() EffectSpec {
	return coloredTorchEffectSpec(color.RGBA{R: 90, G: 255, B: 135, A: 255})
}

func torchPurpleEffectSpec() EffectSpec {
	return coloredTorchEffectSpec(color.RGBA{R: 190, G: 120, B: 255, A: 255})
}

func bubbleEffectSpec() EffectSpec {
	return strEffectSpecRandom("bubble%d", "", 1, 4)
}

func dragonSmokeEffectSpec() EffectSpec {
	return EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "poisonhit",
			AttachedEntity: true,
		}},
	}
}

func mapPillarBlueEffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_blue", color.RGBA{R: 110, G: 175, B: 255, A: 255}, 2, 50.0/255.0)
}

func mapPillar2EffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_blue", color.RGBA{R: 110, G: 175, B: 255, A: 255}, 11, 70.0/255.0)
}

func mapPillar3EffectSpec() EffectSpec {
	return mapPillarEffectSpec("magic_green", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2, 50.0/255.0)
}

func mapPillar4EffectSpec() EffectSpec {
	return mapPillarEffectSpec("ring_red", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2, 50.0/255.0)
}

func mapGhostEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "particle2",
			SpriteRepeat:   true,
			Duration:       1400 * time.Millisecond,
			Duplicate:      7,
			DuplicateDelay: 190 * time.Millisecond,
			AlphaMax:       0.5,
			FadeIn:         true,
			FadeOut:        true,
			BlendAdditive:  true,
			PosZ:           0.2,
			PosZEnd:        2,
			PosXRand:       0.35,
			PosYRand:       0.35,
			SizeStart:      effectTableSize(70),
			SizeEnd:        effectTableSize(120),
			Color:          color.RGBA{R: 180, G: 230, B: 255, A: 255},
		}},
	}
}

func glow1EffectSpec() EffectSpec {
	return mapGlowEffectSpec(color.RGBA{R: 255, G: 245, B: 190, A: 255}, 1.2)
}

func glow2EffectSpec() EffectSpec {
	return mapGlowEffectSpec(color.RGBA{R: 150, G: 210, B: 255, A: 255}, 1.45)
}

func glow4EffectSpec() EffectSpec {
	return mapGlowEffectSpec(color.RGBA{R: 220, G: 175, B: 255, A: 255}, 1.8)
}

func bubbleDropEffectSpec() EffectSpec {
	return mapFallingParticleEffectSpec("effect/pok3.tga", color.RGBA{R: 95, G: 180, B: 255, A: 255}, 30)
}

func rainbowEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{
			mapRainbowBand(color.RGBA{R: 255, G: 90, B: 90, A: 255}, 0),
			mapRainbowBand(color.RGBA{R: 255, G: 210, B: 90, A: 255}, 0.18),
			mapRainbowBand(color.RGBA{R: 90, G: 255, B: 120, A: 255}, 0.36),
			mapRainbowBand(color.RGBA{R: 90, G: 170, B: 255, A: 255}, 0.54),
			mapRainbowBand(color.RGBA{R: 185, G: 110, B: 255, A: 255}, 0.72),
		},
	}
}

func banjjakiiEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "크리스마스",
			Duration:       time.Second,
			AttachedEntity: true,
		}},
	}
}

func bottomBlueEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		Components: []EffectComponent{
			bottomBlueCylinder(1.5, 2.0, 40.0/255.0, 0, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.58, 2.1, 32.0/255.0, 10, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 26.6, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 79.8, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
		},
	}
}

func bottomBlueCylinder(size, height, alpha, angleY float64, tint color.RGBA) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_down",
		Duration:         20 * time.Second,
		AlphaMax:         alpha,
		BottomSize:       size,
		TopSize:          size,
		Height:           height,
		AngleY:           angleY,
		RotateWithCamera: true,
		TotalCircleSides: 4,
		CircleSides:      4,
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Color:            tint,
	}
}

func coloredTorchEffectSpec(tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "torch_01",
			SpriteRepeat:   true,
			Duration:       600 * time.Millisecond,
			SpriteDelay:    100 * time.Millisecond,
			PosX:           0.1,
			PosZ:           0.8,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(100),
			AngleStart:     270,
			AngleEnd:       270,
			RotateToTarget: true,
			AlphaMax:       1,
			BlendAdditive:  true,
			Color:          tint,
		}},
	}
}

func mapGlowEffectSpec(tint color.RGBA, radius float64) EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "alpha_center",
			Duration:         1600 * time.Millisecond,
			AlphaMax:         0.36,
			Fade:             true,
			Rotate:           true,
			BottomSize:       radius,
			TopSize:          radius,
			Height:           0.06,
			TotalCircleSides: 32,
			CircleSides:      32,
			BlendAdditive:    true,
			Color:            tint,
		}},
	}
}

func mapPillarEffectSpec(texture string, tint color.RGBA, radiusStart, alphaMax float64) EffectSpec {
	const (
		duration   = 16 * time.Second
		worldScale = 0.2
	)
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponentFUNC,
			FuncName:      "MapPillar",
			TextureName:   texture,
			Duration:      duration,
			AlphaMax:      alphaMax,
			BottomSize:    radiusStart * worldScale,
			Height:        120 * worldScale,
			BlendMode:     2,
			BlendAdditive: true,
			Color:         tint,
		}},
	}
}

func mapFallingParticleEffectSpec(textureFile string, tint color.RGBA, count int) EffectSpec {
	return EffectSpec{
		Duration: 1400 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    textureFile,
			Duration:       1400 * time.Millisecond,
			Duplicate:      count,
			DuplicateDelay: 18 * time.Millisecond,
			AlphaMax:       0.78,
			FadeIn:         true,
			FadeOut:        true,
			BlendAdditive:  true,
			PosXRand:       4,
			PosYRand:       4,
			PosZStartRand:  1.5,
			PosZ:           12,
			PosZEnd:        0.2,
			PosXEndRand:    1,
			PosYEndRand:    1,
			PosZSmooth:     true,
			SizeStart:      effectTableSize(8),
			SizeEnd:        effectTableSize(5),
			Color:          tint,
		}},
	}
}

func mapRainbowBand(tint color.RGBA, zOffset float64) EffectComponent {
	return EffectComponent{
		Kind:          EffectComponent3D,
		TextureFile:   "effect/alpha_center.tga",
		Duration:      2200 * time.Millisecond,
		AlphaMax:      0.3,
		Fade:          true,
		BlendAdditive: true,
		PosZ:          7 + zOffset,
		SizeStartX:    effectTableSize(900),
		SizeStartY:    effectTableSize(45),
		SizeEndX:      effectTableSize(900),
		SizeEndY:      effectTableSize(45),
		AngleStart:    -18,
		Color:         tint,
	}
}

func weatherRainEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/alpha_center.tga",
			Duration:      900 * time.Millisecond,
			Duplicate:     85,
			AlphaMax:      0.5,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      28,
			PosYRand:      28,
			PosZ:          18,
			PosZEnd:       0,
			PosXEnd:       -3.5,
			PosYEnd:       2.0,
			PosZSmooth:    false,
			SizeStartX:    effectTableSize(2),
			SizeStartY:    effectTableSize(52),
			SizeEndX:      effectTableSize(2),
			SizeEndY:      effectTableSize(52),
			AngleStart:    -25,
			Color:         color.RGBA{R: 185, G: 215, B: 255, A: 255},
		}},
	}
}

func weatherSnowEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 8 * time.Second,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/pok3.tga",
			Duration:      8 * time.Second,
			Duplicate:     180,
			AlphaMax:      0.85,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      60,
			PosYRand:      60,
			PosZStartRand: 2,
			PosZ:          20,
			PosZEnd:       -4,
			PosZSmooth:    false,
			SizeStart:     effectTableSize(7),
			SizeEnd:       effectTableSize(7),
			Color:         color.RGBA{R: 245, G: 250, B: 255, A: 255},
		}},
	}
}

func weatherLeafEffectSpec(spriteFile string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 5200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       spriteFile,
			SpriteRepeat:     true,
			Duration:         5200 * time.Millisecond,
			Duplicate:        36,
			AlphaMax:         0.92,
			FadeIn:           true,
			FadeOut:          true,
			PosXRand:         30,
			PosYRand:         30,
			PosZ:             20,
			PosZEnd:          0,
			PosXEnd:          -4,
			PosYEnd:          2,
			PosXEndRand:      3,
			PosYEndRand:      3,
			PosXSmooth:       true,
			PosYSmooth:       true,
			SizeStart:        effectTableSize(80),
			SizeEnd:          effectTableSize(80),
			AngleRandMin:     0,
			AngleRandMax:     360,
			RotateWithCamera: true,
			Color:            tint,
		}},
	}
}

func weatherCloudEffectSpec(textures []string, tint color.RGBA, alpha, radius, z float64, duration time.Duration) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  textures,
			Duration:      duration,
			Duplicate:     28,
			AlphaMax:      alpha,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			PosXRand:      radius,
			PosYRand:      radius,
			PosXEnd:       2.4,
			PosYEnd:       0.8,
			PosZ:          z,
			SizeStartX:    effectTableSize(1500),
			SizeStartY:    effectTableSize(900),
			SizeEndX:      effectTableSize(1500),
			SizeEndY:      effectTableSize(900),
			AngleRandMin:  -10,
			AngleRandMax:  10,
			Color:         tint,
		}},
	}
}

func einbrochWeatherCloudEffectSpec() EffectSpec {
	const duration = 34 * time.Second
	size := 35 * math.Sqrt2 * 0.1
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  []string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"},
			Duration:      duration,
			Duplicate:     320,
			AlphaMax:      170.0 / 255.0,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			Overlay:       true,
			PosXRand:      15,
			PosYRand:      15,
			PosZRand:      0.5,
			PosXEnd:       1.5,
			PosYEnd:       1.5,
			PosZ:          -2,
			SizeStartX:    size,
			SizeStartY:    size,
			SizeEndX:      size,
			SizeEndY:      size,
			SizeRand:      10 * math.Sqrt2 * 0.1,
			Color:         color.RGBA{R: 252, G: 171, B: 143, A: 255},
		}},
	}
}
