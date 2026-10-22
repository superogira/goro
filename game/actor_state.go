package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) applyActorStateChange(ctx client.Context, change network.ActorStateChange) {
	oldVisualJob := localPlayerVisualJob(ctx)
	oldState, local, ok := applyActorStateSnapshot(ctx, change)
	if !ok {
		return
	}
	m.applyActorEffectStateEffects(ctx, change.ID, oldState, change.EffectState)
	m.syncActorSpiritEffects(ctx, change.ID)
	if (oldState^change.EffectState)&db.EffectStateHide != 0 {
		m.effects().Add(ctx, effects.SummonSlave, change.ID)
		actor, _, _ := scene.ActorForCombatID(ctx, change.ID)
		if change.EffectState&db.EffectStateHide != 0 && actor.Job != db.JobNinja {
			m.effects().Add(ctx, effects.BashBegin, change.ID)
		}
	}
	if change.EffectState&scene.ActorStealthMask != 0 {
		m.effects().RemoveLevel99AuraEffects(change.ID)
		delete(m.speechBubbles, change.ID)
		if local {
			m.effects().RemoveLevel99AuraEffects(scene.LocalSkillTarget(ctx))
			delete(m.speechBubbles, scene.LocalSkillTarget(ctx))
		}
	}
	if local {
		if !localStealthAllowsSkill(ctx, 0) {
			m.cancelAttackIntent()
		}
		if ctx.PlayerHasEffectState(db.EffectStateHide) {
			m.pendingPickup = pickupIntent{}
		}
		if !localStealthAllowsSkill(ctx, m.pendingSkill.skill.ID) {
			m.pendingSkill = pendingSkillTarget{}
		}
		if newVisualJob := localPlayerVisualJob(ctx); newVisualJob != oldVisualJob {
			m.reloadPlayerSpriteView(ctx, fmt.Sprintf("state effect=0x%08X", change.EffectState))
		}
		glog.Debugf("actor state local id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
		return
	}
	if change.EffectState&scene.ActorStealthMask != 0 {
		if m.pendingAttack.targetID == change.ID || m.lockedAttackID == change.ID {
			m.cancelAttackIntent()
		}
		if m.pendingSkill.targetID == change.ID {
			m.pendingSkill = pendingSkillTarget{}
		}
	}
	glog.Debugf("actor state id=%d body=%d health=0x%04X effect=0x%08X", change.ID, change.BodyState, change.HealthState, change.EffectState)
}

// applyActorStateSnapshot updates state shared by the login and world modes.
// Rendering transitions remain the responsibility of WorldMode.
func applyActorStateSnapshot(ctx client.Context, change network.ActorStateChange) (oldState uint32, local, ok bool) {
	if change.ID == 0 || ctx.World == nil {
		return 0, false, false
	}
	if scene.IsLocalActor(ctx, change.ID) {
		oldState = ctx.World.Player.EffectState
		setActorRenderState(&ctx.World.Player, change.BodyState, change.HealthState, change.EffectState)
		setSelectedCharacterOptionBit(ctx, db.EffectStateWedding, change.EffectState&db.EffectStateWedding != 0)
		return oldState, true, true
	}
	actor, ok := ctx.World.Actors[change.ID]
	if !ok {
		return 0, false, false
	}
	oldState = actor.EffectState
	setActorRenderState(&actor, change.BodyState, change.HealthState, change.EffectState)
	upsertActor(ctx, actor)
	return oldState, false, true
}

func (m *WorldMode) applyActorBladeStop(ctx client.Context, blade network.ActorBladeStop) {
	if blade.SourceID == 0 || blade.TargetID == 0 || ctx.World == nil {
		return
	}
	m.applyActorBladeStopSide(ctx, blade.SourceID, blade.TargetID, blade.Active, true)
	m.applyActorBladeStopSide(ctx, blade.TargetID, blade.SourceID, blade.Active, false)
	glog.Debugf("actor blade stop src=%d target=%d active=%t", blade.SourceID, blade.TargetID, blade.Active)
}

func (m *WorldMode) applyActorBladeStopSide(ctx client.Context, actorID, lookID uint32, active, caster bool) {
	actor, ok, local := scene.ActorForCombatID(ctx, actorID)
	if !ok {
		return
	}
	now := time.Now()
	anim := actorAnimation{actionFamily: scene.SpriteActionIdle, started: now, play: true, hasPlay: true}
	if active {
		// Freeze the victim's current frame, before stopping movement or turning.
		state := m.nonPCSpriteState(actor, now)
		if actor.Sitting {
			state.ActionFamily = scene.SpriteActionSit
		}
		started := now
		if res.HasPlayerJobToken(scene.ActorVisualJob(actor)) || companion.IsMercenary(actor) {
			state.LoopIdle = false
			state.HeadTurn, state.HeadDir = true, actor.HeadDir
			if view := m.humanoidSpriteViewForActor(ctx, actor); view != nil {
				started = view.Started
			}
		} else if view := m.nonPCSpriteView(ctx, actor); view != nil {
			started = view.Started
		}
		anim.actionFamily = state.ActionFamily
		anim.fixedMotion = state.FixedMotion
		renderActor := actor
		renderActor.Dir = sprites.DirectionForCamera(actor.Dir, scene.CameraYawForMap(ctx))
		if action, ok := m.actorResolvedAction(ctx, renderActor, state.ActionFamily); ok {
			anim.fixedMotion = sprites.BodyMotionForState(action, state, started, now)
		}
		if caster && res.HasPlayerJobToken(scene.ActorVisualJob(actor)) {
			anim.actionFamily = sprites.ActionPCSkill
			anim.fixedMotion = 4
		}
		anim.play = false
		anim.hasFixedMotion = true
		anim.holdFinal = true
		actor.Opt3State |= db.Opt3Bladestop
	} else {
		actor.Opt3State &^= db.Opt3Bladestop
	}
	target, targetOK, _ := scene.ActorForCombatID(ctx, lookID)
	if targetOK {
		actor.Dir = directionFromDelta(actor.X, actor.Y, target.X, target.Y, actor.Dir)
	}
	if local {
		ctx.World.Player.Opt3State = actor.Opt3State
		ctx.World.Player.Dir = actor.Dir
		ctx.World.Dir = actor.Dir
	} else {
		ctx.World.Actors[actor.ID] = actor
	}
	if active {
		m.stopActorMovementAt(ctx, actorID, now)
	}
	m.setCombatActorAction(ctx, actorID, anim)
}

func setActorRenderState(actor *worldstate.Actor, bodyState, healthState uint16, effectState uint32) {
	actor.BodyState = bodyState
	actor.HealthState = healthState
	actor.EffectState = effectState
	actor.HasState = true
	applyActorCartStateFromEffect(actor)
}

func applyActorBodyState(actor worldstate.Actor, state *sprites.State) {
	switch actor.BodyState {
	case db.BodyStateFreeze:
		state.ActionFamily = freezeActionFamily(actor)
		state.Moving = false
		state.Loop = false
		state.LoopIdle = false
		state.Play = false
		state.HasPlay = true
		state.FixedMotion = 0
		state.HasFixedMotion = true
	case db.BodyStateStone:
		state.Moving = false
		state.Loop = false
		state.LoopIdle = false
		state.Play = false
		state.HasPlay = true
	}
}

func actorStateTint(actor worldstate.Actor) color.RGBA {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	switch actor.BodyState {
	case db.BodyStateStone:
		r, g, b = 0.1, 0.1, 0.1
	case db.BodyStateStonewait:
		r, g, b = 0.3, 0.3, 0.3
	case db.BodyStateFreeze:
		r, g, b = 0.0, 0.4, 0.8
	}
	if actor.HealthState&db.HealthStateCurse != 0 {
		r *= 0.5
		g *= 0.15
		b *= 0.1
	}
	if actor.HealthState&db.HealthStatePoison != 0 {
		r *= 0.9
		g *= 0.4
		b *= 0.8
	}
	if actor.HealthState&db.HealthStateBlind != 0 {
		r *= 0.2
		g *= 0.2
		b *= 0.2
	}
	opt3 := actor.Opt3State
	if scene.IsMonsterLikeHoverActor(actor) {
		opt3 &^= db.Opt3Energycoat | db.Opt3Overthrust // Monsters use these bits for Eske and Eswoo.
	}
	vr, vg, vb, va := actorOpt3StateTint(opt3)
	r *= vr
	g *= vg
	b *= vb
	a *= va
	return color.RGBA{R: byte(scene.ClampUnit(r) * 255), G: byte(scene.ClampUnit(g) * 255), B: byte(scene.ClampUnit(b) * 255), A: byte(scene.ClampUnit(a) * 255)}
}

func actorOpt3StateTint(opt3State uint32) (float64, float64, float64, float64) {
	r, g, b, a := 1.0, 1.0, 1.0, 1.0
	if opt3State&db.Opt3Quicken != 0 {
		b = 0
	}
	if opt3State&db.Opt3Energycoat != 0 {
		r, g, b = 0.5, 0.5, 0.85
	}
	if opt3State&db.Opt3Bunsin != 0 {
		r, g, b = 155.0/255, 155.0/255, 1
	}
	if opt3State&db.Opt3Overthrust != 0 {
		r, g, b = 250.0/255, 150.0/255, 150.0/255
	}
	if opt3State&db.Opt3Bladestop != 0 {
		r, g, b = 64.0/255, 64.0/255, 64.0/255
	}
	return r, g, b, a
}

func (m *WorldMode) actorRenderTint(actor worldstate.Actor, now time.Time) color.RGBA {
	return m.actorBodyColorTint(actor.ID, actorStateTint(actor), now)
}

func (m *WorldMode) playerRenderTint(ctx client.Context, actor worldstate.Actor, now time.Time) color.RGBA {
	tint := m.actorRenderTint(actor, now)
	if ctx.Session == nil {
		return tint
	}
	if ctx.Session.AccountID != actor.ID {
		tint = m.actorBodyColorTint(ctx.Session.AccountID, tint, now)
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		tint = m.actorBodyColorTint(ctx.Session.CharID, tint, now)
	}
	return tint
}

func (m *WorldMode) actorRenderBlend(actorID uint32, now time.Time) render.Blend {
	if actorID != 0 {
		for _, effect := range m.effects().Active {
			if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
				continue
			}
			elapsed := now.Sub(effect.Starts)
			if effects.SniperBodyFlashActive(effect.EffectID, int(elapsed/db.EffectFrameDuration)) {
				return render.BlendLighter // BL_LIGHT_BODY selects RF_EFFECT in the original client.
			}
		}
	}
	return render.BlendSourceOver
}

func (m *WorldMode) playerRenderBlend(ctx client.Context, actor worldstate.Actor, now time.Time) render.Blend {
	if ctx.Session != nil {
		for _, id := range []uint32{ctx.Session.AccountID, ctx.Session.CharID} {
			if blend := m.actorRenderBlend(id, now); blend != render.BlendSourceOver {
				return blend
			}
		}
	}
	return m.actorRenderBlend(actor.ID, now)
}

func (m *WorldMode) actorBodyColorTint(actorID uint32, tint color.RGBA, now time.Time) color.RGBA {
	if actorID == 0 {
		return tint
	}
	base := tint
	for _, effect := range m.effects().Active {
		if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		switch effect.EffectID {
		case effects.Portal4:
			if effects.SniperBodyFlashActive(effect.EffectID, int(now.Sub(effect.Starts)/db.EffectFrameDuration)) {
				tint = scene.MultiplyTint(base, 220.0/255, 250.0/255, 220.0/255, 1)
			}
		case effects.Shrink:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame >= 5 && frame < 26 && frame%2 == 0 {
				tint = scene.MultiplyTint(base, 250.0/255, 250.0/255, 100.0/255, 1)
			}
		case effects.Bash3D:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame >= 20 && frame <= 40 {
				tint = scene.MultiplyTint(base, 1, 200.0/255, 200.0/255, 1)
			}
		case effects.BodyColor:
			tint = effectBodyColorTint(tint, now.Sub(effect.Starts))
		case effects.MagicCrasher2:
			tint = magicCrasher2BodyColorTint(tint, actorID, effect.Starts, now.Sub(effect.Starts))
		case effects.MagicCrasher:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if effects.MagicCrasherFlashActive(now.Sub(effect.Starts)) {
				seed := uint64(actorID)*0x9e3779b185ebca87 ^ uint64(effect.Starts.UnixNano()) ^ uint64(frame)*0xc2b2ae3d27d4eb4f
				tint = scene.MultiplyTint(base, randomBodyColorChannel(seed), randomBodyColorChannel(seed+1), randomBodyColorChannel(seed+2), 1)
			}
		case effects.TransBlueBody:
			frame := now.Sub(effect.Starts) / db.EffectFrameDuration
			if frame < 200 {
				channel := float64(205-frame) / 255
				tint = scene.MultiplyTint(base, channel, channel, 1, 1)
			}
		}
	}
	return tint
}

func effectBodyColorTint(tint color.RGBA, elapsed time.Duration) color.RGBA {
	alpha := scene.ClampFloat(float64(elapsed)/float64(100*time.Millisecond), 0, 1)
	return color.RGBA{
		R: mixTintChannel(tint.R, 255, alpha),
		G: mixTintChannel(tint.G, 0, alpha),
		B: mixTintChannel(tint.B, 0, alpha),
		A: tint.A,
	}
}

func magicCrasher2BodyColorTint(tint color.RGBA, actorID uint32, starts time.Time, elapsed time.Duration) color.RGBA {
	if elapsed < 0 || elapsed/db.EffectFrameDuration > 60 {
		return tint
	}
	bucket := uint64(elapsed / db.EffectFrameDuration)
	seed := uint64(actorID)*0x9e3779b185ebca87 ^ uint64(starts.UnixNano()) ^ bucket*0xc2b2ae3d27d4eb4f
	return scene.MultiplyTint(tint, randomBodyColorChannel(seed), randomBodyColorChannel(seed+1), randomBodyColorChannel(seed+2), 1)
}

func randomBodyColorChannel(seed uint64) float64 {
	seed ^= seed >> 33
	seed *= 0xff51afd7ed558ccd
	seed ^= seed >> 33
	seed *= 0xc4ceb9fe1a85ec53
	seed ^= seed >> 33
	return float64(byte(seed>>56)) / 255
}

func mixTintChannel(from, to uint8, alpha float64) uint8 {
	value := float64(from)*(1-alpha) + float64(to)*alpha
	return uint8(scene.ClampFloat(value, 0, 255))
}

func (m *WorldMode) actorRenderScale(actorID uint32, baseScale float64, now time.Time) float64 {
	return baseScale * m.actorBodySizeMultiplier(actorID, now)
}

func (m *WorldMode) playerRenderScale(ctx client.Context, actor worldstate.Actor, baseScale float64, now time.Time) float64 {
	if scale := m.playerBodyRenderScale(actor.ID, actor.Job, baseScale, now); scale != baseScale {
		return scale
	}
	if ctx.Session == nil {
		return baseScale
	}
	if ctx.Session.AccountID != actor.ID {
		if scale := m.playerBodyRenderScale(ctx.Session.AccountID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	if ctx.Session.CharID != actor.ID && ctx.Session.CharID != ctx.Session.AccountID {
		if scale := m.playerBodyRenderScale(ctx.Session.CharID, actor.Job, baseScale, now); scale != baseScale {
			return scale
		}
	}
	return baseScale
}

func (m *WorldMode) playerBodyRenderScale(actorID uint32, job int16, baseScale float64, now time.Time) float64 {
	bodyScale := playerBodyScaleForJob(job)
	multiplier := m.actorBodySizeMultiplierFrom(actorID, now, bodyScale)
	if multiplier == bodyScale {
		return baseScale
	}
	return baseScale / bodyScale * multiplier
}

func (m *WorldMode) actorBodySizeMultiplier(actorID uint32, now time.Time) float64 {
	return m.actorBodySizeMultiplierFrom(actorID, now, 1)
}

func (m *WorldMode) actorBodySizeMultiplierFrom(actorID uint32, now time.Time, baseMultiplier float64) float64 {
	if actorID == 0 {
		return baseMultiplier
	}
	multiplier := baseMultiplier
	var latest time.Time
	for _, effect := range m.effects().Active {
		if effect.ActorID != actorID || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		if effect.Starts.Before(latest) {
			continue
		}
		if effect.EffectID == effects.BabyBodyBack || (effect.EffectID == effects.BabyBody && effect.Persistent) {
			frame := float64(now.Sub(effect.Starts) / db.EffectFrameDuration)
			if effect.EffectID == effects.BabyBodyBack {
				frame = 45 - frame*3
			}
			multiplier = baseMultiplier * (.75 + .25*math.Cos(scene.DegreesToRadians(scene.ClampFloat(frame, 0, 45)*4)))
			latest = effect.Starts
			continue
		}
		targetSize, transition, ok := actorBodySizeTarget(effect.EffectID)
		if !ok {
			continue
		}
		targetMultiplier := targetSize / 5
		if transition {
			duration := effect.Duration
			if duration <= 0 {
				duration = 300 * time.Millisecond
			}
			progress := scene.ClampFloat(float64(now.Sub(effect.Starts))/float64(duration), 0, 1)
			targetMultiplier = baseMultiplier + (targetMultiplier-baseMultiplier)*progress
		}
		multiplier = targetMultiplier
		latest = effect.Starts
	}
	return multiplier
}

func actorBodySizeTarget(effectID int) (targetSize float64, transition bool, ok bool) {
	switch effectID {
	case effects.BabyBody:
		return 2.5, true, true
	case effects.BabyBody2:
		return 2.5, false, true
	case effects.GiantBody:
		return 7.5, true, true
	case effects.GiantBody2:
		return 7.5, false, true
	default:
		return 0, false, false
	}
}

func freezeActionFamily(actor worldstate.Actor) int {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return sprites.ActionPCFreeze2
	}
	return scene.SpriteActionIdle
}
