package db

import (
	"image/color"
	"time"
)

func level99EffectSpec() EffectSpec {
	spec := funcEffectSpec("Level99Aura", 5*time.Minute, true)
	spec.Components[0].TextureFile = "effect/ring_blue.tga"
	return spec
}

func level99GroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundAura",
			TextureFile:    "effect/pikapika2.bmp",
			AttachedEntity: true,
			SizeStart:      effectTableSize(115),
			SizeEnd:        effectTableSize(130),
			PosZ:           0.05,
		}},
	}
}

func level99BubbleEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Level99Bubble",
			TextureFile:    "effect/whitelight.tga",
			AttachedEntity: true,
			Color:          color.RGBA{R: 80, G: 80, B: 255, A: 250},
			SizeStart:      effectTableSize(22.2),
			PosZ:           0.05,
		}},
	}
}
