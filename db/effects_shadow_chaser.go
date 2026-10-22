package db

func stretchEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\bodypaint.wav")
}

func enervationEffectSpec() EffectSpec {
	return strEffectSpecAttached("enervation", "", false)
}

func enervation2EffectSpec() EffectSpec {
	return strEffectSpecAttached("groomy", "", false)
}

func enervation3EffectSpec() EffectSpec {
	return strEffectSpecAttached("ignorance", "", false)
}

func enervation4EffectSpec() EffectSpec {
	return strEffectSpecAttached("laziness", "effect\\laziness.wav", false)
}

func enervation5EffectSpec() EffectSpec {
	return strEffectSpecAttached("unlucky", "", false)
}

func enervation6EffectSpec() EffectSpec {
	return strEffectSpecAttached("weakness", "", false)
}

func bottomManholeEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\dimension.wav")
}

func manholeEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\manhole.wav")
}

func forestLight6EffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\dimension.wav")
}

func bottomAniEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\chaospanic.wav")
}

func bottomMaelstromEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\maelstrom.wav")
}

func bottomBloodyLustEffectSpec() EffectSpec {
	return soundOnlyEffectSpec("effect\\bloodylust.wav")
}
