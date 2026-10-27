package db

import "time"

func turnUndeadEffectSpec() EffectSpec {
	spec := funcEffectSpec("TurnUndead", 250*EffectFrameDuration, true)
	spec.SFX = []string{"effect\\ef_bash.wav"}
	spec.SFXDelays = []time.Duration{37 * EffectFrameDuration}
	spec.Components[0].TextureFile = "effect/alpha_down.tga"
	spec.Components[0].TextureFiles = []string{"effect/alpha_center.tga", "effect/ring_liner.tga"}
	return spec
}

func reviveEffectSpec() EffectSpec {
	spec := funcEffectSpec("Revive", 250*EffectFrameDuration, true)
	spec.Components[0].TextureFile = "effect/ring_yellow.tga"
	return spec
}

func gloriaEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("gloria", "gloria_min", "effect\\priest_gloria.wav", false)
}

func magnificatEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("magnificat", "magnificat_min", "effect\\priest_magnificat.wav", false)
}

func resurrectionEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("resurrection", "resurrection_min", "effect\\priest_resurrection.wav", false)
}

func recoveryEffectSpec() EffectSpec {
	return strEffectSpecAttached("recovery", "effect\\priest_recovery.wav", false)
}

func sanctuaryEffectSpec() EffectSpec {
	return strEffectSpecAttached("sanctuary", "effect\\priest_sanctuary.wav", false)
}

func impositioEffectSpec() EffectSpec {
	return strEffectSpecAttached("impositio", "effect\\priest_impositio.wav", false)
}

func lexAeternaEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("lexaeterna", "lexaeterna_min", "effect\\priest_lexaeterna.wav", false)
}

func aspersioEffectSpec() EffectSpec {
	return strEffectSpecAttached("aspersio", "effect\\priest_aspersio.wav", false)
}

func lexDivinaEffectSpec() EffectSpec {
	return strEffectSpecAttached("lexdivina", "effect\\priest_lexdivina.wav", false)
}

func suffragiumEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("suffragium", "suffragium_min", "effect\\priest_suffragium.wav", false)
}

func benedictioEffectSpec() EffectSpec {
	return strEffectSpecAttached("benedictio", "effect\\priest_benedictio.wav", false)
}

func kyrieEffectSpec() EffectSpec {
	return strEffectSpecAttachedMin("kyrie", "kyrie_min", "effect\\priest_kyrie_eleison_a.wav", false)
}

func magnusEffectSpec() EffectSpec {
	return strEffectSpec("magnus", "effect\\priest_magnus.wav")
}

func slowPoisonEffectSpec() EffectSpec {
	return strEffectSpec("slowp", "effect\\priest_slowpoison.wav")
}

func bottomMagnusEffectSpec() EffectSpec {
	return groundWallsEffectSpec("effect/ring_red.tga", 9999)
}

func bottomSancEffectSpec() EffectSpec {
	return groundWallsEffectSpec("effect/alpha_down.tga", 9999)
}
