package db

import (
	"image/color"
	"time"
)

func berserkPotionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("버서크", "effect\\ac_concentration.wav", false)
	spec.Duration = 151 * EffectFrameDuration
	spec.SFXDelays = []time.Duration{18 * EffectFrameDuration}
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 18 * EffectFrameDuration
	spec.Components = append(spec.Components, EffectComponent{
		Kind: EffectComponentFUNC, FuncName: "BerserkPotion", AttachedEntity: true,
		TextureFile: "effect/alpha_down.tga", TextureFiles: []string{"effect/ac_center2.tga"},
	})
	return spec
}

func potionEffectSpec(file string, c color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 850 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          c,
			STRFile:        file,
			AttachedEntity: true,
		}},
	}
}

func healingPotionEffectSpec(file string, c color.RGBA) EffectSpec {
	spec := potionEffectSpec(file, c)
	spec.SFX = []string{"_heal_effect.wav"}
	return spec
}

func bluePotionEffectSpec() EffectSpec {
	spec := potionEffectSpec("파란포션", color.RGBA{R: 92, G: 150, B: 255, A: 255})
	spec.SFX = []string{"effect\\흡기.wav"}
	return spec
}

func potionRedEffectSpec() EffectSpec {
	return healingPotionEffectSpec("빨간포션", color.RGBA{R: 255, G: 82, B: 70, A: 255})
}

func potionOrangeEffectSpec() EffectSpec {
	return healingPotionEffectSpec("주홍포션", color.RGBA{R: 255, G: 145, B: 58, A: 255})
}

func potionYellowEffectSpec() EffectSpec {
	return healingPotionEffectSpec("노란포션", color.RGBA{R: 255, G: 226, B: 76, A: 255})
}

func potionWhiteEffectSpec() EffectSpec {
	return healingPotionEffectSpec("하얀포션", color.RGBA{R: 245, G: 245, B: 255, A: 255})
}

func potionGreenEffectSpec() EffectSpec {
	return potionEffectSpec("초록포션", color.RGBA{R: 78, G: 225, B: 98, A: 255})
}

func itemFastEffectSpec() EffectSpec {
	return strEffectSpecAttached("집중", "effect\\ac_concentration.wav", false)
}

func itemFast2EffectSpec() EffectSpec {
	return strEffectSpecAttached("각성", "effect\\ac_concentration.wav", false)
}

func foodChocolateEffectSpec() EffectSpec {
	return sprEffectSpec(
		"vallentine",
		"effect\\vallentine.wav",
		true,
		false,
	)
}

func mapaeEffectSpec() EffectSpec {
	return strEffectSpecAttached("mapae", "effect\\mapae.wav", false)
}

func itemFastDownEffectSpec() EffectSpec {
	return sprEffectSpec("fast", "effect\\fast.wav", true, false)
}

func statFoodSTREffectSpec() EffectSpec {
	return strEffectSpecAttached("food_str", "_heal_effect.wav", false)
}

func statFoodINTEffectSpec() EffectSpec {
	return strEffectSpecAttached("food_int", "_heal_effect.wav", false)
}

func statFoodVITEffectSpec() EffectSpec {
	return strEffectSpecAttached("food_vit", "_heal_effect.wav", false)
}

func statFoodAGIEffectSpec() EffectSpec {
	return strEffectSpecAttached("food_agi", "_heal_effect.wav", false)
}

func statFoodDEXEffectSpec() EffectSpec {
	return strEffectSpecAttached("food_dex", "_heal_effect.wav", false)
}

func statFoodLUKEffectSpec() EffectSpec {
	return strEffectSpecAttached("food_luk", "_heal_effect.wav", false)
}

func cookingOKEffectSpec() EffectSpec {
	return strEffectSpecAttached("cook_suc", "_heal_effect.wav", false)
}

func cookingFailEffectSpec() EffectSpec {
	return strEffectSpecAttached("cook_fail", "caramel_die.wav", false)
}

func dropPinkEffectSpec() EffectSpec {
	return dropEffectSpec("pink", true)
}

func dropYellowEffectSpec() EffectSpec {
	return dropEffectSpec("yellow", false)
}

func dropPurpleEffectSpec() EffectSpec {
	return dropEffectSpec("purple", false)
}

func dropBlueEffectSpec() EffectSpec {
	return dropEffectSpec("blue", false)
}

func dropGreenEffectSpec() EffectSpec {
	return dropEffectSpec("green", false)
}

func dropRedEffectSpec() EffectSpec {
	return dropEffectSpec("red", false)
}

func newSuccessEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("grade_enchant/new_success/new_success", "grade_enchant/new_success/", "", false, false)
}

func newFailureEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("grade_enchant/new_failed/new_failed", "grade_enchant/new_failed/", "", false, false)
}

func newIntroEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("grade_enchant/new_intro/new_intro", "grade_enchant/new_intro/", "", false, false)
}

func enchantYellowEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_intro_yellow/ui_intro_yellow", "ui_enchant/ui_intro_yellow/", "", false, false)
}

func enchantSuccessEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_enchant_success/ui_enchant_success", "ui_enchant/ui_enchant_success/", "", false, false)
}

func enchantFailEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_fail/ui_enchant_fail", "ui_enchant/ui_fail/", "", false, false)
}

func enchantBlueEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_intro_blue/ui_intro_blue", "ui_enchant/ui_intro_blue/", "", false, false)
}

func enchantUpSuccessEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_levelup_success/ui_levelup_success", "ui_enchant/ui_levelup_success/", "", false, false)
}

func enchantUpFailEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_fail/ui_levelup_fail", "ui_enchant/ui_fail/", "", false, false)
}

func enchantGreenEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_intro_green/ui_intro_green", "ui_enchant/ui_intro_green/", "", false, false)
}

func enchantResetOKEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_reset_success/ui_reset_success", "ui_enchant/ui_reset_success/", "", false, false)
}

func enchantResetFailEffectSpec() EffectSpec {
	return strEffectSpecTexturePath("ui_enchant/ui_fail/ui_reset_fail", "ui_enchant/ui_fail/", "", false, false)
}

func foodEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 850 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          color.RGBA{R: 255, G: 182, B: 86, A: 255},
			STRFile:        "fruit",
			AttachedEntity: true,
		}},
	}
}

func foodBlueEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 850 * time.Millisecond,
		SFX:      []string{"effect\\흡기.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          color.RGBA{R: 132, G: 112, B: 255, A: 255},
			STRFile:        "fruit_",
			AttachedEntity: true,
		}},
	}
}

func dropEffectSpec(colorName string, renderBefore bool) EffectSpec {
	topFile := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "/dropitem_" + colorName
	topTexturePath := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "/"
	bottomFile := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "_bottom/dropitem_" + colorName + "_bottom"
	bottomTexturePath := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "_bottom/"
	return EffectSpec{
		SFX: []string{"effect\\drop_" + colorName + ".wav"},
		Components: []EffectComponent{
			{
				Kind:         EffectComponentSTR,
				STRFile:      topFile,
				TexturePath:  topTexturePath,
				RenderBefore: renderBefore,
			},
			{
				Kind:         EffectComponentSTR,
				STRFile:      bottomFile,
				TexturePath:  bottomTexturePath,
				RenderBefore: renderBefore,
			},
		},
	}
}
