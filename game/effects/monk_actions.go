package effects

import (
	"time"

	"github.com/kivutar/goro/db"
)

// Sakexe's GameActorMsgHandler selects individual frames from the Monk's
// skill action; playing the entire action shows unrelated poses in sequence.
func monkSkillActionSpec(skillID uint16) (SkillActionSpec, bool) {
	spec := SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true}
	switch skillID {
	case db.SkillMOSteelbody:
		spec.Frame = 0
		spec.Duration = 20 * time.Millisecond
	case db.SkillMOBladestop:
		spec.Frame = 3
		spec.Duration = 3 * time.Second
	case db.SkillMOInvestigate, db.SkillMOFingeroffensive:
		spec.Frame = 3
		spec.Length = 2
		spec.Speed = 10 * db.EffectFrameDuration
		spec.Play = true
		spec.Duration = 1100 * time.Millisecond
	case db.SkillChPalmstrike:
		spec.Frame = 3
		spec.Length = 2
		spec.Speed = 10 * db.EffectFrameDuration
		spec.Play = true
		spec.Duration = 20 * db.EffectFrameDuration
	case db.SkillChTigerfist:
		spec.Frame = 4
		spec.Combo = true
	case db.SkillMOChaincombo:
		spec.Frame = 5
		spec.Combo = true
	case db.SkillMOTripleattack:
		spec.Action = SkillActorActionAttack3
		spec.Length = 8
		spec.Speed = 4 * db.EffectFrameDuration
		spec.Play = true
		spec.Combo = true
	case db.SkillMOCombofinish:
		spec.Action = skillActorActionAttack2
		spec.Frame = 3
		spec.Length = 2
		spec.Speed = 4 * db.EffectFrameDuration
		spec.Play = true
		spec.Combo = true
	case db.SkillMOExplosionspirits:
		spec.Frame = 1
		spec.Length = 2
		spec.Speed = 30 * db.EffectFrameDuration
		spec.Play = true
		spec.Duration = time.Second
	default:
		return SkillActionSpec{}, false
	}
	return spec, true
}
