package game

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
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

func (m *WorldMode) applySkillNameBubble(ctx client.Context, sourceID uint32, skillID uint16, now time.Time) {
	if skillID == 0 || skillNameBubbleExcluded(skillID) {
		return
	}
	source, ok, local := scene.ActorForCombatID(ctx, sourceID)
	if !ok || !actorSupportsSkillNameBubble(source, local) {
		return
	}
	m.applySpeechBubble(ctx, network.ChatMessage{GID: sourceID, Text: skillNameBubbleText(ctx, skillID)}, now)
}

func skillNameBubbleExcluded(skillID uint16) bool {
	_, ok := skillNameBubbleExcludedSkills[skillID]
	return ok
}

func actorSupportsSkillNameBubble(actor worldstate.Actor, local bool) bool {
	if local || actorIsPlayerObject(actor) {
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

func skillNameBubbleText(ctx client.Context, skillID uint16) string {
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
	if skill, ok := skillByID(ctx.Session, skillID); ok {
		if name := strings.TrimSpace(skill.Name); name != "" {
			return name
		}
	}
	return skillNameBubbleFallback
}
