package battle

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	worldstate "github.com/kivutar/goro/world"
)

func TestSenseActorUsesRecentRequestedTarget(t *testing.T) {
	now := time.Now()
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
		301: {ID: 301, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
	}}
	mode := &System{}
	mode.RememberSenseTarget(db.SkillWZEstimation, 301, now)

	actor, ok := mode.senseActor(client.Context{World: world}, 1002, now.Add(time.Second))
	if !ok || actor.ID != 301 {
		t.Fatalf("sense actor = %+v ok=%t, want requested actor 301", actor, ok)
	}
	if mode.senseRequest.targetID != 0 {
		t.Fatalf("sense request was not consumed: %+v", mode.senseRequest)
	}
}

func TestSenseActorDoesNotGuessPartyResultFromVisibleMonsters(t *testing.T) {
	now := time.Now()
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
	}}
	mode := &System{}
	if actor, ok := mode.senseActor(client.Context{World: world}, 1002, now); ok {
		t.Fatalf("party Sense result was guessed as actor %+v", actor)
	}
}

func TestSenseActorRejectsExpiredRequest(t *testing.T) {
	now := time.Now()
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob},
	}}
	mode := &System{senseRequest: senseRequest{targetID: 300, requestedAt: now.Add(-senseResponseTimeout - time.Millisecond)}}
	if actor, ok := mode.senseActor(client.Context{World: world}, 1002, now); ok {
		t.Fatalf("expired Sense request matched actor %+v", actor)
	}
	if mode.senseRequest.targetID != 0 {
		t.Fatalf("expired Sense request was not cleared: %+v", mode.senseRequest)
	}
}
