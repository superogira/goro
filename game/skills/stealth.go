package skills

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
)

func LocalStealthAllowsSkill(ctx client.Context, skillID uint16) bool {
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

func CheckStealthSkill(ctx client.Context, skillID uint16, targetID uint32) error {
	if !LocalStealthAllowsSkill(ctx, skillID) {
		return fmt.Errorf("skill cannot be used while hidden")
	}
	if actor, ok, local := scene.ActorForCombatID(ctx, targetID); ok && !local && scene.ActorHasStealth(actor) {
		return fmt.Errorf("target is hidden")
	}
	return nil
}
