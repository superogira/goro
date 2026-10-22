package db

import (
	"image/color"
	"time"
)

func pokJukEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 4 * time.Second,
		Components: []EffectComponent{
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				AlphaMax:      0.9,
				FadeIn:        true,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          0,
				PosZEnd:       8,
				PosXEndRand:   1.5,
				PosYEndRand:   1.5,
				PosZSmooth:    true,
				SizeStart:     effectTableSize(8),
				SizeEnd:       effectTableSize(12),
				Color:         color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				Delay:         850 * time.Millisecond,
				Duplicate:     45,
				AlphaMax:      0.9,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   4.5,
				PosYEndRand:   4.5,
				PosZEndRand:   4.5,
				SizeStart:     effectTableSize(14),
				SizeEnd:       effectTableSize(4),
				SizeRand:      effectTableSize(4),
				Color:         color.RGBA{R: 120, G: 180, B: 255, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				Delay:         900 * time.Millisecond,
				Duplicate:     45,
				AlphaMax:      0.85,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   4.0,
				PosYEndRand:   4.0,
				PosZEndRand:   4.0,
				SizeStart:     effectTableSize(12),
				SizeEnd:       effectTableSize(3),
				SizeRand:      effectTableSize(3),
				Color:         color.RGBA{R: 255, G: 120, B: 160, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      750 * time.Millisecond,
				Delay:         950 * time.Millisecond,
				Duplicate:     30,
				AlphaMax:      0.75,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   3.2,
				PosYEndRand:   3.2,
				PosZEndRand:   3.2,
				SizeStart:     effectTableSize(10),
				SizeEnd:       effectTableSize(2),
				SizeRand:      effectTableSize(2),
				Color:         color.RGBA{R: 255, G: 255, B: 130, A: 255},
			},
		},
	}
}

func pongEffectSpec() EffectSpec {
	return strEffectSpecRandom("pong%d", "", 1, 3)
}

func colorPaperEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\wedding.wav")
}

func mochiEffectSpec() EffectSpec {
	return strEffectSpecAttached("찹쌀떡", "", false)
}

func ramadanEffectSpec() EffectSpec {
	return strEffectSpecAttached("ramadan", "", false)
}

func itemPokJukEffectSpec() EffectSpec {
	return strEffectSpecAttached("itempokjuk", "effect\\itempokjuk.wav", false)
}

func valentine05EffectSpec() EffectSpec {
	return sprEffectSpec("05vallentine", "", true, false)
}

func firecracker2EffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("폭죽_러브")
}

func firecracker3EffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("폭죽_화이트데이")
}

func firecracker4EffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("폭죽_발렌타인")
}

func firecracker5EffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("폭죽_생일")
}

func firecracker6EffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("폭죽_크리스마스")
}

func flowerLeafEffectSpec() EffectSpec {
	return strEffectSpecAttached("flower_leaf", "", false)
}

func item315EffectSpec() EffectSpec {
	return strEffectSpecAttached("mobile_ef02", "", false)
}

func item316EffectSpec() EffectSpec {
	return strEffectSpecAttached("mobile_ef01", "", false)
}

func item317EffectSpec() EffectSpec {
	return strEffectSpecAttached("mobile_ef03", "", false)
}

func firecracker7EffectSpec() EffectSpec {
	return strEffectSpec("pokjuk_jap", "")
}

func rwc2011EffectSpec() EffectSpec {
	return strEffectSpecAttached("rwc2011", "", false)
}

func rwc2011TwoEffectSpec() EffectSpec {
	return strEffectSpecAttached("rwc2011_2", "", false)
}

func bakuretsuHadouEffectSpec() EffectSpec {
	return sprTexturePathEffectSpec("bakuretsu_hadou/bakuretsu_hadou", "bakuretsu_hadou/", true, true, true, false, -50)
}

func digitalSpaceEffectSpec() EffectSpec {
	return sprTexturePathEffectSpec("digital_space/digital_space", "digital_space/", true, true, false, true, 0)
}

func christmasCarolEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("angelus", "jong_mini", "effect\\wewish.wav", false)
}

func hapgyeokEffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("합격_")
}

func firecrackerBannerEffectSpec(sprite string) EffectSpec {
	return EffectSpec{
		SFX: []string{"effect\\itempokjuk.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSPR,
				SpriteFile:     sprite,
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentSTR,
				STRFile:        "itempokjuk",
				AttachedEntity: true,
			},
		},
	}
}
