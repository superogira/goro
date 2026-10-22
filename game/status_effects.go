package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) applyStatusEffectChange(ctx client.Context, change network.StatusEffectChange) {
	if ctx.Session == nil || change.StatusID == 0xFFFF {
		return
	}
	if m.applyPushCartStatus(ctx, change) {
		return
	}
	m.applyFalconStatus(ctx, change)
	m.stopPerformance(ctx, change)
	if actorID := applyActorOpt3StateStatus(ctx, change); actorID != 0 {
		m.syncActorSpiritEffects(ctx, actorID)
		if change.StatusID == db.StatusLkconcentration {
			if scene.IsLocalActor(ctx, actorID) {
				actorID = scene.LocalSkillTarget(ctx)
			}
			m.setActorStateEffect(ctx, actorID, effects.LKConcentration, change.Active)
		}
	}
	m.applyTrickDeadStatus(ctx, change)
	m.applyTaekwonStatus(ctx, change)
	m.applyCloseConfineStatus(ctx, change)
	if change.StatusID == db.StatusWzSightblaster {
		id := change.ActorID
		if id == 0 || scene.IsLocalActor(ctx, id) {
			id = scene.LocalSkillTarget(ctx)
		}
		m.setActorStateEffect(ctx, id, effects.SightBlaster, change.Active)
	}
	if change.StatusID == db.StatusSmaReady && change.Active {
		id := change.ActorID
		if id == 0 {
			id = scene.LocalSkillTarget(ctx)
		}
		m.effects().Add(ctx, effects.SmaReady, id)
	}
	m.applyTaekwonNightStatus(ctx, change)
	if !applyLocalStatusEffectChange(ctx.Session, change, time.Now()) {
		return
	}
	if change.StatusID == db.StatusDevil1 {
		m.syncStarDevil(ctx)
	}
	if !change.Active {
		if change.StatusID == db.StatusCashBossAlarm {
			m.ui.minimap.ClearBossMarker()
		}
		glog.Debugf("status effect inactive id=%d actor=%d", change.StatusID, change.ActorID)
		return
	}
	glog.Debugf("status effect active id=%d actor=%d duration_ms=%d", change.StatusID, change.ActorID, change.Duration.Milliseconds())
}

func applyLocalStatusEffectChange(s *session.Session, change network.StatusEffectChange, now time.Time) bool {
	if s == nil || change.StatusID == 0xFFFF || change.StatusID == db.StatusOnPushCart {
		return false
	}
	localID := s.AccountID
	if localID == 0 {
		localID = s.CharID
	}
	if change.ActorID != 0 && localID != 0 && change.ActorID != localID && change.ActorID != s.CharID {
		return false
	}
	if s.Statuses.Active == nil {
		s.Statuses.Active = make(map[uint16]session.StatusEffect)
	}
	if !change.Active {
		delete(s.Statuses.Active, change.StatusID)
		return true
	}
	effect := session.StatusEffect{
		ID:          change.StatusID,
		Source:      change.ActorID,
		StartedAt:   now,
		HasDuration: change.HasDuration,
	}
	if change.HasDuration {
		effect.ExpiresAt = now.Add(change.Duration)
	}
	s.Statuses.Active[change.StatusID] = effect
	return true
}

func (m *WorldMode) applyTaekwonNightStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusSoullink && change.StatusID != db.StatusSke {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 || !scene.IsLocalActor(ctx, id) || m.mapScene.Night == change.Active {
		return
	}
	m.mapScene.Night = change.Active
	// Ground and static-model colors include scene lighting, so rebuild their
	// retained vertices when Soul Link or Eske changes the local night mode.
	m.mapScene.InvalidateLighting()
}

func (m *WorldMode) applyTaekwonStatus(ctx client.Context, change network.StatusEffectChange) {
	if ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 {
		return
	}
	if scene.IsLocalActor(ctx, id) {
		id = scene.LocalSkillTarget(ctx)
	}
	now := time.Now()
	switch change.StatusID {
	case db.StatusRun:
		if m.effects().HasActive(effects.Run, id, now) {
			m.effects().AddAt(ctx, effects.StopRun, id, now)
		}
		m.effects().Remove(effects.Run, id)
		return
	case db.StatusTing:
		m.scheduleSoundAtActor(now, id, "effect\\T_벽튕김.wav")
		m.effects().AddAt(ctx, effects.QuakeBody, id, now)
		m.effects().Remove(effects.Run, id)
		return
	}
}

func applyActorOpt3StateStatus(ctx client.Context, change network.StatusEffectChange) uint32 {
	bit, ok := actorOpt3StateBitForStatus(change.StatusID)
	if !ok || ctx.World == nil {
		return 0
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	if id == 0 {
		return 0
	}
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		return 0
	}
	if change.Active {
		actor.Opt3State |= bit
	} else {
		actor.Opt3State &^= bit
	}
	actor.HasState = true
	if local {
		ctx.World.Player.Opt3State = actor.Opt3State
		ctx.World.Player.HasState = true
	} else {
		upsertActor(ctx, actor)
	}
	return id
}

func actorOpt3StateBitForStatus(statusID uint16) (uint32, bool) {
	bit, ok := db.StatusOpt3State[statusID]
	if !ok {
		return 0, false
	}
	return bit, true
}

func (m *WorldMode) applyTrickDeadStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusTrickdead || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	actor, ok, _ := scene.ActorForCombatID(ctx, id)
	if !ok {
		return
	}
	now := time.Now()
	if change.Active {
		actionFamily := deathActionFamilyForActor(actor)
		m.setTrickDeadStatusAction(ctx, id, actionFamily, now, m.actorActionDuration(ctx, actor, actionFamily, scene.DefaultDeathAnimationDuration), true)
		glog.Debugf("trick dead active actor=%d", id)
		return
	}
	m.setTrickDeadStatusAction(ctx, id, scene.SpriteActionIdle, now, scene.DefaultAttackAnimationDuration, false)
	glog.Debugf("trick dead inactive actor=%d", id)
}

func (m *WorldMode) setTrickDeadStatusAction(ctx client.Context, id uint32, actionFamily int, started time.Time, duration time.Duration, holdFinal bool) {
	m.setCombatActorAction(ctx, id, actorAnimation{
		actionFamily: actionFamily,
		started:      started,
		duration:     duration,
		play:         true,
		hasPlay:      true,
		holdFinal:    holdFinal,
	})
}


func (m *WorldMode) removeExpiredStatusEffects(s *session.Session, now time.Time) {
	if s == nil {
		return
	}
	for id, effect := range s.Statuses.Active {
		if effect.HasDuration && !effect.ExpiresAt.IsZero() && now.After(effect.ExpiresAt) {
			delete(s.Statuses.Active, id)
			if id == db.StatusCashBossAlarm {
				m.ui.minimap.ClearBossMarker()
			}
		}
	}
}

func localActorHasStatus(ctx client.Context, statusID uint16) bool {
	if ctx.Session == nil || ctx.Session.Statuses.Active == nil {
		return false
	}
	_, ok := ctx.Session.Statuses.Active[statusID]
	return ok
}

func localActorHidden(ctx client.Context) bool {
	return localActorHasStatus(ctx, db.StatusHiding)
}
