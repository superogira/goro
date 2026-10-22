package scene

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func ActionAnimationDuration(action res.ACTAction, fallback time.Duration) time.Duration {
	if len(action.Animations) == 0 {
		return fallback
	}
	delayMS := float64(action.DelayMS)
	if delayMS <= 0 {
		delayMS = 150
	}
	duration := time.Duration(delayMS * float64(time.Millisecond) * float64(len(action.Animations)))
	if duration <= 0 {
		return fallback
	}
	if duration > MaxCombatAnimationDuration {
		return MaxCombatAnimationDuration
	}
	return duration
}

const (
	DefaultAttackAnimationDuration  = 600 * time.Millisecond
	DefaultHitAnimationDuration     = 250 * time.Millisecond
	DefaultDeathAnimationDuration   = 900 * time.Millisecond
	MaxCombatAnimationDuration      = 5 * time.Second
	NonPCDeathFadeDuration          = 5 * time.Second
	MultiHitDelay                   = 200 * time.Millisecond
	ReferenceBowArrowFlightDuration = 8 * 24 * time.Millisecond
)

func ActorForCombatID(ctx client.Context, id uint32) (world.Actor, bool, bool) {
	if ctx.World == nil || id == 0 {
		return world.Actor{}, false, false
	}
	if IsLocalActor(ctx, id) {
		actor := ctx.World.Player
		character := ctx.Session.SelectedCharacter()
		actor.ID = id
		actor.Job = character.Job
		actor.Head = character.Hair
		actor.Weapon = character.Weapon
		actor.Shield = character.Shield
		actor.HeadTop = character.HeadTop
		actor.HeadMid = character.HeadMid
		actor.HeadLow = character.HeadLow
		actor.Sex = ctx.Session.Sex
		actor.Level = ctx.Session.Progress.BaseLevel
		actor.HasLevel = actor.Level > 0
		actor.Appearance = true
		return actor, true, true
	}
	actor, ok := ctx.World.Actors[id]
	return actor, ok, false
}
