package game

import (
	"math"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
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
	// gamepadWalkScreenLead is how far ahead of the player's screen
	// position a d-pad walk aims, in pixels. The world target comes from
	// projecting that point back through the camera, so the walk direction
	// is exactly the direction shown on screen regardless of the isometric
	// camera rotation or zoom.
	gamepadWalkScreenLead = 80.0
)

// ensureHeldMenu lazily builds the MENU-tap overlay with its item actions.
func (m *WorldMode) ensureHeldMenu() {
	if m.ui.heldMenu != nil {
		return
	}
	m.ui.heldMenu = gameui.NewHandheldMenu([]string{
		"Items", "Equipment", "Skills", "Stats", "Quests", "World Map", "Chat", "Screenshot", "Exit Game",
	}, func(ctx client.Context, index int) {
		switch index {
		case 0:
			m.ui.inventoryBag.Toggle(ctx)
		case 1:
			m.ui.equipmentWindow.Toggle(ctx)
		case 2:
			m.ui.skillWindow.Toggle(ctx)
		case 3:
			m.ui.statsWindow.Toggle(ctx)
		case 4:
			m.ui.questWindow.Toggle(ctx)
		case 5:
			if err := m.ui.worldMap.Toggle(ctx); err != nil {
				m.ui.console.AddErrorMessage("%s", err.Error())
			}
		case 6:
			m.ui.console.OpenForTyping(ctx)
		case 7:
			if path, err := ctx.RequestScreenshot(); err == nil {
				m.ui.console.AddSystemMessage("Screenshot: %s", path)
			} else {
				m.ui.console.AddErrorMessage("screenshot failed: %s", err.Error())
			}
		case 8:
			if ctx.RequestQuit != nil {
				ctx.RequestQuit()
			}
		}
	})
}

// updateGamepadControls runs the handheld input layer each world frame. It
// returns true when the A press was consumed as a gamepad action, so the
// same-frame mouse click (the platform emits both) must not also walk.
func (m *WorldMode) updateGamepadControls(ctx client.Context, pointerBlocked bool, now time.Time) bool {
	if ctx.Input == nil || ctx.World == nil {
		return false
	}
	// While an NPC dialog is open, A confirms it — checked before the
	// keyboard-blocked gate below, which reports the open dialog itself
	// and would swallow the press.
	if m.ui.npcDialog.IsOpen() && ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
		m.ui.npcDialog.Confirm(ctx)
		return true
	}
	// The MENU-tap overlay owns every button while open: d-pad moves its
	// selection (its own Update), A activates, MENU closes, walking is
	// suspended.
	if m.ui.heldMenu.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			m.ui.heldMenu.Activate(ctx)
		} else if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF15) {
			m.ui.heldMenu.Close(ctx)
		}
		return true
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
		// Nothing in range: consume the press anyway. Falling through would
		// click wherever the stale pointer sits (top-left corner), making A
		// walk the player northwest for no visible reason.
		return true
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
		glog.Infof("gamepad a attack target id=%d name=%q distance=%.1f player=%d,%d", bestActor.ID, bestActor.Name, bestDistance, playerX, playerY)
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
		glog.Infof("gamepad a pickup target id=%d item_id=%d distance=%.1f player=%d,%d", bestItem.ID, bestItem.ItemID, bestItemDistance, playerX, playerY)
		m.clearLockedAttack()
		m.clearAttackFocus()
		m.requestPickup(ctx, bestItem, "gamepad a")
		return true
	}

	// No combat and no loot: talk to the nearest NPC instead.
	bestTalkDistance := math.Inf(1)
	var bestTalkActor world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if !cursorActorCanTalk(actor) {
			continue
		}
		actorX, actorY := actorRenderPosition(actor, now)
		distance := math.Hypot(actorX-float64(playerX), actorY-float64(playerY))
		if distance < bestTalkDistance {
			bestTalkDistance = distance
			bestTalkActor = actor
		}
	}
	if bestTalkDistance <= gamepadActionRange {
		glog.Infof("gamepad a npc talk target id=%d name=%q distance=%.1f player=%d,%d", bestTalkActor.ID, bestTalkActor.Name, bestTalkDistance, playerX, playerY)
		m.requestNPCTalk(ctx, bestTalkActor, "gamepad a")
		return true
	}
	return false
}

// updateGamepadWalk issues walk requests toward the held d-pad direction.
// The direction is interpreted in screen space and projected through the
// camera, so "up" always walks toward the top of the screen. Two held axes
// combine into a diagonal. While MENU is held (KeyF17), the d-pad steers
// the camera instead — up/down zoom, left/right rotate — and walking stops.
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
		m.gamepadDirLogged = false
		return
	}
	if ctx.Input.KeyCodeDown(gpucontext.KeyF17) {
		// Rate-limited fine camera steps: the held d-pad repeats through
		// the game loop, so gate to a gentle per-tenth-second nudge.
		if now.Sub(m.gamepadCameraAt) < 100*time.Millisecond {
			return
		}
		m.gamepadCameraAt = now
		const zoomFactor = 1.06
		const rotateAngle = 8.0
		if !cameraZoomLockedForMap(ctx) {
			if dy < 0 {
				m.camera.ZoomBy(zoomFactor)
			} else if dy > 0 {
				m.camera.ZoomBy(1 / zoomFactor)
			}
		}
		if !cameraRotationLockedForMap(ctx) {
			if dx < 0 {
				m.camera.Rotate(-rotateAngle)
			} else if dx > 0 {
				m.camera.Rotate(rotateAngle)
			}
		}
		m.gamepadDirLogged = false
		return
	}
	// One info line per held direction: proves on the device that the d-pad
	// reaches the world layer (or not) without debug-level logging.
	if !m.gamepadDirLogged {
		m.gamepadDirLogged = true
		glog.Infof("gamepad walk direction active dir=%d,%d", dx, dy)
	}
	if !m.walkReady(now) || (!m.nextHeldWalkAt.IsZero() && now.Before(m.nextHeldWalkAt)) {
		return
	}
	m.nextHeldWalkAt = now.Add(heldWalkRepeatInterval)

	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	playerX, playerY := currentPlayerCell(ctx, now)
	terrainZ := terrainHeightAt(ctx.World, float64(playerX), float64(playerY))
	point := projection.Project(cellCenter(float64(playerX)), cellCenter(float64(playerY)), terrainZ)
	screenX := math.Min(math.Max(float64(point.x)+float64(dx)*gamepadWalkScreenLead, 0), float64(screenW-1))
	screenY := math.Min(math.Max(float64(point.y)+float64(dy)*gamepadWalkScreenLead, 0), float64(screenH-1))
	targetX, targetY, ok := clickedWalkTarget(ctx, projection, int(screenX), int(screenY))
	if !ok || playerAtWalkTarget(ctx.World.Player, targetX, targetY, now) {
		return
	}
	// Walking elsewhere is the cancel gesture for in-flight actions, same
	// as a ground click: stop chasing a pickup target and drop the attack
	// intent so combat auto-repeat ends.
	m.pendingPickup = pickupIntent{}
	m.cancelAttackIntent()
	if shouldUseTurnOnlyGroundClick(ctx) {
		m.requestChangeDirection(ctx, targetX, targetY, "gamepad walk")
		return
	}
	m.requestWalk(ctx, targetX, targetY, "gamepad walk")
}
