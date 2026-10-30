package npc

import (
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func ApplySpriteChange(ctx client.Context, change network.NPCSpriteChange) {
	if ctx.World == nil || change.ID == 0 || scene.IsLocalActor(ctx, change.ID) || change.ID == ctx.World.Player.ID {
		return
	}
	if change.Job > math.MaxInt16 || res.HasPlayerJobToken(int(change.Job)) {
		return
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok || scene.ActorRepresentsPlayer(actor) {
		return
	}
	actor.Job = int16(change.Job)
	actor.Appearance = true
	// Sprite/model caches are keyed by job, so the next draw selects the new
	// resources without changing views shared by other actors. Updating the
	// appearance leaves the actor's movement and other state intact.
	ctx.World.Actors[change.ID] = actor
}

// Like GameModeShowEFST, this updates the displayed effects without replacing
// the actor's movement, appearance, or body/health states.
func ApplyShowEffectUpdate(ctx client.Context, update network.NPCShowEffectUpdate) bool {
	if ctx.World == nil {
		return false
	}
	actor, ok := ctx.World.Actors[update.ActorID]
	if !ok {
		return false
	}
	actor.Opt3State = update.Opt3State
	actor.HasState = true
	actor.Level, actor.HasLevel = update.Level, true
	ctx.World.Actors[actor.ID] = actor
	return true
}

func IsWarpActor(actor worldstate.Actor) bool {
	return IsWarpJob(int(actor.Job))
}

func IsWarpJob(job int) bool {
	return job == scene.ActorJobWarpPortal || job == scene.ActorJobWarpPortalActive || job == scene.ActorJobWarpPortalWaiting
}

func CanTalk(actor worldstate.Actor) bool {
	if actor.ID == 0 || !actor.HasObjectType || actor.Job == scene.ActorJobHiddenWarpNPC {
		return false
	}
	switch actor.ObjectType {
	case scene.ActorObjectTypeNPC, scene.ActorObjectTypeNPC2:
		return !IsWarpActor(actor)
	default:
		return false
	}
}
