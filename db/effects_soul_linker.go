package db

import (
	"image/color"
	"time"
)

func estinEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  200 * EffectFrameDuration,
		SFX:       []string{"effect\\t_에너지방출.wav"},
		SFXDelays: []time.Duration{10 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Estin",
			TextureFile: "effect/270바람.tga", Color: color.RGBA{80, 80, 255, 255},
		}},
	}
}

func estunEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 250 * EffectFrameDuration,
		SFX:      []string{"effect\\t_에너지방출.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Estun",
			TextureFiles: []string{"effect/thunder_ball_0002.bmp", "effect/thunder_ball_0003.bmp"},
			Color:        color.RGBA{150, 150, 250, 255},
		}},
	}
}

func esmaEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		SFX: []string{
			"effect\\t_날라차기.wav", "effect\\t_날라차기.wav", "effect\\t_날라차기.wav",
			"effect\\t_날라차기.wav", "effect\\t_날라차기.wav",
		},
		SFXDelays: []time.Duration{5 * EffectFrameDuration, 20 * EffectFrameDuration, 35 * EffectFrameDuration, 50 * EffectFrameDuration, 65 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "Esma",
			TextureFile: "effect/270바람.tga", Color: color.RGBA{80, 80, 255, 255},
		}},
	}
}

func esmaReadyEffectSpec() EffectSpec {
	return hatedEffectSpec()
}

func esmaHitTextureEffectSpec() EffectSpec {
	return EffectSpec{Duration: 200 * EffectFrameDuration, Components: []EffectComponent{{
		Kind: EffectComponentFUNC, FuncName: "EsmaHitTexture",
		TextureFile: "effect/freeze_a.bmp", AttachedEntity: true,
	}}}
}

func soulLinkEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:  500 * EffectFrameDuration,
		SFX:       []string{"effect\\t_캐스팅.wav", "effect\\t_영혼.wav"},
		SFXDelays: []time.Duration{0, 145 * EffectFrameDuration},
		Components: []EffectComponent{{
			Kind: EffectComponentFUNC, FuncName: "SoulLink",
			TextureFile:    "effect/whitelight.tga",
			TextureFiles:   []string{"effect/soul_s.tga", "effect/soul_o.tga", "effect/soul_u.tga", "effect/soul_l.tga", "effect/soul_l.tga", "effect/soul_i.tga", "effect/soul_n.tga", "effect/soul_k.tga"},
			AttachedEntity: true,
		}},
	}
}

func kaupeDodgeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * EffectFrameDuration,
		SFX:      []string{"effect\\t_회피.wav"}, SFXDelays: []time.Duration{15 * EffectFrameDuration},
		Components: []EffectComponent{{Kind: EffectComponentFUNC, FuncName: "KaupeDodge", TextureFile: "effect/whitelight.tga"}},
	}
}

func reflectBodyEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("ReflectBody", "effect\\t_마법반사.wav", 200)
}

func fourWayBodyEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("FourWayBody", "", 100)
}

func asuraBodyEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("AsuraBody", "", 500)
}

func asuraBodyMonsterEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("AsuraBodyMonster", "effect\\t_공격력.wav", 150)
}

func babyBodyEffectSpec() EffectSpec {
	return bodySizeEffectSpec("EffectSmallTransition", 300*time.Millisecond, 2.5)
}

func blueBodyEffectSpec() EffectSpec {
	return bodyTransformEffectSpec("BlueBody", "effect\\t_치잉.wav", 200)
}

func blueHitEffectSpec() EffectSpec {
	return funcEffectSpec("BlueHit", 60*EffectFrameDuration, true)
}

func quakeBody3EffectSpec() EffectSpec {
	return funcEffectSpec("QuakeBody3", 60*EffectFrameDuration, true)
}

func m07EffectSpec() EffectSpec {
	return sprEffectSpec("m_ef07", "", true, false)
}

func kaizelEffectSpec() EffectSpec {
	spec := hatedEffectSpec()
	spec.SFX = []string{"effect\\priest_resurrection.wav"}
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "Kaizel",
		TextureFile: "effect/magic_blue.tga", AttachedEntity: true,
	})
	return spec
}
