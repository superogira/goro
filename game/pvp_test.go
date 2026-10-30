package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestPvPMapAllowsPlayerAttackAndEnemySkillTargets(t *testing.T) {
	world := worldstate.New()
	world.MapProperty = worldstate.MapPropertyFreePvPZone
	actor := worldstate.Actor{
		ID:            300,
		ObjectType:    scene.ActorObjectTypePC,
		HasObjectType: true,
	}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	if !actors.ActorCanBeAttackClicked(ctx, actor) {
		t.Fatal("player was not attackable on a PvP map")
	}
	if !actors.ActorCanBeSkillTargeted(ctx, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, actor) {
		t.Fatal("player was not an enemy-skill target on a PvP map")
	}

	world.MapProperty = worldstate.MapPropertyNothing
	if actors.ActorCanBeAttackClicked(ctx, actor) {
		t.Fatal("player remained attackable outside PvP")
	}
	if actors.ActorCanBeSkillTargeted(ctx, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, actor) {
		t.Fatal("player remained an enemy-skill target outside PvP")
	}
}

func TestPvPTargetsLegacyPlayerEntryWithoutObjectType(t *testing.T) {
	world := worldstate.New()
	world.MapProperty = worldstate.MapPropertyFreePvPZone
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}
	legacyPlayer := worldstate.Actor{ID: 300, Job: db.JobArcher, Appearance: true}

	if !actors.ActorCanBeAttackClicked(ctx, legacyPlayer) {
		t.Fatal("legacy player entry without object type was not attackable on a PvP map")
	}
	if !actors.ActorCanBeSkillTargeted(ctx, session.Skill{ID: 13, Type: skills.SkillTargetEnemy}, legacyPlayer) {
		t.Fatal("legacy player entry without object type was not an enemy-skill target on a PvP map")
	}

	explicitMob := legacyPlayer
	explicitMob.HasObjectType = true
	explicitMob.ObjectType = scene.ActorObjectTypeMob
	if !actors.ActorCanBeAttackClicked(ctx, explicitMob) {
		t.Fatal("explicit mob stopped being attackable")
	}
	if scene.ActorRepresentsPlayer(explicitMob) {
		t.Fatal("explicit mob was classified as a player from its job value")
	}

	moveOnly := worldstate.Actor{ID: 400, Job: db.JobNovice}
	if scene.ActorRepresentsPlayer(moveOnly) || actors.ActorCanBeAttackClicked(ctx, moveOnly) {
		t.Fatal("movement-only actor was classified as a player from its zero-value job")
	}
}

func TestWoETargetingUsesGuildRelationship(t *testing.T) {
	world := worldstate.New()
	world.MapProperty = worldstate.MapPropertyAgitZone
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, Guild: session.Guild{ID: 10}},
		World:   world,
	}
	enemy := worldstate.Actor{ID: 300, GuildID: 20, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	guildless := worldstate.Actor{ID: 301, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	mate := worldstate.Actor{ID: 302, GuildID: 10, ObjectType: scene.ActorObjectTypePC, HasObjectType: true}

	for _, actor := range []worldstate.Actor{enemy, guildless} {
		if !actors.ActorCanBeAttackClicked(ctx, actor) || !actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, actor) {
			t.Fatalf("WoE enemy was not targetable: %+v", actor)
		}
	}
	if actors.ActorCanBeAttackClicked(ctx, mate) || actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, mate) {
		t.Fatal("guild member was targetable as a WoE enemy")
	}
}

func TestWoETargetingFollowsAllianceChanges(t *testing.T) {
	w := worldstate.New()
	w.MapProperty = worldstate.MapPropertyAgitZone
	s := &session.Session{Guild: session.Guild{ID: 10}}
	ctx := client.Context{Session: s, World: w}
	for _, kind := range []byte{scene.ActorObjectTypePC, scene.ActorObjectTypeHomunculus, scene.ActorObjectTypeMercenary} {
		actor := worldstate.Actor{ID: 300, GuildID: 20, ObjectType: kind, HasObjectType: true}
		for _, relation := range []uint32{session.GuildRelationAlliance, session.GuildRelationOpposition} {
			s.Guild.Relations = []session.GuildRelation{{GuildID: 20, Relation: relation}}
			want := relation == session.GuildRelationOpposition
			if actors.ActorCanBeAttackClicked(ctx, actor) != want || actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, actor) != want {
				t.Fatalf("targeting kind=%d relation=%d, want enemy=%t", kind, relation, want)
			}
		}
		s.Guild.Relations = nil
		if !actors.ActorCanBeAttackClicked(ctx, actor) {
			t.Fatal("former ally stayed protected after relation removal")
		}
	}
}

func TestWoETargetingIncludesEnemyCompanions(t *testing.T) {
	world := worldstate.New()
	world.MapProperty = worldstate.MapPropertyAgitZone
	ctx := client.Context{
		Session: &session.Session{
			Guild:      session.Guild{ID: 10},
			Homunculus: session.Companion{ID: 400},
			Mercenary:  session.Companion{ID: 500},
		},
		World: world,
	}
	for _, tc := range []struct {
		name       string
		objectType byte
		ownID      uint32
		enemyID    uint32
	}{
		{name: "homunculus", objectType: scene.ActorObjectTypeHomunculus, ownID: 400, enemyID: 401},
		{name: "mercenary", objectType: scene.ActorObjectTypeMercenary, ownID: 500, enemyID: 501},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enemy := worldstate.Actor{ID: tc.enemyID, GuildID: 20, ObjectType: tc.objectType, HasObjectType: true}
			guildmate := worldstate.Actor{ID: tc.enemyID + 100, GuildID: 10, ObjectType: tc.objectType, HasObjectType: true}
			own := worldstate.Actor{ID: tc.ownID, GuildID: 20, ObjectType: tc.objectType, HasObjectType: true}

			if !actors.ActorCanBeAttackClicked(ctx, enemy) || !actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, enemy) {
				t.Fatalf("enemy %s was not targetable in WoE", tc.name)
			}
			if actors.ActorCanBeAttackClicked(ctx, guildmate) || actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, guildmate) {
				t.Fatalf("guild member's %s was targetable in WoE", tc.name)
			}
			if actors.ActorCanBeAttackClicked(ctx, own) || actors.ActorCanBeSkillTargeted(ctx, session.Skill{Type: skills.SkillTargetEnemy}, own) {
				t.Fatalf("local %s was targetable in WoE", tc.name)
			}
		})
	}
}

func TestWoEShowsPlayerNameLabels(t *testing.T) {
	state := worldstate.New()
	ctx := client.Context{World: state}
	actor := worldstate.Actor{Name: "Enemy", ObjectType: scene.ActorObjectTypePC, HasObjectType: true}
	mode := &WorldMode{}
	if labels := mode.actors().HoveredActorDisplayLabels(ctx, actor, time.Time{}); len(labels) == 0 || labels[0] != "Enemy" {
		t.Fatalf("ordinary-map player labels = %q", labels)
	}
	state.MapProperty = worldstate.MapPropertyAgitZone
	if labels := mode.actors().HoveredActorDisplayLabels(ctx, actor, time.Time{}); len(labels) == 0 || labels[0] != "Enemy" {
		t.Fatalf("siege player labels = %q", labels)
	}
}

func TestApplyPvPRankingUpdatesMatchingActorsOnlyInRankedPvP(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 200}
	world.Actors[300] = worldstate.Actor{ID: 300}
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyPvPRanking(ctx, network.PvPRanking{ActorID: 100, Rank: 2, Total: 8})
	if world.Player.HasPvPRanking {
		t.Fatal("ranking was applied before a PvP map property")
	}

	applyMapPropertyNotify(ctx, network.MapPropertyNotify{Property: uint16(worldstate.MapPropertyFreePvPZone)})
	applyPvPRanking(ctx, network.PvPRanking{ActorID: 100, Rank: 2, Total: 8})
	if !world.Player.HasPvPRanking || world.Player.PvPRank != 2 || world.Player.PvPTotal != 8 {
		t.Fatalf("local ranking = %+v, want 2/8", world.Player)
	}

	applyPvPRanking(ctx, network.PvPRanking{ActorID: 300, Rank: 5, Total: 8})
	remote := world.Actors[300]
	if !remote.HasPvPRanking || remote.PvPRank != 5 || remote.PvPTotal != 8 {
		t.Fatalf("remote ranking = %+v, want 5/8", remote)
	}

	applyPvPRanking(ctx, network.PvPRanking{ActorID: 999, Rank: 1, Total: 8})
	if _, ok := world.Actors[999]; ok {
		t.Fatal("ranking packet created an actor that was not in the world")
	}
}

func TestMapPropertyChangeClearsStalePvPRankings(t *testing.T) {
	world := worldstate.New()
	world.MapProperty = worldstate.MapPropertyFreePvPZone
	world.Player = worldstate.Actor{ID: 200, PvPRank: 1, PvPTotal: 4, HasPvPRanking: true}
	world.Actors[300] = worldstate.Actor{ID: 300, PvPRank: 2, PvPTotal: 4, HasPvPRanking: true}
	ctx := client.Context{World: world}

	applyMapPropertyNotify(ctx, network.MapPropertyNotify{Property: uint16(worldstate.MapPropertyNothing)})
	if world.MapProperty != worldstate.MapPropertyNothing {
		t.Fatalf("map property = %d, want nothing", world.MapProperty)
	}
	if world.Player.HasPvPRanking || world.Actors[300].HasPvPRanking {
		t.Fatal("rankings survived leaving the PvP map")
	}
}

func TestApplyPvPInfoAddsClassicConsoleSummary(t *testing.T) {
	mode := &WorldMode{}
	mode.applyPvPInfo(network.PvPInfo{Wins: 7, Losses: 3, Points: 42})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "PvP: 7 wins, 3 losses, 42 points." {
		t.Fatalf("console messages = %+v", messages)
	}
}
