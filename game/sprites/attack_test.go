package sprites_test

import (
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestBodyMotionHonorsRobrowserActionMetadata(t *testing.T) {
	action := res.ACTAction{
		DelayMS: 100,
		Animations: []res.ACTAnimation{
			{}, {}, {}, {}, {},
		},
	}
	started := time.Now()
	state := sprites.State{
		Started:        started,
		ActionFamily:   sprites.ActionPCSkill,
		FrameOffset:    1,
		HasFrameOffset: true,
		Length:         3,
		HasLength:      true,
		Speed:          50 * time.Millisecond,
		HasSpeed:       true,
	}

	if got := sprites.BodyMotionForState(action, state, started, started.Add(25*time.Millisecond)); got != 1 {
		t.Fatalf("initial offset motion = %d, want 1", got)
	}
	if got := sprites.BodyMotionForState(action, state, started, started.Add(125*time.Millisecond)); got != 3 {
		t.Fatalf("length-limited motion = %d, want 3", got)
	}
	state.Play = false
	state.HasPlay = true
	if got := sprites.BodyMotionForState(action, state, started, started.Add(time.Second)); got != 1 {
		t.Fatalf("play=false motion = %d, want fixed frame offset 1", got)
	}
}

func TestAttackActionFamilyUsesRobrowserWizardRodAction(t *testing.T) {
	femaleWizard := worldstate.Actor{Job: db.JobWizard, Sex: 0, Weapon: 1601}
	if got := sprites.AttackActionFamily(nil, femaleWizard); got != sprites.ActionPCAttack3 {
		t.Fatalf("female Wizard rod attack action = %d, want ATTACK3", got)
	}
	maleWizard := worldstate.Actor{Job: db.JobWizard, Sex: 1, Weapon: 1601}
	if got := sprites.AttackActionFamily(nil, maleWizard); got != sprites.ActionPCAttack2 {
		t.Fatalf("male Wizard rod attack action = %d, want ATTACK2", got)
	}
	unarmedWizard := worldstate.Actor{Job: db.JobWizard, Sex: 0}
	if got := sprites.AttackActionFamily(nil, unarmedWizard); got != sprites.ActionPCAttack1 {
		t.Fatalf("female Wizard unarmed attack action = %d, want ATTACK1", got)
	}
	leftHandRod := worldstate.Actor{Job: db.JobWizard, Sex: 0, Shield: 1601}
	if got := sprites.AttackActionFamily(nil, leftHandRod); got != sprites.ActionPCAttack3 {
		t.Fatalf("female Wizard left-hand rod attack action = %d, want ATTACK3", got)
	}
}

func TestAttackActionFamilyUsesExpandedWeaponViewIDs(t *testing.T) {
	knightSword := worldstate.Actor{Job: db.JobKnight, Weapon: 48}
	if got := sprites.AttackActionFamily(nil, knightSword); got != sprites.ActionPCAttack2 {
		t.Fatalf("Knight two-hand sword view attack action = %d, want ATTACK2", got)
	}
	archerBow := worldstate.Actor{Job: db.JobArcher, Weapon: 73}
	if got := sprites.AttackActionFamily(nil, archerBow); got != sprites.ActionPCAttack2 {
		t.Fatalf("Archer CrossBow view attack action = %d, want ATTACK2", got)
	}
	hunterBow := worldstate.Actor{Job: db.JobHunter, Weapon: 74}
	if got := sprites.AttackActionFamily(nil, hunterBow); got != sprites.ActionPCAttack3 {
		t.Fatalf("Hunter Arbalest view attack action = %d, want ATTACK3", got)
	}
}
