package skills

import (
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	worldstate "github.com/kivutar/goro/world"
)

const skillNameBubbleFallback = "Unknown Skill"

var skillNameBubbleExcludedSkills = map[uint16]struct{}{
	db.SkillTFHiding:             {},
	db.SkillASCloaking:           {},
	db.SkillSTChasewalk:          {},
	db.SkillGCCloakingexceed:     {},
	db.SkillRACamouflage:         {},
	db.SkillNCStealthfield:       {},
	db.SkillSCShadowform:         {},
	db.SkillSCInvisibility:       {},
	db.SkillKOYamikumo:           {},
	db.SkillBaFrostjoke:          {},
	db.SkillDCScream:             {},
	db.SkillLGOverbrandBrandish:  {},
	db.SkillLGOverbrandPlusatk:   {},
	db.SkillWmReverberationMelee: {},
	db.SkillWmReverberationMagic: {},
	db.SkillWLTetravortexFire:    {},
	db.SkillWLTetravortexWater:   {},
	db.SkillWLTetravortexWind:    {},
	db.SkillWLTetravortexGround:  {},
	db.SkillWLSummonAtkFire:      {},
	db.SkillWLSummonAtkWind:      {},
	db.SkillWLSummonAtkWater:     {},
	db.SkillWLSummonAtkGround:    {},
}

func SkillNameBubbleExcluded(skillID uint16) bool {
	_, ok := skillNameBubbleExcludedSkills[skillID]
	return ok
}

func ActorSupportsSkillNameBubble(actor worldstate.Actor, local bool) bool {
	if local || scene.ActorIsPlayerObject(actor) {
		return true
	}
	if !actor.HasObjectType {
		return false
	}
	switch actor.ObjectType {
	case scene.ActorObjectTypeDisguised, scene.ActorObjectTypePet, scene.ActorObjectTypeHomunculus, scene.ActorObjectTypeMercenary, scene.ActorObjectTypeElemental:
		return true
	default:
		return false
	}
}

func SkillNameBubbleText(ctx client.Context, skillID uint16) string {
	name := skillNameBubbleDisplayName(ctx, skillID)
	return name + " !!"
}

func skillNameBubbleDisplayName(ctx client.Context, skillID uint16) string {
	if ctx.Resources != nil {
		if name, ok := ctx.Resources.SkillDisplayName(int(skillID)); ok {
			if name = strings.TrimSpace(name); name != "" {
				return name
			}
		}
	}
	if skill, ok := SkillByID(ctx.Session, skillID); ok {
		if name := strings.TrimSpace(skill.Name); name != "" {
			return name
		}
	}
	return skillNameBubbleFallback
}
