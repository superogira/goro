package actors

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	worldstate "github.com/kivutar/goro/world"
)

// The 2008 Sakexe hides cloaked bodies, including self and party members.
// CPc::Render (0x005e4920) reveals them only through Clairvoyance, in black.
// CGameActor::SetAttrState (0x00572b80) keeps Hiding shadows for self/GM;
// Invisible takes precedence and is not revealed by Clairvoyance.
func actorStealthView(ctx client.Context, actor worldstate.Actor, local bool) scene.StealthView {
	if !scene.ActorHasStealth(actor) {
		return scene.StealthVisible
	}
	if actor.EffectState&db.EffectStateInvisible != 0 {
		return scene.StealthHidden
	}
	if localActorHasStatus(ctx, db.StatusClairvoyance) {
		return scene.StealthSilhouette
	}
	if actor.EffectState&(db.EffectStateCloak|db.EffectStateChasewalk) == 0 && (local || LocalPlayerIsAdmin(ctx)) {
		return scene.StealthShadow
	}
	return scene.StealthHidden
}

func localActorHasStatus(ctx client.Context, statusID uint16) bool {
	if ctx.Session == nil || ctx.Session.Statuses.Active == nil {
		return false
	}
	_, ok := ctx.Session.Statuses.Active[statusID]
	return ok
}
