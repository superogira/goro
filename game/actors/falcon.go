package actors

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

const (
	actorPersistentEffectOptionMask = ActorEffectCartMask | db.EffectStateFalcon | db.EffectStateWedding
	falconGlideHeight               = 5.0
	falconFollowRange               = 2
	falconStopRange                 = 1
	FalconRetargetInterval          = time.Second
	FalconAttackMoveSpeedMS         = 35
	FalconAttackReturnDelay         = 432 * time.Millisecond
	falconAttackOvershoot           = 5
)

type FalconRenderState struct {
	X            float64
	Y            float64
	Direction    int
	Path         []worldstate.WalkStep
	MoveStarted  time.Time
	MoveDuration time.Duration
	MoveSpeedMS  int
	MoveStartX   float64
	MoveStartY   float64
	Moving       bool
	HasTarget    bool
	TargetX      int
	TargetY      int
	LastWalkTick time.Time
	Attacking    bool
	ReturnAt     time.Time
}

func actorHasFalcon(actor worldstate.Actor) bool {
	if actor.EffectState&db.EffectStateFalcon == 0 {
		return false
	}
	return actorJobSupportsFalconSprite(int(actor.Job))
}

func actorJobSupportsFalconSprite(job int) bool {
	switch job {
	case db.JobHunter, db.JobHunterH, db.JobHunterB,
		db.JobRanger, db.JobRanger2, 4098,
		db.JobWindhawk, 4270, 4278:
		return true
	default:
		return false
	}
}

func falconMoveSpeedMS(ownerSpeed int) int {
	if ownerSpeed <= 0 {
		return 100
	}
	speed := ownerSpeed - 50
	if speed < 1 {
		return 1
	}
	return speed
}

func falconSkillAttacksTarget(skillID uint16) bool {
	switch skillID {
	case db.SkillHTBlitzbeat, db.SkillSNFalconassault:
		return true
	default:
		return false
	}
}

func falconSkillAttacksGround(skillID uint16) bool {
	return skillID == db.SkillHTDetecting
}

func (m *System) ApplyFalconStatus(ctx client.Context, change network.StatusEffectChange) {
	if change.StatusID != db.StatusFalcon || ctx.World == nil {
		return
	}
	id := change.ActorID
	if id == 0 {
		id = scene.LocalSkillTarget(ctx)
	}
	actor, ok, local := scene.ActorForCombatID(ctx, id)
	if !ok {
		return
	}
	if change.Active {
		actor.EffectState |= db.EffectStateFalcon
	} else {
		actor.EffectState &^= db.EffectStateFalcon
	}
	actor.HasState = true
	if local {
		ctx.World.Player.EffectState = actor.EffectState
		ctx.World.Player.HasState = true
		setSelectedCharacterOptionBit(ctx, db.EffectStateFalcon, change.Active)
		if !change.Active {
			m.removeFalconState(id)
			if ctx.Session != nil {
				m.removeFalconState(ctx.Session.CharID)
			}
		}
		glog.Debugf("actor falcon status local actor=%d active=%t", id, change.Active)
		return
	}
	UpsertActor(ctx, actor)
	if !change.Active {
		m.removeFalconState(id)
	}
	glog.Debugf("actor falcon status actor=%d active=%t", id, change.Active)
}

func (m *System) ApplyFalconSkillNoDamageNotify(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	if !falconSkillAttacksTarget(notify.SkillID) {
		return
	}
	target, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID)
	if !ok {
		return
	}
	targetX, targetY := scene.ActorRenderPosition(target, now)
	m.StartFalconAttackAt(ctx, notify.SourceID, int(targetX), int(targetY), now)
}

func (m *System) ApplyFalconActorActionNotify(ctx client.Context, action network.ActorActionNotify, now time.Time) {
	if !falconSkillAttacksTarget(action.SkillID) {
		return
	}
	target, ok, _ := scene.ActorForCombatID(ctx, action.TargetID)
	if !ok {
		return
	}
	targetX, targetY := scene.ActorRenderPosition(target, now)
	m.StartFalconAttackAt(ctx, action.SourceID, int(targetX), int(targetY), now)
}

func (m *System) ApplyFalconGroundSkillNotify(ctx client.Context, notify network.GroundSkillNotify, now time.Time) {
	if !falconSkillAttacksGround(notify.SkillID) {
		return
	}
	m.StartFalconAttackAt(ctx, notify.SourceID, int(notify.X), int(notify.Y), now)
}

func (m *System) StartFalconAttackAt(ctx client.Context, sourceID uint32, targetX, targetY int, now time.Time) bool {
	source, ok, local := scene.ActorForCombatID(ctx, sourceID)
	if !ok {
		return false
	}
	if local {
		source = actorWithSelectedPersistentEffectOptions(ctx, source)
		source.ID = localFalconOwnerID(ctx)
	}
	if source.ID == 0 || !actorHasFalcon(source) {
		return false
	}
	falcon := m.UpdateFalconState(source, now)
	if falcon == nil {
		return false
	}
	falcon.StartAttack(targetX, targetY, now)
	return true
}

func actorWithSelectedPersistentEffectOptions(ctx client.Context, actor worldstate.Actor) worldstate.Actor {
	if ctx.Session == nil {
		return actor
	}
	character := ctx.Session.SelectedCharacter()
	actor.Job = character.Job
	if actor.HasState {
		actor.EffectState = (actor.EffectState &^ actorPersistentEffectOptionMask) | (character.Option & actorPersistentEffectOptionMask)
	} else {
		actor.EffectState = character.Option
	}
	return actor
}

func localFalconOwnerID(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.CharID != 0 {
		return ctx.Session.CharID
	}
	return ctx.Session.AccountID
}

func setSelectedCharacterOptionBit(ctx client.Context, bit uint32, active bool) {
	if ctx.Session == nil || bit == 0 {
		return
	}
	if active {
		ctx.Session.Selected.Option |= bit
	} else {
		ctx.Session.Selected.Option &^= bit
	}
	id := ctx.Session.Selected.ID
	if id == 0 {
		id = ctx.Session.CharID
	}
	for i := range ctx.Session.Characters {
		if ctx.Session.Characters[i].ID != id {
			continue
		}
		if active {
			ctx.Session.Characters[i].Option |= bit
		} else {
			ctx.Session.Characters[i].Option &^= bit
		}
		return
	}
}

func (m *System) removeFalconState(ownerID uint32) {
	if m.Falcons == nil || ownerID == 0 {
		return
	}
	delete(m.Falcons, ownerID)
}

func (m *System) falconSpriteView(ctx client.Context, job int) *scene.SpriteView {
	if ctx.Resources == nil {
		return nil
	}
	if m.FalconViews == nil {
		m.FalconViews = make(map[int]*scene.SpriteView)
	}
	if m.FalconViewMiss == nil {
		m.FalconViewMiss = make(map[int]struct{})
	}
	if view, ok := m.FalconViews[job]; ok {
		return view
	}
	if _, miss := m.FalconViewMiss[job]; miss {
		return nil
	}
	view, status := sprites.LoadFalcon(ctx.Resources, job)
	if view == nil {
		m.FalconViewMiss[job] = struct{}{}
		glog.Warnf("falcon sprite unavailable job=%d: %s", job, status)
		return nil
	}
	m.FalconViews[job] = view
	if status != "" {
		glog.Debugf("falcon sprite resources job=%d %s", job, status)
	}
	return view
}

func (m *System) DrawSceneActorFalcons(screen *render.Frame, ctx client.Context, projection scene.Projection, entries []scene.ActorDrawEntry) {
	if len(entries) == 0 {
		m.pruneFalconStates(nil)
		return
	}
	now := time.Now()
	activeOwners := make(map[uint32]struct{})
	for _, entry := range entries {
		if !actorHasFalcon(entry.Actor) {
			continue
		}
		activeOwners[entry.Actor.ID] = struct{}{}
		alpha := 1.0
		if entry.Stealth == scene.StealthHidden {
			alpha = 0.35
		}
		alpha *= m.actorVisualAlpha(entry.Actor.ID, now)
		m.drawActorFalcon3D(screen, ctx, projection, entry.Actor, projection.CameraYaw, alpha, now)
	}
	m.pruneFalconStates(activeOwners)
}

func (m *System) pruneFalconStates(activeOwners map[uint32]struct{}) {
	if m.Falcons == nil {
		return
	}
	for ownerID := range m.Falcons {
		if _, ok := activeOwners[ownerID]; ok {
			continue
		}
		delete(m.Falcons, ownerID)
	}
}

func (m *System) drawActorFalcon3D(screen *render.Frame, ctx client.Context, projection scene.Projection, actor worldstate.Actor, cameraYaw float64, alpha float64, now time.Time) bool {
	if ctx.World == nil || !actorHasFalcon(actor) {
		return false
	}
	view := m.falconSpriteView(ctx, int(actor.Job))
	if view == nil {
		return false
	}
	falcon := m.UpdateFalconState(actor, now)
	if falcon == nil {
		return false
	}
	state := falcon.spriteState(cameraYaw)
	billboard, ok := sprites.BillboardForState(view, state, now)
	if !ok {
		return false
	}
	terrainZ := scene.TerrainHeightAt(ctx.World, falcon.X, falcon.Y)
	worldX, worldY := cellWorldAnchor(falcon.X, falcon.Y)
	worldZ := terrainZ + falconGlideHeight
	scale := scene.ActorBillboardScreenScale(projection, worldX, worldY, worldZ)
	shadow := ActorShadowFactor(ctx.World, falcon.X, falcon.Y)
	m.DrawActorSpriteBillboardTintAlpha3D(screen, projection, billboard, worldX, worldY, worldZ, scale, alpha, shadow, color.RGBA{R: 255, G: 255, B: 255, A: 255}, render.BlendSourceOver)
	return true
}

func (m *System) UpdateFalconState(actor worldstate.Actor, now time.Time) *FalconRenderState {
	if actor.ID == 0 {
		return nil
	}
	if m.Falcons == nil {
		m.Falcons = make(map[uint32]*FalconRenderState)
	}
	ownerX, ownerY := scene.ActorRenderPosition(actor, now)
	state := m.Falcons[actor.ID]
	if state == nil {
		targetX, targetY := falconFollowTargetCell(ownerX, ownerY)
		state = &FalconRenderState{
			X:           ownerX,
			Y:           ownerY,
			Direction:   actor.Dir,
			MoveSpeedMS: falconMoveSpeedMS(actor.Speed),
			HasTarget:   true,
			TargetX:     targetX,
			TargetY:     targetY,
		}
		m.Falcons[actor.ID] = state
		return state
	}
	state.Advance(now)
	if state.Attacking {
		if !state.ReturnAt.IsZero() && !now.Before(state.ReturnAt) {
			state.Attacking = false
			if state.HasTarget {
				state.startPathTo(state.TargetX, state.TargetY, 0, now)
			}
		}
		return state
	}
	if falconFollowDistance(ownerX, ownerY, state.X, state.Y) < falconFollowRange {
		return state
	}
	targetX, targetY := falconFollowTargetCell(ownerX, ownerY)
	targetChanged := !state.HasTarget || state.TargetX != targetX || state.TargetY != targetY
	if !targetChanged {
		return state
	}
	if !state.LastWalkTick.IsZero() && state.LastWalkTick.Add(FalconRetargetInterval).After(now) {
		return state
	}
	state.MoveSpeedMS = falconMoveSpeedMS(actor.Speed)
	state.startFollow(targetX, targetY, now)
	return state
}

func (state *FalconRenderState) Advance(now time.Time) {
	if state == nil || !state.Moving || len(state.Path) < 2 {
		return
	}
	elapsed := now.Sub(state.MoveStarted)
	if state.MoveDuration <= 0 || elapsed >= state.MoveDuration {
		last := state.Path[len(state.Path)-1]
		state.X = float64(last.X)
		state.Y = float64(last.Y)
		state.Moving = false
		state.Path = nil
		return
	}
	state.X, state.Y = scene.RenderPathPositionWithSpeed(state.Path, elapsed, state.MoveSpeedMS, state.MoveStartX, state.MoveStartY, true)
	fromX, fromY, toX, toY := scene.RenderPathSegmentWithSpeed(state.Path, elapsed, state.MoveSpeedMS, state.MoveStartX, state.MoveStartY, true)
	state.Direction = scene.DirectionFromFloatDelta(fromX, fromY, toX, toY, state.Direction)
}

func (state *FalconRenderState) startFollow(targetX, targetY int, now time.Time) {
	if state == nil {
		return
	}
	state.HasTarget = true
	state.TargetX = targetX
	state.TargetY = targetY
	state.LastWalkTick = now
	state.startPathTo(targetX, targetY, falconStopRange, now)
}

func (state *FalconRenderState) StartAttack(targetX, targetY int, now time.Time) {
	if state == nil {
		return
	}
	state.Advance(now)
	overX, overY := falconOvershootTarget(state.X, state.Y, targetX, targetY)
	state.Attacking = true
	state.ReturnAt = now.Add(FalconAttackReturnDelay)
	state.MoveSpeedMS = FalconAttackMoveSpeedMS
	state.startPathTo(overX, overY, 0, now)
}

func (state *FalconRenderState) startPathTo(targetX, targetY int, stopRange int, now time.Time) {
	if state == nil {
		return
	}
	fromX, fromY := falconFollowTargetCell(state.X, state.Y)
	path := falconFollowPath(fromX, fromY, targetX, targetY, stopRange)
	if len(path) < 2 {
		state.Moving = false
		state.Path = nil
		return
	}
	state.Path = path
	state.MoveStarted = now
	state.MoveStartX = state.X
	state.MoveStartY = state.Y
	state.MoveDuration = ActorMovementDurationFromWithSpeed(path, fromX, fromY, path[len(path)-1].X, path[len(path)-1].Y, state.MoveSpeedMS, state.MoveStartX, state.MoveStartY, true)
	state.Moving = state.MoveDuration > 0
	if state.Moving {
		state.Direction = DirectionFromDelta(fromX, fromY, path[1].X, path[1].Y, state.Direction)
	}
}

func (state *FalconRenderState) spriteState(cameraYaw float64) sprites.State {
	if state == nil {
		return sprites.State{}
	}
	sprite := sprites.State{
		ActionFamily: scene.SpriteActionIdle,
		Direction:    state.Direction,
		CameraYaw:    cameraYaw,
		LoopIdle:     true,
		Moving:       state.Moving,
		MoveSpeedMS:  state.MoveSpeedMS,
		Loop:         true,
	}
	if state.Attacking {
		sprite.ActionFamily = scene.SpriteActionWalk
	}
	return sprite
}

func falconFollowDistance(ownerX, ownerY, falconX, falconY float64) int {
	return int(math.Floor(math.Hypot(ownerX-falconX, ownerY-falconY)))
}

func falconFollowTargetCell(x, y float64) (int, int) {
	return int(x), int(y)
}

func falconOvershootTarget(fromX, fromY float64, targetX, targetY int) (int, int) {
	overX := targetX
	overY := targetY
	toX := float64(targetX)
	toY := float64(targetY)
	switch {
	case fromX > toX && fromY > toY:
		overX -= falconAttackOvershoot
		overY -= falconAttackOvershoot
	case fromX < toX && fromY > toY:
		overX += falconAttackOvershoot
		overY -= falconAttackOvershoot
	case fromX < toX && fromY < toY:
		overX += falconAttackOvershoot
		overY += falconAttackOvershoot
	case fromX > toX && fromY < toY:
		overX -= falconAttackOvershoot
		overY += falconAttackOvershoot
	case fromY < toY:
		overY += falconAttackOvershoot
	case fromY > toY:
		overY -= falconAttackOvershoot
	case fromX < toX:
		overX += falconAttackOvershoot
	case fromX > toX:
		overX -= falconAttackOvershoot
	}
	return overX, overY
}

func falconFollowPath(fromX, fromY, targetX, targetY, stopRange int) []worldstate.WalkStep {
	path := []worldstate.WalkStep{{X: fromX, Y: fromY}}
	x, y := fromX, fromY
	for steps := 0; steps < 256 && !falconWithinRange(x, y, targetX, targetY, stopRange); steps++ {
		dx := SignInt(targetX - x)
		dy := SignInt(targetY - y)
		if dx == 0 && dy == 0 {
			break
		}
		x += dx
		y += dy
		path = append(path, worldstate.WalkStep{X: x, Y: y})
	}
	return path
}

func falconWithinRange(x, y, targetX, targetY, stopRange int) bool {
	return int(math.Floor(math.Hypot(float64(targetX-x), float64(targetY-y)))) <= stopRange
}

func SignInt(value int) int {
	switch {
	case value > 0:
		return 1
	case value < 0:
		return -1
	default:
		return 0
	}
}
