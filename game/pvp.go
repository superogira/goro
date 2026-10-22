package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func applyMapPropertyNotify(ctx client.Context, notify network.MapPropertyNotify) {
	if ctx.World == nil {
		return
	}
	next := worldstate.MapProperty(notify.Property)
	if ctx.World.MapProperty != next || !next.PvPRankingEnabled() {
		ctx.World.ClearPvPRankings()
	}
	ctx.World.MapProperty = next
}

func applyPvPRanking(ctx client.Context, ranking network.PvPRanking) {
	if ctx.World == nil || !ctx.World.MapProperty.PvPRankingEnabled() || ranking.ActorID == 0 {
		return
	}
	if scene.IsLocalActor(ctx, ranking.ActorID) {
		setActorPvPRanking(&ctx.World.Player, ranking)
		return
	}
	actor, ok := ctx.World.Actors[ranking.ActorID]
	if !ok {
		return
	}
	setActorPvPRanking(&actor, ranking)
	ctx.World.Actors[ranking.ActorID] = actor
}

func setActorPvPRanking(actor *worldstate.Actor, ranking network.PvPRanking) {
	actor.PvPRank = int(ranking.Rank)
	actor.PvPTotal = int(ranking.Total)
	actor.HasPvPRanking = true
}

func (m *WorldMode) applyPvPInfo(info network.PvPInfo) {
	m.ui.console.AddBlueMessage("PvP: %d wins, %d losses, %d points.", info.Wins, info.Losses, info.Points)
}
