package db

import (
	"image/color"
	"time"
)

func judexEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1 * time.Second,
		SFX:      []string{"effect\\ab_judex.wav"},
		Components: []EffectComponent{
			judexCylinder(0.4, 0.5, 3.5),
			judexCylinder(0.45, 0.75, 2.5),
			judexCylinder(0.5, 1, 1.5),
		},
	}
}

func judexCylinder(bottom, top, height float64) EffectComponent {
	return EffectComponent{
		Kind:        EffectComponentCylinder,
		TextureName: "ring_white",
		Duration:    1 * time.Second,
		BottomSize:  bottom,
		TopSize:     top,
		Height:      height,
		Rotate:      true,
	}
}

func glassWall3EffectSpec() EffectSpec {
	tint := color.RGBA{R: 153, G: 255, B: 153, A: 255}
	portalWave := EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "magic_green",
		Color:            tint,
		Duration:         500 * time.Millisecond,
		AlphaMax:         0.4,
		Animation:        4,
		BottomSize:       2.4,
		TopSize:          3.9,
		Height:           0.1,
		PosZ:             0.1,
		FadeOut:          true,
		Rotate:           true,
		Duplicate:        150,
		DuplicateDelay:   200 * time.Millisecond,
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
	}
	cap := robrCylinderBlendComponent("alpha1", tint, 30*time.Second, 0.5, 0, 1, 1, 1, true, true, true, 2)
	cap.PosZ = 2
	cap.TotalCircleSides = 20
	cap.CircleSides = 10
	return EffectSpec{
		Components: []EffectComponent{
			portalWave,
			robrCylinderBlendComponent("magic_green", tint, 30*time.Second, 0.4, 0, 0.6, 0.6, 7, true, true, true, 2),
			robrCylinderBlendComponent("magic_green", tint, 30*time.Second, 0.4, 0, 0.8, 0.8, 6, true, true, true, 2),
			cap,
		},
	}
}

func glassWall4EffectSpec() EffectSpec {
	tint := effectTableColor(0.0001, 1, 0.0001)
	return EffectSpec{
		Duration: 30 * time.Second,
		SFX:      []string{"effect\\ef_readyportal.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/ef_epitree.tga",
				Color:          tint,
				Duration:       30 * time.Second,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				AttachedEntity: true,
				PosZ:           7,
				SizeStart:      effectTableSize(400),
				SizeEnd:        effectTableSize(400),
			},
			epiclesisTreePulseComponent(380, 420, 0),
			epiclesisTreePulseComponent(420, 380, time.Second),
		},
	}
}

func epiclesisTreePulseComponent(sizeStart, sizeEnd float64, delay time.Duration) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    "effect/ef_epitree.tga",
		Color:          effectTableColor(0.0001, 1, 0.0001),
		Duration:       990 * time.Millisecond,
		Delay:          delay,
		Duplicate:      15,
		DuplicateDelay: 2 * time.Second,
		AlphaMax:       0.6,
		BlendMode:      2,
		BlendAdditive:  true,
		AttachedEntity: true,
		PosZ:           7,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(sizeEnd),
	}
}

func forestLight5EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\ab_renovation.wav")
}

func adoramusEffectSpec() EffectSpec {
	return strEffectSpecAttached("ado", "effect\\ab_adoramus.wav", false)
}

func laulamusEffectSpec() EffectSpec {
	return strEffectSpecAttached("laulamus", "", false)
}

func lauagnusEffectSpec() EffectSpec {
	return strEffectSpecAttached("lauagnus", "", false)
}

func abOffertoriumEffectSpec() EffectSpec {
	return strEffectSpecAttached("ab_offertorium", "effect\\ab_offertorium.wav", false)
}

func offertoriumRingEffectSpec() EffectSpec {
	return strEffectSpecAttached("ab_offertorium_ring", "", false)
}
