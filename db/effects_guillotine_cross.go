package db

import (
	"image/color"
	"strconv"
	"time"
)

func rollingCutterCounterEffectSpec(index int) EffectSpec {
	texture := "effect/회전카운터" + strconv.Itoa(index) + ".tga"
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{
			rollingCutterCounterComponent(texture, 1.0, 1.0, 1.0, 1.0, 200, 1),
			rollingCutterCounterComponent(texture, 0.2, 0.7, 0.7, 1.0, 220, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.5, 0.5, 1.0, 240, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.3, 0.3, 1.0, 260, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.1, 0.1, 1.0, 280, 2),
		},
	}
}

func rollingCutterCounterComponent(texture string, alphaMax, red, green, blue, sizeStart float64, blendMode int) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    texture,
		Color:          color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255},
		Duration:       time.Second,
		AlphaMax:       alphaMax,
		FadeIn:         true,
		FadeOut:        true,
		PosZ:           4,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(20),
		SizeSmooth:     true,
		BlendMode:      blendMode,
		BlendAdditive:  blendMode == 2,
		AttachedEntity: true,
	}
}

func rolling1EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(1)
}

func rolling2EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(2)
}

func rolling3EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(3)
}

func rolling4EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(4)
}

func rolling5EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(5)
}

func rolling6EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(6)
}

func rolling7EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(7)
}

func rolling8EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(8)
}

func rolling9EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(9)
}

func rolling10EffectSpec() EffectSpec {
	return rollingCutterCounterEffectSpec(10)
}

func castSpin2EffectSpec() EffectSpec {
	return funcEffectSpec("CastSpin2", 500*time.Millisecond, true)
}

func gcDarkCrowEffectSpec() EffectSpec {
	return strEffectSpecAttached("gc_darkcrow", "", false)
}
