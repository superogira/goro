package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) startSkillActionAnimation(ctx client.Context, id uint32, actor worldstate.Actor, spec skillActionSpec, started time.Time, duration time.Duration) {
	anim := spec.actorAnimationForActorWithResources(ctx.Resources, actor, started, duration)
	m.setActorAction(ctx, id, anim)
	if ctx.Session == nil || !scene.IsLocalActor(ctx, id) {
		return
	}
	m.setActorAction(ctx, ctx.Session.AccountID, anim)
	m.setActorAction(ctx, ctx.Session.CharID, anim)
}

type skillActorAction int

const skillActorActionNone skillActorAction = -1

const (
	skillActorActionIdle skillActorAction = iota
	skillActorActionSkill
	skillActorActionAttack
	skillActorActionAttack1
	skillActorActionAttack2
	skillActorActionAttack3
	skillActorActionPickup
	skillActorActionReadyFight
	skillActorActionWalk
)

type skillActionSpec struct {
	defined       bool
	action        skillActorAction
	frame         int
	hasFrame      bool
	length        int
	speed         time.Duration
	duration      time.Duration
	play          bool
	repeat        bool
	pingPong      bool
	randomReverse bool
	delay         time.Duration
	next          *skillActionSpec
}

var (
	idleSkillActionSpec       = newSkillActionSpec(skillActorActionIdle, true, nil)
	defaultSkillActionSpec    = newSkillActionSpec(skillActorActionSkill, false, &idleSkillActionSpec)
	attackSkillActionSpec     = newSkillActionSpec(skillActorActionAttack, false, &idleSkillActionSpec)
	readyFightSkillActionSpec = newSkillActionSpec(skillActorActionReadyFight, false, &idleSkillActionSpec)
)

func newSkillActionSpec(action skillActorAction, repeat bool, next *skillActionSpec) skillActionSpec {
	return skillActionSpec{
		defined: true,
		action:  action,
		frame:   0,
		repeat:  repeat,
		play:    true,
		next:    cloneSkillActionSpec(next),
	}
}

func cloneSkillActionSpec(spec *skillActionSpec) *skillActionSpec {
	if spec == nil {
		return nil
	}
	clone := *spec
	clone.next = cloneSkillActionSpec(spec.next)
	return &clone
}

func (s skillActionSpec) actionFamilyForActorWithResources(manager *res.Manager, actor worldstate.Actor) int {
	if s.action == skillActorActionNone {
		return -1
	}
	mercenary := companion.IsMercenary(actor)
	if !res.HasPlayerJobToken(scene.ActorVisualJob(actor)) && !mercenary {
		return sprites.ActionNonPCAttack
	}
	switch s.action {
	case skillActorActionIdle:
		return scene.SpriteActionIdle
	case skillActorActionWalk:
		return scene.SpriteActionWalk
	case skillActorActionAttack:
		return attackActionFamilyForActorWithResources(manager, actor)
	case skillActorActionAttack1:
		return sprites.ActionPCAttack1
	case skillActorActionAttack2:
		return sprites.ActionPCAttack2
	case skillActorActionAttack3:
		return sprites.ActionPCAttack3
	case skillActorActionPickup:
		return scene.SpriteActionPickup
	case skillActorActionReadyFight:
		if mercenary {
			return sprites.ActionPCSkill
		}
		return sprites.ActionPCReadyFight
	default:
		return sprites.ActionPCSkill
	}
}

func (s skillActionSpec) actorAnimationForActorWithResources(manager *res.Manager, actor worldstate.Actor, started time.Time, duration time.Duration) actorAnimation {
	if !s.defined {
		s = defaultSkillActionSpec
	}
	if s.randomReverse && started.UnixNano()&1 != 0 {
		copy := cloneSkillActionSpec(&s)
		last := copy
		for last.next != nil {
			last = last.next
		}
		end := s.frame + last.frame
		for step := copy; step != nil; step = step.next {
			step.frame = end - step.frame
			step.randomReverse = false
		}
		s = *copy
	}
	if s.duration > 0 {
		duration = s.duration
	}
	if s.delay > 0 && !started.IsZero() {
		started = started.Add(s.delay)
	}
	anim := actorAnimation{
		actionFamily: s.actionFamilyForActorWithResources(manager, actor),
		started:      started,
		startDelay:   s.delay,
		duration:     duration,
		loop:         s.repeat,
		pingPong:     s.pingPong,
		play:         s.play,
		hasPlay:      true,
	}
	if s.hasFrame {
		anim.frameOffset = s.frame
		anim.hasFrameOffset = true
		if !s.play {
			anim.fixedMotion = s.frame
			anim.hasFixedMotion = true
		}
	}
	if s.length > 0 {
		anim.length = s.length
		anim.hasLength = true
	}
	if s.speed > 0 {
		anim.speed = s.speed
		anim.hasSpeed = true
	}
	if s.next != nil && (!s.next.isNaturalIdleFallback() || s.repeat) {
		next := s.next.actorAnimationForActorWithResources(manager, actor, time.Time{}, duration)
		anim.next = &next
	}
	return anim
}

func (s skillActionSpec) isNaturalIdleFallback() bool {
	return s.defined && s.action == skillActorActionIdle && s.repeat && s.play && s.next == nil
}

func importedSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	switch skillID {
	case db.SkillSGFeel, db.SkillSGHate:
		return newSkillActionSpec(skillActorActionSkill, false, &idleSkillActionSpec), true
	case db.SkillSGFusion:
		return idleSkillActionSpec, true
	case db.SkillSGSunWarm, db.SkillSGMoonWarm, db.SkillSGStarWarm:
		return skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 3, duration: time.Second}, true
	case db.SkillSGSunComfort, db.SkillSGMoonComfort, db.SkillSGStarComfort:
		return skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 2, duration: time.Second}, true
	}
	if spec, ok := taekwonSkillActionSpec(skillID); ok {
		return spec, true
	}
	if db.IsSoulLinkSkill(skillID) {
		return skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 3, duration: 2 * time.Second}, true
	}
	if spec, ok := dancerSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := bardSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := monkSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := gunslingerSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := ninjaSkillActionSpec(skillID); ok {
		return spec, true
	}
	if skillID == db.SkillASSonicblow {
		return sonicBlowSkillActionSpec(), true
	}
	switch db.SkillActions[skillID] {
	case db.SkillActionNone:
		return skillActionSpec{defined: true, action: skillActorActionNone}, true
	case db.SkillActionIdle:
		return idleSkillActionSpec, true
	case db.SkillActionAttack:
		return attackSkillActionSpec, true
	case db.SkillActionAttack1:
		return newSkillActionSpec(skillActorActionAttack1, false, &idleSkillActionSpec), true
	case db.SkillActionAttack2:
		return newSkillActionSpec(skillActorActionAttack2, false, &idleSkillActionSpec), true
	case db.SkillActionAttack3:
		return newSkillActionSpec(skillActorActionAttack3, false, &idleSkillActionSpec), true
	case db.SkillActionSkill, db.SkillActionAction:
		return defaultSkillActionSpec, true
	case db.SkillActionAttackFixedFrame:
		spec := newSkillActionSpec(skillActorActionAttack, false, nil)
		spec.hasFrame = true
		spec.frame = 0
		spec.play = false
		return spec, true
	case db.SkillActionPickup:
		return newSkillActionSpec(skillActorActionPickup, false, &idleSkillActionSpec), true
	case db.SkillActionReadyfight:
		return readyFightSkillActionSpec, true
	case db.SkillActionDance:
		return danceSkillActionSpec(), true
	default:
		return skillActionSpec{}, false
	}
}

func skillAction(skillID uint16) skillActionSpec {
	spec := skillEffectSpecFor(skillID).action
	if !spec.defined {
		return defaultSkillActionSpec
	}
	return spec
}

func sonicBlowSkillActionSpec() skillActionSpec {
	next := newSkillActionSpec(skillActorActionReadyFight, true, nil)
	for hit := 8; hit >= 1; hit-- {
		spec := newSkillActionSpec(skillActorActionAttack, false, &next)
		if hit > 1 {
			spec.speed = 30 * time.Millisecond
		}
		next = spec
	}
	return next
}

func bardSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	if skillID == db.SkillBaFrostjoke || skillID == db.SkillBaPangvoice {
		spec := newSkillActionSpec(skillActorActionSkill, true, &idleSkillActionSpec)
		spec.frame, spec.hasFrame, spec.length = 4, true, 2
		spec.speed, spec.duration = 10*db.EffectFrameDuration, 1800*time.Millisecond
		return spec, true
	}
	if skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun {
		spec := danceSkillActionSpec()
		spec.pingPong = false // ST_SING loops 1,2,3; ST_DANCE plays 1,2,3,2.
		return spec, true
	}
	return skillActionSpec{}, false
}

func dancerSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	if skillID == db.SkillDCScream {
		return skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 0, duration: 1200 * time.Millisecond}, true
	}
	return skillActionSpec{}, false
}

func gunslingerSkillActionSpec(id uint16) (skillActionSpec, bool) {
	action, first, last, duration := skillActorActionSkill, 0, 3, time.Second
	switch id {
	case db.SkillGSFling, db.SkillGSMadnesscancel, db.SkillGSAdjustment, db.SkillGSIncreasing, db.SkillGSDesperado, db.SkillGSGatlingfever:
	case db.SkillGSGlittering:
		first, last, duration = 4, 5, 500*time.Millisecond
	case db.SkillGSCracker, db.SkillGSGrounddrift:
		action, last, duration = skillActorActionAttack1, 2, 500*time.Millisecond
	case db.SkillGSTripleaction, db.SkillGSBullseye, db.SkillGSMagicalbullet, db.SkillGSTracking, db.SkillGSDisarm,
		db.SkillGSPiercingshot, db.SkillGSRapidshower, db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack:
		return attackSkillActionSpec, true
	default:
		return skillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := skillActionSpec{defined: true, action: action, hasFrame: true, frame: last, duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = skillActionSpec{defined: true, action: action, hasFrame: true, frame: frame, duration: step, next: cloneSkillActionSpec(&next)}
	}
	next.randomReverse = first == 0 && last == 3
	return next, true
}

func gunslingerCastAction(id uint16, now time.Time) (skillActionSpec, bool) {
	spec := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true, frame: 1 + int(now.UnixNano()&1), duration: 500 * time.Millisecond}
	switch id {
	case db.SkillGSBullseye, db.SkillGSGatlingfever, db.SkillGSTracking, db.SkillGSMadnesscancel,
		db.SkillGSAdjustment, db.SkillGSDisarm, db.SkillGSPiercingshot, db.SkillGSRapidshower:
	case db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack, db.SkillGSGrounddrift:
		spec.action, spec.frame = skillActorActionAttack1, 0
	default:
		return skillActionSpec{}, false
	}
	return spec, true
}

func ninjaSkillActionSpec(id uint16) (skillActionSpec, bool) {
	first, last, duration := 0, 1, 500*time.Millisecond
	switch id {
	case db.SkillNJSyuriken, db.SkillNJKunai, db.SkillNJHuuma, db.SkillNJZenynage,
		db.SkillNJUtsusemi, db.SkillNJKouenka, db.SkillNJHyousyouraku,
		db.SkillNJHuujin, db.SkillNJRaigekisai, db.SkillNJKamaitachi, db.SkillNJNen:
		first, last, duration = 2, 5, time.Second
	case db.SkillNJTatamigaeshi, db.SkillNJKasumikiri, db.SkillNJShadowjump,
		db.SkillNJBunsinjyutsu, db.SkillNJKaensin, db.SkillNJBakuenryu, db.SkillNJHyousensou, db.SkillNJSuiton:
	case db.SkillNJIssen:
		return defaultSkillActionSpec, true
	case db.SkillNJKirikage:
		return newSkillActionSpec(skillActorActionAttack2, false, &idleSkillActionSpec), true
	default:
		return skillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true,
		frame: last, duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = skillActionSpec{defined: true, action: skillActorActionSkill, hasFrame: true,
			frame: frame, duration: step, next: cloneSkillActionSpec(&next)}
	}
	return next, true
}

func danceSkillActionSpec() skillActionSpec {
	spec := newSkillActionSpec(skillActorActionSkill, true, nil)
	spec.hasFrame = true
	spec.frame = 1
	spec.length = 3
	spec.speed = 10 * db.EffectFrameDuration
	spec.pingPong = true
	return spec
}

func taekwonSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	pose := func(action skillActorAction, frame int, duration time.Duration) skillActionSpec {
		return skillActionSpec{defined: true, action: action, hasFrame: true, frame: frame, duration: duration}
	}
	switch skillID {
	case db.SkillTKRun:
		return idleSkillActionSpec, true
	case db.SkillTKReadystorm:
		return pose(skillActorActionSkill, 0, 2*time.Second), true
	case db.SkillTKStormkick:
		return pose(skillActorActionSkill, 1, 2*time.Second), true
	case db.SkillTKReadydown:
		return pose(skillActorActionSkill, 2, 2*time.Second), true
	case db.SkillTKReadyturn:
		return pose(skillActorActionSkill, 3, 2*time.Second), true
	case db.SkillTKReadycounter:
		return pose(skillActorActionSkill, 4, 2*time.Second), true
	case db.SkillTKDodge:
		return pose(skillActorActionPickup, 1, 2*time.Second), true
	case db.SkillTKJumpkick:
		// EF_JUMPKICK replaces the three-second pose at tick five with a 25 ms hold.
		return pose(skillActorActionSkill, 5, 5*db.EffectFrameDuration+25*time.Millisecond), true
	case db.SkillTKDownkick:
		return taekwonKickSequence(skillActorActionAttack2, [5]int{3, 5, 7, 8, 2}, 5), true
	case db.SkillTKTurnkick:
		return taekwonKickSequence(skillActorActionAttack3, [5]int{4, 5, 6, 7, 2}, 5), true
	case db.SkillTKCounter:
		return taekwonKickSequence(skillActorActionAttack1, [5]int{2, 3, 4, 4, 4}, 4), true
	case db.SkillTKHighjump:
		return taekwonJumpSequence(false), true
	default:
		return skillActionSpec{}, false
	}
}

// Native SetForceAnimation2 plays five selected ACT motions, then holds the last
// one until its timeout. The existing animation chain also permits the landing
// pose to finish in a different ACT action.
func taekwonKickSequence(action skillActorAction, frames [5]int, ticks int) skillActionSpec {
	step := time.Duration(ticks) * db.EffectFrameDuration
	next := skillActionSpec{
		defined: true, action: action, hasFrame: true, frame: frames[4],
		duration: 500*time.Millisecond - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = skillActionSpec{
			defined: true, action: action, hasFrame: true, frame: frames[i],
			duration: step, next: cloneSkillActionSpec(&next),
		}
	}
	return next
}

func taekwonJumpSequence(casting bool) skillActionSpec {
	step := 10 * db.EffectFrameDuration
	action, frames := skillActorActionPickup, [5]int{1, 1, 1, 1, 5}
	finalAction := skillActorActionWalk
	if casting {
		action, frames = skillActorActionAttack1, [5]int{0, 0, 0, 1, 1}
		finalAction = action
	}
	next := skillActionSpec{
		defined: true, action: finalAction, hasFrame: true, frame: frames[4],
		duration: 3*time.Second - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = skillActionSpec{
			defined: true, action: action, hasFrame: true, frame: frames[i],
			duration: step, next: cloneSkillActionSpec(&next),
		}
	}
	return next
}
