package actors

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func actorWithVisualJob(actor worldstate.Actor) worldstate.Actor {
	actor.Job = int16(scene.ActorVisualJob(actor))
	return actor
}

func CharacterWithVisualJob(character session.Character) session.Character {
	return characterWithVisualJobForEffectState(character, character.Option)
}

func characterWithVisualJobForEffectState(character session.Character, effectState uint32) session.Character {
	character.Job = int16(scene.VisualJobForEffectState(int(character.Job), effectState))
	return character
}

func LocalPlayerVisualCharacter(ctx client.Context) session.Character {
	if ctx.Session == nil {
		return session.Character{}
	}
	character := ctx.Session.SelectedCharacter()
	effectState := character.Option
	if ctx.World != nil && ctx.World.Player.HasState {
		effectState = ctx.World.Player.EffectState
	} else if ctx.World != nil && ctx.World.Player.EffectState != 0 {
		effectState = ctx.World.Player.EffectState
	}
	return characterWithVisualJobForEffectState(character, effectState)
}

func LocalPlayerVisualJob(ctx client.Context) int {
	if ctx.Session == nil {
		return 0
	}
	return int(LocalPlayerVisualCharacter(ctx).Job)
}

func (m *System) ReloadPlayerSpriteView(ctx client.Context, reason string) {
	if ctx.Config.Headless || ctx.Session == nil || ctx.Resources == nil {
		return
	}
	character := LocalPlayerVisualCharacter(ctx)
	view, status := sprites.LoadPlayer(ctx.Resources, character, ctx.Session.Sex, LocalPlayerIsAdmin(ctx))
	if view != nil {
		m.PlayerView = view
		glog.Debugf("player sprite reloaded reason=%s visual_job=%d %s", reason, character.Job, status)
		return
	}
	glog.Warnf("player sprite reload failed reason=%s visual_job=%d: %s", reason, character.Job, status)
	m.PlayerView = nil
}
