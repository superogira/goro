package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	worldstate "github.com/kivutar/goro/world"
)

func TestActorShadowSuppressionUsesActorActions(t *testing.T) {
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte}
	bat := worldstate.Actor{ID: 300, Job: 1005}
	homunculus := worldstate.Actor{ID: 400, Job: 6001}
	archer := worldstate.Actor{ID: 500, Job: 6017, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}
	lancer := worldstate.Actor{ID: 600, Job: 6027, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}
	fencer := worldstate.Actor{ID: 700, Job: 6037, ObjectType: scene.ActorObjectTypeMercenary, HasObjectType: true}
	for _, tc := range []struct {
		name       string
		actor      worldstate.Actor
		action     int
		suppressed bool
	}{
		{"player idle", player, scene.SpriteActionIdle, false},
		{"player attack 1", player, sprites.ActionPCAttack1, false},
		{"player attack 2", player, sprites.ActionPCAttack2, false},
		{"player attack 3", player, sprites.ActionPCAttack3, false},
		{"player combat ready", player, sprites.ActionPCReadyFight, false},
		{"player sitting", player, scene.SpriteActionSit, true},
		{"player dead", player, sprites.ActionPCDeath, true},
		{"bat idle", bat, scene.SpriteActionIdle, false},
		{"bat attacking", bat, sprites.ActionNonPCAttack, false},
		{"bat hurt", bat, sprites.ActionNonPCHurt, false},
		{"bat dead", bat, sprites.ActionNonPCDeath, true},
		{"monster special", bat, sprites.ActionNonPCSpecial, false},
		{"homunculus attacking", homunculus, sprites.ActionNonPCAttack, false},
		{"homunculus dead", homunculus, sprites.ActionNonPCDeath, true},
		{"archer attacking", archer, sprites.ActionPCAttack2, false},
		{"archer sitting", archer, scene.SpriteActionSit, true},
		{"archer dead", archer, sprites.ActionNonPCDeath, true},
		// Archer mercenaries use action 8 for freeze, not player death.
		{"archer frozen", archer, 8, false},
		{"lancer attacking", lancer, sprites.ActionPCAttack3, false},
		{"lancer dead", lancer, sprites.ActionPCDeath, true},
		{"fencer attacking", fencer, sprites.ActionPCAttack2, false},
		{"fencer dead", fencer, sprites.ActionPCDeath, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
				tc.actor.ID: {actionFamily: tc.action, started: now, loop: true},
			}}
			if got := mode.actorShadowSuppressed(tc.actor, now); got != tc.suppressed {
				t.Fatalf("shadow suppressed = %t, want %t", got, tc.suppressed)
			}
		})
	}
}

func TestActorShadowRemainsAfterMaceSwing(t *testing.T) {
	now := time.Now()
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte, Weapon: db.WeaponMace}
	mode := &WorldMode{actorAnims: map[uint32]actorAnimation{
		player.ID: {
			actionFamily: attackActionFamilyForActorWithResources(nil, player),
			started:      now,
			duration:     time.Second,
			next:         readyFightAnimation(now.Add(time.Second)),
		},
	}}
	for _, elapsed := range []time.Duration{0, time.Second, 2 * time.Second} {
		if mode.actorShadowSuppressed(player, now.Add(elapsed)) {
			t.Fatalf("shadow disappeared at %s into the attack/ready transition", elapsed)
		}
	}
}

func TestSittingActorSuppressesShadowWithoutAnimation(t *testing.T) {
	mode := &WorldMode{}
	player := worldstate.Actor{ID: 200, Job: db.JobAcolyte, Sitting: true}
	if !mode.actorShadowSuppressed(player, time.Now()) {
		t.Fatal("sitting player retained their shadow")
	}
}
