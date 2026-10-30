package actors_test

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestAdminSkinLookupUsesClientInfoAdminList(t *testing.T) {
	sessionState := &session.Session{
		AccountID: 2000002,
		CharID:    150000,
		AdminList: []uint32{2000002, 2000003},
	}
	ctx := client.Context{
		Session: sessionState,
		World:   worldstate.New(),
	}

	if !actors.LocalPlayerIsAdmin(ctx) {
		t.Fatal("local player account ID was not recognized as admin")
	}
	if !actors.ActorIsAdmin(ctx, 2000003) {
		t.Fatal("remote admin ID was not recognized")
	}
	if actors.ActorIsAdmin(ctx, 2000004) {
		t.Fatal("non-admin actor was recognized as admin")
	}

	actors.UpsertNetworkActor(ctx, network.ActorEntry{ID: 2000003, Job: 0, Sex: 1, Appearance: true})
	actor := ctx.World.Actors[2000003]
	if !actor.IsAdmin {
		t.Fatalf("network actor was not marked admin: %+v", actor)
	}
}
