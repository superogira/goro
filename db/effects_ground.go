package db

import "time"

func groundWallsEffectSpec(texture string, frames int) EffectSpec {
	spec := funcEffectSpec("PriestGround", time.Duration(frames)*EffectFrameDuration, true)
	spec.Components[0].TextureFile = texture
	return spec
}

func groundTextureEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundTexture",
			TextureFile:    texture,
			Duration:       1500 * time.Millisecond,
			SizeStart:      1,
			SizeEnd:        1,
			PosZ:           0.05,
			BlendAdditive:  true,
			AttachedEntity: false,
		}},
	}
}

func propertyGroundEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("PropertyGround", 29999*EffectFrameDuration, false)
	spec.Components[0].TextureFile = "effect/" + texture + ".tga"
	return spec
}
