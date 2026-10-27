package db

import "time"

func spiritSphereEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "SpiritSphere",
			TextureFile:    "effect/thunder_center.bmp",
			AttachedEntity: true,
			Duplicate:      5,
		}},
	}
}
