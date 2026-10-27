package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
)

type actorEffectStateEffect struct {
	bit      uint32
	effectID int
}

var actorEffectStateEffects = []actorEffectStateEffect{
	{bit: db.EffectStateSight, effectID: effects.Sight},
	{bit: db.EffectStateRuwach, effectID: effects.Ruwach},
}

func (m *WorldMode) applyActorEffectStateEffects(ctx client.Context, actorID uint32, oldState, newState uint32) {
	if actorID == 0 {
		return
	}
	for _, stateEffect := range actorEffectStateEffects {
		if stateEffect.effectID == effects.Sight {
			m.setActorStateEffect(ctx, actorID, effects.Sight, newState&stateEffect.bit != 0)
			continue
		}
		if newState&stateEffect.bit != 0 {
			m.effects().AddIfMissing(ctx, stateEffect.effectID, actorID)
		} else if oldState&stateEffect.bit != 0 {
			m.effects().Remove(stateEffect.effectID, actorID)
		}
	}
}

func (m *WorldMode) syncCurrentActorEffectStateEffects(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.HasState {
		m.applyActorEffectStateEffects(ctx, ctx.World.Player.ID, 0, ctx.World.Player.EffectState)
	}
	if ctx.Session != nil {
		if status, ok := ctx.Session.Statuses.Active[db.StatusLkconcentration]; ok && (!status.HasDuration || time.Now().Before(status.ExpiresAt)) {
			m.setActorStateEffect(ctx, scene.LocalSkillTarget(ctx), effects.LKConcentration, true)
		}
		for _, id := range []uint16{db.StatusRgCconfineM, db.StatusRgCconfineS} {
			if status, ok := ctx.Session.Statuses.Active[id]; ok {
				remaining := time.Until(status.ExpiresAt)
				m.applyCloseConfineStatus(ctx, network.StatusEffectChange{
					StatusID: id, Active: !status.HasDuration || remaining > 0,
					HasDuration: status.HasDuration, Duration: remaining,
				})
			}
		}
	}
	m.syncStarDevil(ctx)
	m.syncActorSpiritEffects(ctx, ctx.World.Player.ID)
	for _, actor := range ctx.World.Actors {
		if actor.HasState {
			m.applyActorEffectStateEffects(ctx, actor.ID, 0, actor.EffectState)
		}
		m.syncActorSpiritEffects(ctx, actor.ID)
	}
}

func (m *WorldMode) removeActorEffectStateEffects(actorID uint32) {
	for _, stateEffect := range actorEffectStateEffects {
		m.effects().Remove(stateEffect.effectID, actorID)
	}
	m.effects().Remove(effects.NPCStop2, actorID)
	m.effects().Remove(effects.Gumgang, actorID)
	m.effects().Remove(effects.DoubleGumgang, actorID)
	m.effects().Remove(effects.RedLightBody, actorID)
	m.effects().Remove(effects.SteelBody, actorID)
	m.effects().Remove(effects.Assumptio, actorID)
	m.effects().Remove(effects.RedBody, actorID)
	m.effects().Remove(effects.Moonlit, actorID)
	m.effects().Remove(effects.PinkBody, actorID)
	m.effects().RemoveMarionetteLinks(actorID)
	m.effects().Remove(effects.LKConcentration, actorID)
	m.effects().Remove(effects.LightBlade, actorID)
	m.effects().Remove(effects.AsuraBody, actorID)
	m.effects().Remove(effects.AsuraBodyMonster, actorID)
	m.effects().Remove(effects.BabyBody, actorID)
	m.effects().Remove(effects.BabyBodyBack, actorID)
	m.effects().Remove(effects.Chookgi, actorID)
	m.effects().Remove(effects.Chookgi2, actorID)
	m.effects().Remove(effects.Chookgi3, actorID)
}

// Sphere counts are authoritative server state, not a side effect of casting
// Call Spirits: spheres can also be spent, absorbed, or expire.
func applySpiritSphereCount(ctx client.Context, update network.SpiritSphereCount) bool {
	actor, ok, local := scene.ActorForCombatID(ctx, update.ActorID)
	if !ok {
		return false
	}
	actor.SpiritSpheres = update.Count
	actor.HasSpiritSpheres = true
	if local {
		ctx.World.Player.SpiritSpheres = update.Count
		ctx.World.Player.HasSpiritSpheres = true
	} else {
		ctx.World.Actors[actor.ID] = actor
	}
	return true
}

// Like GameModeShowEFST, this updates the displayed effects without replacing
// the actor's movement, appearance, or body/health states.
func applyNPCShowEffectUpdate(ctx client.Context, update network.NPCShowEffectUpdate) bool {
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

func (m *WorldMode) applySoulLinkerMonsterSkill(ctx client.Context, notify network.SkillNoDamageNotify) {
	var bit uint32
	switch notify.SkillID {
	case db.SkillSLSwoo:
		bit = db.Opt3Overthrust
	case db.SkillSLSke:
		bit = db.Opt3Energycoat
	case db.SkillSLSka:
		bit = db.Opt3Steelbody
	default:
		return
	}
	actor, ok, local := scene.ActorForCombatID(ctx, notify.TargetID)
	if !ok || local || !scene.IsMonsterLikeHoverActor(actor) {
		return
	}
	// Monsters need not receive status icons. The successful skill starts the
	// effect; actor entries and ZC_NPC_SHOWEFST_UPDATE carry its later state.
	actor.Opt3State |= bit
	actor.HasState = true
	ctx.World.Actors[actor.ID] = actor
	m.syncActorSpiritEffects(ctx, actor.ID)
}

func (m *WorldMode) syncActorSpiritEffects(ctx client.Context, actorID uint32) {
	if scene.IsLocalActor(ctx, actorID) {
		actorID = scene.LocalSkillTarget(ctx)
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	visible := !scene.ActorHasStealth(actor)
	m.setActorStateEffect(ctx, actorID, effects.DoubleGumgang, visible && actor.Opt3State&db.Opt3Warm != 0)
	m.setActorStateEffect(ctx, actorID, effects.RedLightBody, visible && actor.Opt3State&db.Opt3Warm != 0)
	m.setActorStateEffect(ctx, actorID, effects.AsuraBody, visible && actor.Opt3State&db.Opt3Soullink != 0)
	if actor.HasState && scene.IsMonsterLikeHoverActor(actor) {
		shrinking := actor.Opt3State&db.Opt3Overthrust != 0
		if shrinking && visible {
			m.effects().Remove(effects.BabyBodyBack, actorID)
			m.setActorStateEffect(ctx, actorID, effects.BabyBody, true)
		} else {
			for _, effect := range m.effects().Active {
				if effect.EffectID == effects.BabyBody && effect.ActorID == actorID && effect.Persistent {
					m.effects().Remove(effects.BabyBody, actorID)
					if visible && !shrinking {
						m.effects().Add(ctx, effects.BabyBodyBack, actorID)
					}
					break
				}
			}
		}
	}
	if actor.HasState && scene.IsMonsterLikeHoverActor(actor) && actor.Opt3State&db.Opt3Energycoat == 0 {
		m.effects().Remove(effects.AsuraBodyMonster, actorID)
	}
	m.setActorStateEffect(ctx, actorID, effects.Gumgang, visible && actor.Opt3State&db.Opt3Explosionspirits != 0)
	m.setActorStateEffect(ctx, actorID, effects.SteelBody, visible && actor.Opt3State&db.Opt3Steelbody != 0)
	m.setActorStateEffect(ctx, actorID, effects.Assumptio, visible && actor.Opt3State&db.Opt3Assumptio != 0)
	m.setActorStateEffect(ctx, actorID, effects.RedBody, visible && actor.Opt3State&db.Opt3Berserk != 0)
	m.setActorStateEffect(ctx, actorID, effects.LightBlade, visible && actor.Opt3State&db.Opt3Lightblade != 0)
	m.setActorStateEffect(ctx, actorID, effects.Moonlit, visible && actor.Opt3State&db.Opt3Moonlit != 0)
	m.setActorStateEffect(ctx, actorID, effects.PinkBody, visible && actor.Opt3State&db.Opt3Marionette != 0)
	if !visible || actor.HasState && actor.Opt3State&db.Opt3Marionette == 0 {
		m.effects().RemoveMarionetteLinks(actorID)
	}
	sphereEffect := effects.Chookgi
	switch actor.Job {
	case db.JobMonkB:
		sphereEffect = effects.Chookgi2
	case db.JobGunslinger, db.JobGunslingerB:
		sphereEffect = effects.Chookgi3
	}
	if sphereEffect == effects.Chookgi3 && visible && actor.SpiritSpheres > 0 {
		for i := range m.effects().Active {
			effect := &m.effects().Active[i]
			if effect.EffectID == effects.Chookgi3 && effect.ActorID == actorID && effect.HitCount != min(int(actor.SpiritSpheres), 10) {
				effect.HitCount = min(int(actor.SpiritSpheres), 10)
				effect.Starts = time.Now()
				effect.GunslingerCoins = effects.NewGunslingerCoins(*effect)
			}
		}
	}
	for _, id := range [...]int{effects.Chookgi, effects.Chookgi2, effects.Chookgi3} {
		m.setActorStateEffect(ctx, actorID, id, visible && actor.SpiritSpheres > 0 && id == sphereEffect)
	}
}

// A state effect lives until the server clears it. Ground skill units still use
// persistent=true with a deadline; a zero deadline means explicitly removed.
func (m *WorldMode) setActorStateEffect(ctx client.Context, actorID uint32, effectID int, active bool) {
	if !active {
		m.effects().Remove(effectID, actorID)
		return
	}
	m.effects().AddIfMissing(ctx, effectID, actorID)
	now := time.Now()
	for i := range m.effects().Active {
		effect := &m.effects().Active[i]
		if effect.EffectID == effectID && effect.ActorID == actorID && !effect.Expired(now) {
			effect.Persistent = true
			effect.Expires = time.Time{}
		}
	}
}
