package game

import (
	"math"
	"time"

	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/world"
	"image/color"
)

// Handheld controls (rg35xx and friends): the d-pad walks the character
// directly and the A button performs a context action — attack the nearest
// attackable actor, else pick up the nearest dropped item. The fbdev
// platform tags A presses with a KeyF13 edge next to the normal mouse
// click, so mouse-driven platforms are untouched.

const (
	// gamepadActionRange bounds the A-button attack/pickup search in tiles;
	// NPC talk is capped tighter (walking up to an NPC 9 tiles away just to
	// have the request drop server-side feels broken).
	gamepadActionRange  = 9.0
	gamepadNPCTalkRange = 5.0
	// gamepadWalkScreenLead is how far ahead of the player's screen
	// position a d-pad walk aims, in pixels. The world target comes from
	// projecting that point back through the camera, so the walk direction
	// is exactly the direction shown on screen regardless of the isometric
	// camera rotation or zoom.
	gamepadWalkScreenLead = 80.0
)

// ensureHeldMenu lazily builds the handheld menu item actions.
func (m *WorldMode) heldMenuActivate(ctx client.Context) {
	glog.Infof("handheld menu activate item=%d %q", m.heldMenuSel, heldMenuItems[m.heldMenuSel])
	switch m.heldMenuSel {
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
}

var heldMenuItems = []string{
	"Items", "Equipment", "Skills", "Stats", "Quests", "World Map", "Chat", "Screenshot", "Exit Game",
}

// updateHeldMenuInput drives the direct-drawn MENU overlay: d-pad moves the
// selection (debounced — the polled d-pad bounces), A or START activates,
// SELECT closes. MENU itself toggles at the world level and must NOT be
// re-checked here — the same JustPressed edge that opened the menu would
// close it within the same frame (observed as open=true logs with nothing
// rendered and walking continuing underneath).
func (m *WorldMode) updateHeldMenuInput(ctx client.Context, now time.Time) {
	if ctx.Input.JustPressed(input.KeyEscape) {
		m.heldMenuOpen = false
		return
	}
	if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) || ctx.Input.JustPressed(input.KeyEnter) {
		m.heldMenuOpen = false
		m.heldMenuActivate(ctx)
		return
	}
	dx, dy := 0, 0
	if ctx.Input.JustPressed(input.KeyArrowUp) {
		dy--
	}
	if ctx.Input.JustPressed(input.KeyArrowDown) {
		dy++
	}
	if ctx.Input.JustPressed(input.KeyArrowLeft) {
		dx--
	}
	if ctx.Input.JustPressed(input.KeyArrowRight) {
		dx++
	}
	if dx == 0 && dy == 0 {
		return
	}
	if now.Sub(m.heldMenuMovedAt) < 180*time.Millisecond {
		return
	}
	m.heldMenuMovedAt = now
	step := dy
	if step == 0 {
		step = dx
	}
	m.heldMenuSel += step
	if m.heldMenuSel < 0 {
		m.heldMenuSel = 0
	}
	if m.heldMenuSel >= len(heldMenuItems) {
		m.heldMenuSel = len(heldMenuItems) - 1
	}
}

// drawHeldMenu renders the overlay with immediate primitives only —
// DrawRect + DrawOutlinedTextAt draw straight into the frame (the volume
// HUD path). The earlier UI-label variants appended to compositor lists
// that never flushed for game-drawn content, so nothing appeared.
func (m *WorldMode) drawHeldMenu(screen *render.Frame) {
	if !m.heldMenuOpen || screen == nil {
		return
	}
	const itemH = 18
	const pad = 10
	const menuW = 190
	titleH := 22
	menuH := titleH + pad + len(heldMenuItems)*itemH + pad
	bounds := screen.Bounds()
	x := (float64(bounds.Dx()) - menuW) / 2
	y := (float64(bounds.Dy()) - float64(menuH)) / 2

	// Panel and title bar.
	render.DrawRect(screen, x, y, menuW, float64(menuH), color.RGBA{R: 24, G: 20, B: 34, A: 235})
	render.DrawRect(screen, x, y, menuW, float64(titleH), color.RGBA{R: 60, G: 48, B: 84, A: 245})
	render.DrawRect(screen, x, y+float64(titleH-2), menuW, 2, color.RGBA{R: 214, G: 178, B: 92, A: 255})
	render.DrawOutlinedTextAt(screen, "MENU", int(x)+pad, int(y)+4, color.RGBA{R: 250, G: 240, B: 210, A: 255}, color.RGBA{A: 200})

	// Items; the selection inverts its row.
	for i, label := range heldMenuItems {
		rowY := y + float64(titleH+pad+i*itemH)
		if i == m.heldMenuSel {
			render.DrawRect(screen, x+4, rowY-1, menuW-8, float64(itemH), color.RGBA{R: 214, G: 178, B: 92, A: 235})
			render.DrawOutlinedTextAt(screen, label, int(x)+pad, int(rowY)+2, color.RGBA{R: 30, G: 22, B: 12, A: 255}, color.RGBA{A: 0})
			continue
		}
		render.DrawOutlinedTextAt(screen, label, int(x)+pad, int(rowY)+2, color.RGBA{R: 235, G: 232, B: 240, A: 255}, color.RGBA{A: 200})
	}
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
	// The direct-drawn MENU overlay owns every button while open.
	if m.heldMenuOpen {
		m.updateHeldMenuInput(ctx, now)
		return true
	}
	// The stats window owns the d-pad while open: up/down (or left/right)
	// move the stat selection, A raises it, walking is suspended.
	if m.ui.statsWindow.IsOpen() {
		if ctx.Input.KeyCodeJustPressed(gpucontext.KeyF13) {
			m.ui.statsWindow.GamepadConfirm(ctx)
			return true
		}
		delta := 0
		if ctx.Input.JustPressed(input.KeyArrowUp) || ctx.Input.JustPressed(input.KeyArrowLeft) {
			delta--
		}
		if ctx.Input.JustPressed(input.KeyArrowDown) || ctx.Input.JustPressed(input.KeyArrowRight) {
			delta++
		}
		if delta != 0 && now.Sub(m.statsSelMovedAt) >= 180*time.Millisecond {
			m.statsSelMovedAt = now
			m.ui.statsWindow.GamepadNavigate(ctx, delta)
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
	if bestTalkDistance <= gamepadNPCTalkRange {
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
