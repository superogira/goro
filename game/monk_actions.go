package game

import (
	"time"

	"github.com/kivutar/goro/db"
)

// Sakexe's GameActorMsgHandler selects individual frames from the Monk's
// skill action; playing the entire action shows unrelated poses in sequence.
func monkSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	spec := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true}
	switch skillID {
	case db.SkillMOSteelbody:
		spec.frame = 0
		spec.duration = 20 * time.Millisecond
	case db.SkillMOBladestop:
		spec.frame = 3
		spec.duration = 3 * time.Second
	case db.SkillMOInvestigate, db.SkillMOFingeroffensive:
		spec.frame = 3
		spec.length = 2
		spec.speed = 10 * db.EffectFrameDuration
		spec.play = true
		spec.duration = 1100 * time.Millisecond
	case db.SkillMOChaincombo:
		spec.frame = 5
	case db.SkillMOCombofinish:
		spec.frame = 1
		spec.length = 2
		spec.speed = 6 * db.EffectFrameDuration
		spec.play = true
	case db.SkillMOTripleattack:
		spec.action = skillActorActionAttack3
		spec.length = 8
		spec.speed = 4 * db.EffectFrameDuration
		spec.play = true
	default:
		return skillActionSpec{}, false
	}
	return spec, true
}
