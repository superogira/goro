package game

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) applyCompanionLife(ctx client.Context, id uint32, hp, maxHP, sp, maxSP, attackRange int) {
	if id == 0 || ctx.World == nil {
		return
	}
	if actor, ok := ctx.World.Actors[id]; ok {
		if attackRange > 0 {
			actor.AttackRange = attackRange
		}
		ctx.World.Actors[id] = actor
	}
	if maxHP <= 0 && maxSP <= 0 {
		return
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	life := m.actorLife[id]
	if maxHP > 0 {
		life.hp = scene.ClampInt(hp, 0, maxHP)
		life.maxHP = maxHP
	}
	if maxSP > 0 {
		life.sp = scene.ClampInt(sp, 0, maxSP)
		life.maxSP = maxSP
		life.hasSP = true
	}
	m.actorLife[id] = life
}

func (m *WorldMode) applyCompanionHunger(ctx client.Context, id uint32, hunger, maxHunger int) {
	if id == 0 || ctx.World == nil || maxHunger <= 0 {
		return
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	life := m.actorLife[id]
	life.hunger = scene.ClampInt(hunger, 0, maxHunger)
	life.maxHunger = maxHunger
	life.hasHunger = true
	m.actorLife[id] = life
}

func (m *WorldMode) handleCompanionAICommandClick(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || ctx.World == nil || ctx.Session == nil || m.mapPointerBlocked(ctx) || !ctx.Input.Pressed(input.KeyAlt) {
		return false
	}
	kind := ""
	id := uint32(0)
	if ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		kind = "mercenary"
		id = ctx.Session.Mercenary.ID
	} else if ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		kind = "homunculus"
		id = ctx.Session.Homunculus.ID
	} else {
		return false
	}
	if id == 0 {
		return true
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	if actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths); ok {
		m.companions().SetMessage(id, fmt.Sprintf("3,%d", actor.ID))
		glog.Debugf("%s AI command attack companion=%d target=%d", kind, id, actor.ID)
		return true
	}
	if x, y, ok := maps.ClickedWalkTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY); ok {
		m.companions().SetMessage(id, fmt.Sprintf("1,%d,%d", x, y))
		glog.Debugf("%s AI command move companion=%d target=%d,%d", kind, id, x, y)
		return true
	}
	return true
}

func (m *WorldMode) companionLife(ctx client.Context, id uint32) (hp, maxHP, sp, maxSP int, ok bool) {
	if hp, maxHP, sp, maxSP, ok := companion.Life(ctx, id); ok {
		return hp, maxHP, sp, maxSP, true
	}
	if m.actorLife != nil {
		if life, found := m.actorLife[id]; found {
			return life.hp, life.maxHP, life.sp, life.maxSP, true
		}
	}
	return 0, 0, 0, 0, false
}

func (m *WorldMode) openHomunculusContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || ctx.Input.Pressed(input.KeyAlt) || m.mapPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedHomunculusTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	m.companions().OpenHomunculusContext(ctx, actor)
	return true
}

func clickedHomunculusTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	expectedID := uint32(0)
	if ctx.Session != nil {
		expectedID = ctx.Session.Homunculus.ID
	}
	if expectedID == 0 {
		expectedID = companion.FindActorID(ctx, scene.ActorObjectTypeHomunculus)
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypeHomunculus {
		return worldstate.Actor{}, false
	}
	if expectedID != 0 && actor.ID != expectedID {
		return worldstate.Actor{}, false
	}
	return actor, true
}

func (m *WorldMode) openMercenaryContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || ctx.Input.Pressed(input.KeyAlt) || m.mapPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedMercenaryTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	m.companions().OpenMercenaryContext(ctx, actor)
	return true
}

func clickedMercenaryTarget(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	expectedID := uint32(0)
	if ctx.Session != nil {
		expectedID = ctx.Session.Mercenary.ID
	}
	if expectedID == 0 {
		expectedID = companion.FindActorID(ctx, scene.ActorObjectTypeMercenary)
	}
	actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, deadActors)
	if !ok || actor.ID == 0 || !actor.HasObjectType || actor.ObjectType != scene.ActorObjectTypeMercenary {
		return worldstate.Actor{}, false
	}
	if expectedID != 0 && actor.ID != expectedID {
		return worldstate.Actor{}, false
	}
	return actor, true
}

func (m *WorldMode) companions() *companion.System {
	m.companionSystem.Bind(companion.Bindings{
		Windows:   &m.ui.companions,
		SetLife:   m.applyCompanionLife,
		SetHunger: m.applyCompanionHunger,
		LevelUp:   func(ctx client.Context, id uint32) { m.effects().AddIfMissing(ctx, effects.HoUp, id) },
		SnapshotLife: func() map[uint32]companion.Vitals {
			life := make(map[uint32]companion.Vitals, len(m.actorLife))
			for id, value := range m.actorLife {
				life[id] = companion.Vitals{HP: value.hp, MaxHP: value.maxHP, SP: value.sp, MaxSP: value.maxSP}
			}
			return life
		},
		Error:   m.ui.console.AddErrorMessage,
		Actions: m,
	})
	return &m.companionSystem
}
