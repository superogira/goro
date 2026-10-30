package actors

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/world"
)

func (m *System) HumanoidSpriteViewForActor(ctx client.Context, actor world.Actor) *sprites.Humanoid {
	if scene.IsLocalActor(ctx, actor.ID) {
		return m.PlayerView
	}
	if companion.IsMercenary(actor) {
		return m.MercenaryHumanoidSpriteView(ctx, actor)
	}
	actor = actorWithVisualJob(actor)
	weapon, shield := res.NormalizePlayerWeaponShield(int(actor.Weapon), int(actor.Shield))
	key := sprites.Appearance{
		Job:         int(actor.Job),
		Head:        int(actor.Head),
		Sex:         actor.Sex,
		Admin:       actor.IsAdmin,
		BodyPalette: int(actor.BodyPal),
		HeadPalette: int(actor.HeadPal),
		Weapon:      weapon,
		Shield:      shield,
		HeadTop:     int(actor.HeadTop),
		HeadMid:     int(actor.HeadMid),
		HeadLow:     int(actor.HeadLow),
	}
	return m.ActorViews[key]
}

func (m *System) nonPCResolvedAction(ctx client.Context, actor world.Actor, actionFamily int) (res.ACTAction, bool) {
	view := m.NonPCSpriteView(ctx, actor)
	if view == nil {
		return res.ACTAction{}, false
	}
	_, action, ok := sprites.ResolveAction(view.Act, actionFamily, actor.Dir)
	return action, ok
}

func (m *System) ActorResolvedAction(ctx client.Context, actor world.Actor, actionFamily int) (res.ACTAction, bool) {
	if res.HasPlayerJobToken(scene.ActorVisualJob(actor)) || companion.IsMercenary(actor) {
		view := m.HumanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.Body == nil {
			return res.ACTAction{}, false
		}
		_, action, ok := sprites.ResolveAction(view.Body.Act, actionFamily, actor.Dir)
		return action, ok
	}
	return m.nonPCResolvedAction(ctx, actor, actionFamily)
}

func (m *System) ActorActionFrameDelay(ctx client.Context, actor world.Actor, actionFamily int, duration time.Duration) time.Duration {
	if duration <= 0 {
		return 0
	}
	action, ok := m.ActorResolvedAction(ctx, actor, actionFamily)
	if !ok || len(action.Animations) == 0 {
		return 0
	}
	return duration / time.Duration(len(action.Animations))
}

func (m *System) nonPCActionACT(ctx client.Context, actor world.Actor) *res.ACT {
	view := m.NonPCSpriteView(ctx, actor)
	if view == nil {
		return nil
	}
	return view.Act
}

func (m *System) ActorActionACT(ctx client.Context, actor world.Actor) *res.ACT {
	if res.HasPlayerJobToken(scene.ActorVisualJob(actor)) || companion.IsMercenary(actor) {
		view := m.HumanoidSpriteViewForActor(ctx, actor)
		if view == nil || view.Body == nil {
			return nil
		}
		return view.Body.Act
	}
	return m.nonPCActionACT(ctx, actor)
}

func (m *System) ActorActionDuration(ctx client.Context, actor world.Actor, actionFamily int, fallback time.Duration) time.Duration {
	if !res.HasPlayerJobToken(int(actor.Job)) && !companion.IsMercenary(actor) && m.nonPCActorHasGR2Model(ctx, actor) {
		if action, ok := gr2ActionForActionFamily(actionFamily); ok {
			if view := m.NonPCGR2ModelView(ctx, actor); view != nil {
				if duration := view.actionDuration(action); duration > 0 {
					return duration
				}
			}
		}
	}
	if action, ok := m.ActorResolvedAction(ctx, actor, actionFamily); ok {
		return scene.ActionAnimationDuration(action, fallback)
	}
	return fallback
}

func (m *System) ApplyComboDelay(ctx client.Context, notify network.ActorComboDelay, now time.Time) {
	_, ok, local := scene.ActorForCombatID(ctx, notify.ID)
	if !ok {
		return
	}
	ids := []uint32{notify.ID}
	if local {
		ids = []uint32{ctx.Session.AccountID, ctx.Session.CharID}
	}
	if m.ComboDelays == nil {
		m.ComboDelays = make(map[uint32]time.Duration)
	}
	delay := time.Duration(notify.DelayTime) * time.Millisecond
	for _, id := range ids {
		if id == 0 {
			continue
		}
		// The server normally sends this before the skill result; retain it
		// for the next combo pose and update an existing one without restarting it.
		m.ComboDelays[id] = delay
		if anim, ok := m.ActorAnims[id]; ok && anim.Combo {
			anim.Duration = now.Add(delay).Sub(anim.Started)
			m.ActorAnims[id] = anim
		}
	}
}

func (m *System) ActorAnimation(id uint32, now time.Time) (scene.ActorAnimation, bool) {
	if m.ActorAnims == nil || id == 0 {
		return scene.ActorAnimation{}, false
	}
	anim, ok := m.ActorAnims[id]
	if !ok {
		return scene.ActorAnimation{}, false
	}
	if anim.Duration <= 0 && !anim.Combo {
		anim.Duration = scene.DefaultAttackAnimationDuration
	}
	if now.Before(anim.Started) {
		return scene.ActorAnimation{}, false
	}
	if anim.Loop && anim.Next == nil {
		return anim, true
	}
	if !now.Before(anim.Started.Add(anim.Duration)) {
		if anim.HoldFinal {
			return anim, true
		}
		if anim.Next != nil {
			next := *anim.Next
			if next.Started.IsZero() {
				next.Started = anim.Started.Add(anim.Duration).Add(next.StartDelay)
				next.StartDelay = 0
			}
			m.ActorAnims[id] = next
			return m.ActorAnimation(id, now)
		}
		delete(m.ActorAnims, id)
		return scene.ActorAnimation{}, false
	}
	return anim, true
}

func DeathActionFamilyForActor(actor world.Actor) int {
	if companion.IsMercenary(actor) {
		if companion.MercenaryUsesArcherActions(actor.Job) {
			return sprites.ActionNonPCDeath
		}
		return sprites.ActionPCDeath
	}
	if res.HasPlayerJobToken(int(actor.Job)) {
		return sprites.ActionPCDeath
	}
	return sprites.ActionNonPCDeath
}

func (m *System) ActorDeathAlpha(id uint32, now time.Time) float64 {
	removeAt, ok := m.ActorDeaths[id]
	if !ok {
		return 1
	}
	if removeAt.IsZero() {
		return 1
	}
	started := now
	if anim, ok := m.ActorAnims[id]; ok && !anim.Started.IsZero() {
		started = anim.Started
	}
	total := removeAt.Sub(started)

	if total <= 0 {
		return 0
	}
	elapsed := now.Sub(started)
	if elapsed <= 0 {
		return 1
	}
	alpha := 1 - float64(elapsed)/float64(total)
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func (m *System) actorVisualAlpha(id uint32, now time.Time) float64 {
	return math.Min(m.ActorDeathAlpha(id, now), m.ActorVanishAlpha(id, now))
}

func (m *System) ActorVanishAlpha(id uint32, now time.Time) float64 {
	fade, ok := m.ActorVanishes[id]
	if !ok {
		return 1
	}
	total := fade.RemoveAt.Sub(fade.Started)
	if total <= 0 {
		return 0
	}
	elapsed := now.Sub(fade.Started)
	if elapsed <= 0 {
		return 1
	}
	alpha := 1 - float64(elapsed)/float64(total)
	if alpha < 0 {
		return 0
	}
	if alpha > 1 {
		return 1
	}
	return alpha
}

func (m *System) SkillEffectDirection(ctx client.Context, actorID uint32, direction int, now time.Time) int {
	for _, effect := range m.bindings.Effects.Active {
		if (effect.EffectID != effects.FalconAssault && effect.EffectID != effects.CastSpin && effect.EffectID != effects.StormKick) || now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		if effect.ActorID != actorID && !(scene.IsLocalActor(ctx, actorID) && scene.IsLocalActor(ctx, effect.ActorID)) {
			continue
		}
		frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
		turn := 0.0
		switch effect.EffectID {
		case effects.StormKick:
			if frame > 47 {
				continue
			}
			turn = float64(frame+1) * 30
		case effects.FalconAssault:
			if frame < 30 || frame > 54 {
				continue
			}
			turn = float64(frame-29) * 30
		case effects.CastSpin:
			anim, ok := m.ActorAnims[effect.ActorID]
			if frame >= 100 || !ok || anim.ActionFamily != scene.SpriteActionIdle || !anim.Started.Equal(effect.Starts) || now.Sub(anim.Started) >= anim.Duration {
				continue
			}
			ticks := frame + 1
			turn = float64(12*min(ticks, 16) + 18*scene.ClampInt(ticks-16, 0, 15) + 24*scene.ClampInt(ticks-31, 0, 10) + 48*max(0, ticks-41))
		default:
			continue
		}
		direction = (direction + int(math.Floor((turn+22.5)/45))) % 8
	}
	return direction
}
