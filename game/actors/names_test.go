package actors_test

import (
	"image/color"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func actorDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	labels := actors.ActorDisplayLabels(ctx, actor, isPlayer)
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

func TestApplyActorNameAckUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1002})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Guide#prontera", PartyName: "Adventurers", GuildName: "Knights"})

	if got := world.Actors[300].Name; got != "Guide" {
		t.Fatalf("actor name = %q, want Guide", got)
	}
	if got := world.Actors[300].PartyName; got != "Adventurers" {
		t.Fatalf("actor party = %q, want Adventurers", got)
	}
	if got := world.Actors[300].GuildName; got != "Knights" {
		t.Fatalf("actor guild = %q, want Knights", got)
	}
}

func TestApplyActorNameAckUpdatesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar", PartyName: "Adventurers", GuildName: "Goro"})

	if got := world.Player.Name; got != "Kivutar" {
		t.Fatalf("player name = %q, want Kivutar", got)
	}
	if got := world.Player.PartyName; got != "Adventurers" {
		t.Fatalf("player party = %q, want Adventurers", got)
	}
	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
}

func TestApplyActorNameAckPreservesLocalGuildOnEmptyNameAck(t *testing.T) {
	world := worldstate.New()
	world.Player.GuildName = "Goro"
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, GuildName: "Goro"},
		World:   world,
	}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar"})

	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
	if got := ctx.Session.GuildName; got != "Goro" {
		t.Fatalf("session guild = %q, want Goro", got)
	}
}

func TestApplyActorNameAckClearsGuildStateFromAuthoritativeEmptyGuildName(t *testing.T) {
	s := &session.Session{AccountID: 200, GuildID: 9, GuildName: "Goro", EmblemVersion: 4, Guild: session.Guild{ID: 9, Name: "Goro"}}
	w := &worldstate.World{Player: worldstate.Actor{ID: 200, GuildID: 9, GuildName: "Goro", EmblemVersion: 4}}
	ctx := client.Context{Session: s, World: w}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar", HasGuildName: true})

	if w.Player.GuildID != 0 || w.Player.GuildName != "" || w.Player.EmblemVersion != 0 {
		t.Fatalf("player guild state = %+v, want cleared", w.Player)
	}
	if s.GuildID != 0 || s.GuildName != "" || s.EmblemVersion != 0 || s.Guild.ID != 0 {
		t.Fatalf("session guild id=%d name=%q emblem=%d nested=%+v, want cleared", s.GuildID, s.GuildName, s.EmblemVersion, s.Guild)
	}
}

func TestApplyActorNameAckClearsRemotePartyName(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Alice", PartyName: "Old Party"})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Alice"})

	if got := world.Actors[300].PartyName; got != "" {
		t.Fatalf("actor party = %q, want cleared", got)
	}
}

func TestApplyActorNameAckClearsRemoteGuildFromAuthoritativeEmptyName(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Alice", GuildID: 9, GuildName: "Goro", EmblemVersion: 4})
	ctx := client.Context{Session: &session.Session{AccountID: 100, CharID: 200}, World: world}

	actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Alice", HasGuildName: true})

	if actor := world.Actors[300]; actor.GuildID != 0 || actor.GuildName != "" || actor.EmblemVersion != 0 {
		t.Fatalf("remote actor guild state = %+v, want cleared", actor)
	}
}

func TestActorDisplayNameUsesSelectedCharacterForPlayer(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player"}, true); got != "Kivutar" {
		t.Fatalf("display name = %q, want Kivutar", got)
	}
}

func TestActorDisplayNameIncludesPartyName(t *testing.T) {
	ctx := client.Context{Session: &session.Session{
		CharID:   200,
		Selected: session.Character{ID: 200, Name: "Kivutar"},
		Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 300, Name: "Alice"}},
		},
	}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player", PartyName: "Goro"}, true); got != "Kivutar (Goro)" {
		t.Fatalf("local display name = %q, want Kivutar (Goro)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 300, Name: "Alice", PartyName: "Other Party"}, false); got != "Alice (Other Party)" {
		t.Fatalf("remote display name = %q, want Alice (Other Party)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 400, Name: "Bob"}, false); got != "Bob" {
		t.Fatalf("remote without packet party name = %q, want Bob", got)
	}
}

func TestActorDisplayLabelsIncludeGuildNameOnSecondLine(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	labels := actors.ActorDisplayLabels(ctx, worldstate.Actor{Name: "Player", GuildName: "Knights"}, true)
	if len(labels) != 2 || labels[0] != "Kivutar" || labels[1] != "Knights" {
		t.Fatalf("local labels = %#v, want Kivutar / Knights", labels)
	}

	labels = actors.ActorDisplayLabels(ctx, worldstate.Actor{ID: 300, Name: "Alice", GuildName: "Knights", HasObjectType: true, ObjectType: scene.ActorObjectTypePC}, false)
	if len(labels) != 2 || labels[0] != "Alice" || labels[1] != "Knights" {
		t.Fatalf("actor labels = %#v, want Alice / Knights", labels)
	}

	labels = actors.ActorDisplayLabels(ctx, worldstate.Actor{Name: "Poring", GuildName: "Knights", HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}, false)
	if len(labels) != 1 || labels[0] != "Poring" {
		t.Fatalf("mob labels = %#v, want Poring only", labels)
	}
}

func TestGuildRankTitleFollowsIdentityUpdates(t *testing.T) {
	for _, local := range []bool{false, true} {
		w := worldstate.New()
		s := &session.Session{AccountID: 100, CharID: 200}
		id := uint32(300)
		if local {
			id = s.AccountID
		}
		w.UpsertActor(worldstate.Actor{ID: id, HasObjectType: true, ObjectType: scene.ActorObjectTypePC})
		ctx := client.Context{Session: s, World: w}
		for _, title := range []string{"Leader", "Officer", ""} {
			actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: id, Name: "Alice", GuildName: "Knights", Title: title, HasGuildName: true})
			actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: id, Name: "Alice"})
			w.UpsertActor(worldstate.Actor{ID: id, X: 42})
			actor := w.Actors[id]
			if local {
				actor = w.Player
			}
			want := "Knights"
			if title != "" {
				want += " [" + title + "]"
			}
			labels := actors.ActorDisplayLabels(ctx, actor, local)
			if len(labels) != 2 || labels[1] != want {
				t.Fatalf("local=%t title=%q labels=%q, want %q", local, title, labels, want)
			}
		}
		actors.ApplyActorNameAck(ctx, network.ActorNameAck{ID: id, Name: "Alice", HasGuildName: true})
		if local && (w.Player.GuildTitle != "" || s.GuildTitle != "") || !local && w.Actors[id].GuildTitle != "" {
			t.Fatal("leaving guild retained rank title")
		}
	}
}

func TestActorNameLabelColorUsesYellowForAdmin(t *testing.T) {
	want := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	if got := actors.ActorNameLabelColor(worldstate.Actor{IsAdmin: true}, true); got != want {
		t.Fatalf("local admin label color = %+v, want %+v", got, want)
	}
	if got := actors.ActorNameLabelColor(worldstate.Actor{IsAdmin: true}, false); got != want {
		t.Fatalf("remote admin label color = %+v, want %+v", got, want)
	}
}

func TestActorDisplayNameUsesServerNameBeforeFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Name: "Kafra Employee#izlude", Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Kafra Employee" {
		t.Fatalf("display name = %q, want Kafra Employee", got)
	}
}

func TestActorDisplayNameUsesImportedMonsterFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Poring" {
		t.Fatalf("display name = %q, want Poring from imported DB", got)
	}
}

func TestActorDisplayNameDoesNotLabelUnnamedPlayerJob(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 0}

	if got := actorDisplayName(ctx, actor, false); got != "" {
		t.Fatalf("display name = %q, want empty", got)
	}
}

func TestHoverActorServerNameLookupIncludesCompanions(t *testing.T) {
	for _, actor := range []worldstate.Actor{
		{HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
		{HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary},
	} {
		if !actors.ShouldUseServerNameForHoverActor(actor) {
			t.Fatalf("companion actor should request server name: %+v", actor)
		}
	}
}

func TestActorDisplayNameDoesNotLabelWarpPortal(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	for _, job := range []int16{scene.ActorJobWarpPortal, scene.ActorJobWarpPortalActive, scene.ActorJobWarpPortalWaiting} {
		actor := worldstate.Actor{Job: job}

		if got := actorDisplayName(ctx, actor, false); got != "" {
			t.Fatalf("job %d display name = %q, want empty", job, got)
		}
		if !npc.IsWarpActor(actor) {
			t.Fatalf("job %d expected warp actor classification", job)
		}
	}
}
