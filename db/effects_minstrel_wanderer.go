package db

import (
	"image/color"
	"time"
)

func dance1EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\수줍은하루의우울.wav")
}

func botReverbEffectSpec() EffectSpec {
	return melodyBillboardEffectSpec("effect/melody_b.bmp", "effect\\reverberation.wav", effectTableColor(1, 0.6, 0.6))
}

func rainParticleEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\rainstorm.wav")
}

func chemicalV2EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\안식의자장가.wav")
}

func botReverb2EffectSpec() EffectSpec {
	return melodyBillboardEffectSpec("effect/melody_a.bmp", "effect\\나락의노래.wav", effectTableColor(0.6, 0.6, 1))
}

func circlePower2EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\순환하는자연의소리.wav")
}

func sprPlant2EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\워그와함께춤을.wav")
}

func sprPlant3EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\마나의노래.wav")
}

func sprPlant4EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\새터데이나이트피버.wav")
}

func sprPlant5EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\레라드의이슬.wav")
}

func sprPlant6EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\멜로디오브싱크.wav")
}

func sprPlant7EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\비욘드오브워크라이.wav")
}

func sprPlant8EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\언리미티드허밍보이스.wav")
}

func heartAsuraEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\세이렌의목소리.wav")
}

// Musical ground effects are shared by the Bard and Dancer job families.
func melodyBillboardEffectSpec(texture, wav string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 100 * time.Millisecond,
		SFX:      []string{wav},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    texture,
			Color:          tint,
			Duration:       100 * time.Millisecond,
			AlphaMax:       0.6,
			AttachedEntity: true,
			SizeStart:      effectTableSize(50),
			SizeEnd:        effectTableSize(50),
			PosZ:           0.5,
			Repeat:         true,
		}},
	}
}

func secra2EffectSpec() EffectSpec {
	components := make([]EffectComponent, 0, 10)
	for i := 0; i < 10; i++ {
		sizeStart := 850 - float64(i)*50
		tint := effectTableColor(1, 0.55+float64(i)*0.05, 0.55+float64(i)*0.05)
		components = append(components, EffectComponent{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/priest_spell.bmp",
			Color:          tint,
			Duration:       1500 * time.Millisecond,
			AlphaMax:       0.3,
			BlendMode:      2,
			BlendAdditive:  true,
			FadeIn:         true,
			FadeOut:        true,
			AttachedEntity: true,
			PosZ:           7,
			SizeStart:      effectTableSize(sizeStart),
			SizeEnd:        effectTableSize(100),
			SizeSmooth:     true,
		})
	}
	return EffectSpec{
		Duration:   1500 * time.Millisecond,
		SFX:        []string{"effect\\ab_ancilla.wav"},
		Components: components,
	}
}
