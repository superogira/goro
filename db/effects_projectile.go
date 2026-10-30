package db

import "time"

func throwItemFlightEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("ThrowItem", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = texture
	return spec
}

func pressureDropEffectSpec(textureFile, delayedSFX string) EffectSpec {
	spec := EffectSpec{
		Duration: 1001 * time.Millisecond,
		SFX:      []string{"effect\\프레셔.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    textureFile,
				Duration:       500 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				Rotate:         true,
				AngleStart:     0,
				AngleEnd:       -611,
				PosZ:           20,
				PosZEnd:        5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponent3D,
				TextureFile:    textureFile,
				Duration:       500 * time.Millisecond,
				Delay:          501 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				FadeOut:        true,
				AngleStart:     -611,
				AngleEnd:       -611,
				PosZ:           5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
		},
	}
	if delayedSFX != "" {
		spec.SFX = append(spec.SFX, delayedSFX)
		spec.SFXDelays = []time.Duration{0, 500 * time.Millisecond}
	}
	return spec
}
