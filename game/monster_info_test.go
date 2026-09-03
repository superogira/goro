package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func TestSenseActorUsesRecentRequestedTarget(t *testing.T) {
	now := time.Now()
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: actorObjectTypeMob},
		301: {ID: 301, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: actorObjectTypeMob},
	}}
	mode := &WorldMode{}
	mode.rememberSenseTarget(db.SkillWZEstimation, 301, now)

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
		300: {ID: 300, Job: 1002, HasObjectType: true, ObjectType: actorObjectTypeMob},
	}}
	mode := &WorldMode{}
	if actor, ok := mode.senseActor(client.Context{World: world}, 1002, now); ok {
		t.Fatalf("party Sense result was guessed as actor %+v", actor)
	}
}

func TestSenseActorRejectsExpiredRequest(t *testing.T) {
	now := time.Now()
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, HasObjectType: true, ObjectType: actorObjectTypeMob},
	}}
	mode := &WorldMode{senseRequest: senseRequest{targetID: 300, requestedAt: now.Add(-senseResponseTimeout - time.Millisecond)}}
	if actor, ok := mode.senseActor(client.Context{World: world}, 1002, now); ok {
		t.Fatalf("expired Sense request matched actor %+v", actor)
	}
	if mode.senseRequest.targetID != 0 {
		t.Fatalf("expired Sense request was not cleared: %+v", mode.senseRequest)
	}
}

func TestMonsterInfoPacketOpensInformationWindow(t *testing.T) {
	now := time.Now()
	uiManager := &worldModeTestUIManager{}
	world := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		300: {ID: 300, Job: 1002, Name: "Poring", HasObjectType: true, ObjectType: actorObjectTypeMob},
	}}
	ctx := client.Context{
		ScreenW:   800,
		ScreenH:   600,
		World:     world,
		UIManager: uiManager,
	}
	mode := &WorldMode{actorLife: map[uint32]actorLife{300: {hp: 50, maxHP: 100}}}
	mode.rememberSenseTarget(db.SkillWZEstimation, 300, now)
	data := make([]byte, 29)
	binary.LittleEndian.PutUint16(data[0:2], network.PacketZCMonsterInfo)
	binary.LittleEndian.PutUint16(data[2:4], 1002)
	binary.LittleEndian.PutUint16(data[4:6], 4)
	binary.LittleEndian.PutUint32(data[8:12], 45)
	for i := 20; i < len(data); i++ {
		data[i] = 100
	}

	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCMonsterInfo, Data: data}, now); next != nil || stop {
		t.Fatalf("monster info changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.monsterInfoWindow.IsOpen() {
		t.Fatal("monster information window did not open")
	}
	if mode.ui.interactionModalOpen() {
		t.Fatal("monster information window should not block world interaction")
	}
	if life := mode.actorLife[300]; life.hp != 45 || life.maxHP != 100 {
		t.Fatalf("cached monster life = %+v, want 45/100", life)
	}
	if len(uiManager.overlays) != 1 {
		t.Fatalf("monster info overlays = %d, want 1", len(uiManager.overlays))
	}
}
