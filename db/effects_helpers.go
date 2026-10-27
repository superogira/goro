package db

import (
	"image/color"
	"strconv"
	"time"
)

func effectTableSize(value float64) float64 {
	return value * EffectPixelRatio
}

func effectTableColor(red, green, blue float64) color.RGBA {
	return color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255}
}

func strEffectSpec(file, wav string) EffectSpec {
	return strEffectSpecRandom(file, wav, 0, 0)
}

func strEffectSpecRandom(file, wav string, randMin, randMax int) EffectSpec {
	return strEffectSpecRandomAttached(file, wav, randMin, randMax, false, false)
}

func strEffectSpecAttached(file, wav string, head bool) EffectSpec {
	return strEffectSpecRandomAttached(file, wav, 0, 0, true, head)
}

func strEffectSpecAttachedMin(file, minFile, wav string, head bool) EffectSpec {
	spec := strEffectSpecAttached(file, wav, head)
	spec.Components[0].STRMinFile = minFile
	return spec
}

func strEffectSpecAttachedTexturePath(file, texturePath, wav string, head bool) EffectSpec {
	return strEffectSpecTexturePath(file, texturePath, wav, true, head)
}

func strEffectSpecTexturePath(file, texturePath, wav string, attached, head bool) EffectSpec {
	spec := strEffectSpecRandomAttached(file, wav, 0, 0, attached, head)
	spec.Components[0].TexturePath = texturePath
	return spec
}

func strEffectSpecRandomAttached(file, wav string, randMin, randMax int, attached, head bool) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        file,
			STRRandMin:     randMin,
			STRRandMax:     randMax,
			AttachedEntity: attached,
			SpriteHead:     head,
		}},
	}
	if wav != "" {
		spec.SFX = []string{wav}
		spec.SFXRandMin = randMin
		spec.SFXRandMax = randMax
	}
	return spec
}

func sprEffectSpec(file, wav string, attached, head bool) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     file,
			AttachedEntity: attached,
			SpriteHead:     head,
		}},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func sprDirectionEffectSpec(file, wav string) EffectSpec {
	spec := sprEffectSpec(file, wav, true, false)
	spec.Components[0].SpriteDirection = true
	return spec
}

func sprRepeatEffectSpec(file, wav string, attached bool) EffectSpec {
	spec := sprEffectSpec(file, wav, attached, false)
	spec.Components[0].Repeat = true
	return spec
}

func sprTexturePathEffectSpec(file, texturePath string, attached, repeat, head, renderBefore bool, yOffset float64) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     file,
			TexturePath:    texturePath,
			AttachedEntity: attached,
			Repeat:         repeat,
			SpriteRepeat:   repeat,
			SpriteHead:     head,
			SpriteYOffset:  yOffset,
			RenderBefore:   renderBefore,
		}},
	}
	if repeat {
		spec.Duration = 5 * time.Minute
	}
	return spec
}

func funcEffectSpec(name string, duration time.Duration, attached bool) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       name,
			AttachedEntity: attached,
		}},
	}
}

func robrCylinderBlendComponent(textureName string, tint color.RGBA, duration time.Duration, alphaMax float64, animation int, bottomSize, topSize, height float64, attached, rotate, fade bool, blendMode int) EffectComponent {
	component := robrCylinderComponent(textureName, tint, duration, alphaMax, animation, bottomSize, topSize, height, attached, rotate, fade, blendMode == 2)
	component.BlendMode = blendMode
	component.BlendAdditive = blendMode == 2
	return component
}

func soundOnlyEffectSpec(paths ...string) EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      paths,
	}
}

func directedSpriteEffectSpec(sprite string) EffectSpec {
	spec := funcEffectSpec("DirectedSprite", 100*EffectFrameDuration, false)
	c := &spec.Components[0]
	c.SpriteFile = sprite
	c.SpriteDelay = 6 * EffectFrameDuration
	c.WorldSizedSprite = true
	c.Overlay = true
	return spec
}

func twoDigitString(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func delayedSoundEffectSpec(paths []string, delays []time.Duration) EffectSpec {
	return EffectSpec{
		Duration:  500 * time.Millisecond,
		SFX:       paths,
		SFXDelays: delays,
	}
}

func delayedSoundEffectSpecDuration(duration time.Duration, paths []string, delays []time.Duration) EffectSpec {
	spec := delayedSoundEffectSpec(paths, delays)
	spec.Duration = duration
	return spec
}

func robrCylinderComponent(textureName string, tint color.RGBA, duration time.Duration, alphaMax float64, animation int, bottomSize, topSize, height float64, attached, rotate, fade, additive bool) EffectComponent {
	blendMode := 0
	if additive {
		blendMode = 2
	}
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            tint,
		TextureName:      textureName,
		Duration:         duration,
		AlphaMax:         alphaMax,
		Fade:             fade,
		Rotate:           rotate,
		Animation:        animation,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		AttachedEntity:   attached,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        blendMode,
		BlendAdditive:    additive,
	}
}

func tintedEffectComponent(component EffectComponent, tint color.RGBA) EffectComponent {
	component.Color = tint
	return component
}

func quadHornEffectComponent(textureFile string, duration time.Duration, height, offsetX, offsetY [2]float64, offsetZ float64, bottomSize, rotateY, rotateZ [2]float64, animation int, animationSpeed time.Duration, animationOut bool, blendMode int) EffectComponent {
	return quadHornEffectComponentFull(textureFile, duration, height, offsetX, offsetY, offsetZ, bottomSize, [2]float64{}, rotateY, rotateZ, animation, animationSpeed, animationOut, blendMode)
}

func quadHornEffectComponentFull(textureFile string, duration time.Duration, height, offsetX, offsetY [2]float64, offsetZ float64, bottomSize, rotateX, rotateY, rotateZ [2]float64, animation int, animationSpeed time.Duration, animationOut bool, blendMode int) EffectComponent {
	return EffectComponent{
		Kind:               EffectComponentQuadHorn,
		TextureFile:        textureFile,
		Duration:           duration,
		AttachedEntity:     false,
		Color:              color.RGBA{R: 255, G: 255, B: 255, A: 255},
		BlendMode:          blendMode,
		BlendAdditive:      blendMode == 2,
		Animation:          animation,
		QuadHornHeightMin:  height[0],
		QuadHornHeightMax:  height[1],
		QuadHornOffsetXMin: offsetX[0],
		QuadHornOffsetXMax: offsetX[1],
		QuadHornOffsetYMin: offsetY[0],
		QuadHornOffsetYMax: offsetY[1],
		QuadHornOffsetZ:    offsetZ,
		QuadHornBottomMin:  bottomSize[0],
		QuadHornBottomMax:  bottomSize[1],
		QuadHornRotateXMin: rotateX[0],
		QuadHornRotateXMax: rotateX[1],
		QuadHornRotateYMin: rotateY[0],
		QuadHornRotateYMax: rotateY[1],
		QuadHornRotateZMin: rotateZ[0],
		QuadHornRotateZMax: rotateZ[1],
		QuadHornAnimSpeed:  animationSpeed,
		QuadHornAnimOut:    animationOut,
	}
}
