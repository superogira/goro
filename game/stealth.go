package game

import (
	"fmt"

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
	if actor.EffectState&(db.EffectStateCloak|db.EffectStateChasewalk) == 0 && (local || localPlayerIsAdmin(ctx)) {
		return scene.StealthShadow
	}
	return scene.StealthHidden
}

func localStealthAllowsSkill(ctx client.Context, skillID uint16) bool {
	// Commands to companions are independent of their owner's stealth.
	if skillCasterKindForID(skillID) != skillCasterPlayer {
		return true
	}
	if ctx.PlayerHasEffectState(db.EffectStateChasewalk) {
		return skillID == db.SkillSTChasewalk
	}
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		switch skillID {
		case db.SkillTFHiding, db.SkillASGrimtooth, db.SkillRGBackstap, db.SkillRGRaid, db.SkillNJShadowjump, db.SkillNJKirikage:
			return true
		default:
			return false
		}
	}
	return true
}

func checkStealthSkill(ctx client.Context, skillID uint16, targetID uint32) error {
	if !localStealthAllowsSkill(ctx, skillID) {
		return fmt.Errorf("skill cannot be used while hidden")
	}
	if actor, ok, local := scene.ActorForCombatID(ctx, targetID); ok && !local && scene.ActorHasStealth(actor) {
		return fmt.Errorf("target is hidden")
	}
	return nil
}
