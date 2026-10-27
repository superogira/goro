package db

import "time"

func mvpEffectSpec() EffectSpec {
	return strEffectSpecAttached("mvp", "effect\\st_mvp.wav", false)
}

func baseLevelUpEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1300 * time.Millisecond,
		SFX:      []string{"levelup.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "angel",
			AttachedEntity: true,
		}},
	}
}

func jobLevelUpEffectSpec() EffectSpec {
	return EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "joblvup",
			AttachedEntity: true,
		}},
	}
}

func jobLvUp50EffectSpec() EffectSpec {
	return strEffectSpecAttached("joblvup", "", false)
}

func achCompleteEffectSpec() EffectSpec {
	return strEffectSpecAttachedTexturePath("ach_complete/ppring3", "ach_complete/", "", false)
}
