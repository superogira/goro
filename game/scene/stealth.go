package scene

import (
	"image/color"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

const ActorStealthMask = db.EffectStateHide | db.EffectStateCloak | db.EffectStateInvisible | db.EffectStateChasewalk

type StealthView uint8

const (
	StealthVisible StealthView = iota
	StealthHidden
	StealthShadow
	StealthSilhouette
)

func ActorHasStealth(actor worldstate.Actor) bool {
	return actor.EffectState&ActorStealthMask != 0
}

func (v StealthView) Tint(tint color.RGBA) color.RGBA {
	if v == StealthSilhouette {
		tint.R, tint.G, tint.B = 0, 0, 0
	}
	return tint
}

func (v StealthView) Blend(blend render.Blend) render.Blend {
	if v == StealthSilhouette {
		// An additive black silhouette would disappear completely.
		return render.BlendSourceOver
	}
	return blend
}
