package db

import "time"

func throwItemFlightEffectSpec(texture string) EffectSpec {
	spec := funcEffectSpec("ThrowItem", 200*EffectFrameDuration, false)
	spec.Components[0].TextureFile = texture
	return spec
}

func throwItemEffectSpec(texture string, size float64) EffectSpec {
	return throwItemEffectSpecFull(texture, size, 300*time.Millisecond, 0.5)
}

func throwItemEffectSpecFull(texture string, size float64, duration time.Duration, rotations float64) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      texture,
			Duration:         duration,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Rotate:           true,
			AngleStart:       180,
			AngleEnd:         180 + 360*rotations,
			PosZ:             1,
			SizeStart:        effectTableSize(size),
			SizeEnd:          effectTableSize(size),
			AttachedEntity:   true,
		}},
	}
}

func throwItemSoundEffectSpec(texture string, size float64, wav string) EffectSpec {
	spec := throwItemEffectSpecFull(texture, size, 200*time.Millisecond, 1)
	spec.SFX = []string{wav}
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
