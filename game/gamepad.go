package game

import (
	"math"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/world"
)

// Handheld controls (rg35xx and friends): the d-pad walks the character
// directly and the A button performs a context action — attack the nearest
// attackable actor, else pick up the nearest dropped item. The fbdev
// platform tags A presses with a KeyF13 edge next to the normal mouse
// click, so mouse-driven platforms are untouched.

const (
	// gamepadActionRange bounds the A-button attack/pickup search in tiles.
	gamepadActionRange = 9.0
	// gamepadWalkLeadTiles is how far ahead of the player a d-pad walk
	// request aims; the server pathfinds, so a longer lead means fewer
	// requests for the same distance.
	gamepadWalkLeadTiles = 4
)

// updateGamepadControls runs the handheld input layer each world frame. It
// returns true when the A press was consumed as a gamepad action, so the
// same-frame mouse click (the platform emits both) must not also walk.
func (m *WorldMode) updateGamepadControls(ctx client.Context, pointerBlocked bool, now time.Time) bool {
	if ctx.Input == nil || ctx.World == nil {
		return false
	}
	if m.ui.console.Active() || m.ui.keyboardInputBlocked(ctx) {
		return false
	}
	if m.pendingSkill.skill.ID != 0 || m.pendingPetCapture.active {
		return false
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) && !pointerBlocked {
		if m.gamepadPrimaryAction(ctx, now) {
			return true
		}
	}
	m.updateGamepadWalk(ctx, pointerBlocked, now)
	return false
}

// gamepadPrimaryAction attacks the nearest attackable actor in range, else
// requests pickup of the nearest floor item in range.
func (m *WorldMode) gamepadPrimaryAction(ctx client.Context, now time.Time) bool {
	playerX, playerY := currentPlayerCell(ctx, now)

	bestDistance := math.Inf(1)
	var bestActor world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if !actorCanBeAttackClicked(ctx, actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		distance := math.Hypot(actorX-float64(playerX), actorY-float64(playerY))
		if distance < bestDistance {
			bestDistance = distance
			bestActor = actor
		}
	}
	if bestDistance <= gamepadActionRange {
		glog.Debugf("gamepad a attack target id=%d name=%q distance=%.1f player=%d,%d", bestActor.ID, bestActor.Name, bestDistance, playerX, playerY)
		m.requestAttack(ctx, bestActor, "gamepad a")
		return true
	}

	bestItemDistance := math.Inf(1)
	var bestItem world.FloorItem
	for _, item := range ctx.World.Items {
		distance := math.Hypot(float64(item.X)-float64(playerX), float64(item.Y)-float64(playerY))
		if distance < bestItemDistance {
			bestItemDistance = distance
			bestItem = item
		}
	}
	if bestItemDistance <= gamepadActionRange {
		glog.Debugf("gamepad a pickup target id=%d item_id=%d distance=%.1f player=%d,%d", bestItem.ID, bestItem.ItemID, bestItemDistance, playerX, playerY)
		m.clearLockedAttack()
		m.clearAttackFocus()
		m.requestPickup(ctx, bestItem, "gamepad a")
		return true
	}
	return false
}

// updateGamepadWalk issues walk requests toward the held d-pad direction.
// Two held axes combine into a diagonal.
func (m *WorldMode) updateGamepadWalk(ctx client.Context, pointerBlocked bool, now time.Time) {
	if pointerBlocked {
		return
	}
	dx, dy := 0, 0
	if ctx.Input.Pressed(input.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.Pressed(input.KeyArrowRight) {
		dx++
	}
	if ctx.Input.Pressed(input.KeyArrowUp) {
		dy--
	}
	if ctx.Input.Pressed(input.KeyArrowDown) {
		dy++
	}
	if dx == 0 && dy == 0 {
		return
	}
	if !m.walkReady(now) || (!m.nextHeldWalkAt.IsZero() && now.Before(m.nextHeldWalkAt)) {
		return
	}
	m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)

	playerX, playerY := currentPlayerCell(ctx, now)
	for lead := gamepadWalkLeadTiles; lead >= 1; lead-- {
		targetX, targetY := playerX+dx*lead, playerY+dy*lead
		if !m.gamepadWalkTarget(ctx, targetX, targetY) {
			continue
		}
		m.cancelAttackIntent()
		m.requestWalk(ctx, targetX, targetY, "gamepad walk")
		return
	}
}

// gamepadWalkTarget reports whether a d-pad walk destination is a tile the
// client believes is walkable.
func (m *WorldMode) gamepadWalkTarget(ctx client.Context, x, y int) bool {
	if ctx.World == nil || ctx.World.GAT == nil {
		return walkTargetInBounds(ctx, x, y)
	}
	return ctx.World.GAT.InBounds(x, y) && ctx.World.GAT.Walkable(x, y)
}
