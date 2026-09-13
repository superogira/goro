package game

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

// Touch controls: the page (when the touch overlay is enabled) turns touch
// drags on empty ground into a floating virtual stick and provides an action
// button that picks the nearest floor item or attacks the nearest monster.
// The page pushes "touch:" actions through the shared web action queue:
//
//	touch:stick:<dx>,<dy>  stick vector, normalized -1..1; 0,0 releases
//	touch:action           pick/attack the nearest target around the player
//
// Whether the action button finds a target is governed by the same Snap
// Radius gameplay setting as cursor picking, so one knob tunes both.

const (
	// touchStickDeadzone is the minimum stick magnitude that steers; below it
	// the last walk keeps running to completion and no new target is set.
	touchStickDeadzone = 0.2
	// touchStickHoldScreenPx is how far ahead of the player (in screen
	// pixels) the steering target is placed; far enough that the projected
	// cell moves clearly in the dragged direction on every camera zoom.
	touchStickHoldScreenPx = 140
	// touchStickStaleAfter drops steering if the page stops refreshing the
	// vector (finger lifted without the release event, tab hidden, ...).
	touchStickStaleAfter = 1200 * time.Millisecond
	// touchActionBaseCells is the search radius around the player for the
	// action button at Snap Radius 1x; the setting multiplies it.
	touchActionBaseCells = 2.5
)

type touchControls struct {
	stickX    float64
	stickY    float64
	stickSeen time.Time
	active    bool
	lastWalk  time.Time
}

func (m *WorldMode) drainTouchActions(ctx client.Context) {
	for _, action := range gameui.DrainWebActions("touch:") {
		switch {
		case strings.HasPrefix(action, "touch:stick:"):
			m.applyTouchStick(action[len("touch:stick:"):])
		case action == "touch:action":
			m.touchActionNearest(ctx)
		}
	}
}

func (m *WorldMode) applyTouchStick(vec string) {
	x, y, ok := parseTouchStickVector(vec)
	if !ok {
		return
	}
	wasActive := m.touch.active
	m.touch.stickX, m.touch.stickY = x, y
	m.touch.stickSeen = time.Now()
	m.touch.active = x != 0 || y != 0
	if !m.touch.active {
		m.touch.lastWalk = time.Time{}
		return
	}
	if !wasActive {
		// Grabbing the stick breaks off whatever the action button (or a
		// click) started: the player is steering, so chasing a monster or
		// walking to a floor item must stop.
		m.cancelAttackIntent()
		m.pendingPickup = pickupIntent{}
	}
}

func parseTouchStickVector(vec string) (float64, float64, bool) {
	parts := strings.Split(vec, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	x, okX := parseFloatLoose(parts[0])
	y, okY := parseFloatLoose(parts[1])
	if !okX || !okY {
		return 0, 0, false
	}
	return x, y, true
}

func parseFloatLoose(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v, err == nil
}

// updateTouchControls runs every world update: it refreshes steering while a
// stick vector is held and releases stale ones. Walk requests ride the same
// cadence as held ground clicks so the server sees the same traffic shape.
func (m *WorldMode) updateTouchControls(ctx client.Context, now time.Time) {
	if !m.touch.active {
		return
	}
	if now.Sub(m.touch.stickSeen) > touchStickStaleAfter {
		m.touch.active = false
		return
	}
	magnitude := math.Hypot(m.touch.stickX, m.touch.stickY)
	if magnitude < touchStickDeadzone {
		return
	}
	if now.Before(m.touch.lastWalk.Add(heldWalkRepeatInterval)) {
		return
	}
	m.touch.lastWalk = now
	m.steerTouchWalk(ctx, now)
}

// steerTouchWalk converts the screen-space stick vector into a walk target by
// projecting a point touchStickHoldScreenPx ahead of the player and using the
// same screen-to-cell pick the ground click path uses — which keeps steering
// correct under every camera yaw and zoom.
func (m *WorldMode) steerTouchWalk(ctx client.Context, now time.Time) {
	if ctx.World == nil || ctx.World.GAT == nil || ctx.Input == nil {
		return
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	playerX, playerY := currentPlayerCell(ctx, now)
	terrainZ := terrainHeightAt(ctx.World, float64(playerX), float64(playerY))
	playerScreen := projection.Project(cellCenter(float64(playerX)), cellCenter(float64(playerY)), terrainZ)
	dirX, dirY := m.touch.stickX, m.touch.stickY
	if magnitude := math.Hypot(dirX, dirY); magnitude > 1 {
		dirX, dirY = dirX/magnitude, dirY/magnitude
	}
	targetScreenX := int(float64(playerScreen.x) + dirX*touchStickHoldScreenPx)
	targetScreenY := int(float64(playerScreen.y) + dirY*touchStickHoldScreenPx)
	if targetX, targetY, ok := m.hoveredWalkCell(ctx, projection, targetScreenX, targetScreenY); ok {
		m.requestWalk(ctx, targetX, targetY, "touch stick")
	}
}

// touchActionNearest implements the action button: pick the closest floor
// item within range, else attack the closest attackable monster. The range
// comes from the Snap Radius gameplay setting, matching cursor pick reach.
func (m *WorldMode) touchActionNearest(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	now := time.Now()
	playerX, playerY := currentPlayerCell(ctx, now)
	radius := touchActionBaseCells * inputPickMultiplier(ctx)

	bestDistance := math.Inf(1)
	var bestItem world.FloorItem
	for _, item := range ctx.World.Items {
		x, y := floorItemWorldPosition(item)
		distance := math.Hypot(x-float64(playerX), y-float64(playerY))
		if distance <= radius && distance < bestDistance {
			bestDistance = distance
			bestItem = item
		}
	}
	if bestDistance < math.Inf(1) {
		if m.requestPickup(ctx, bestItem, "touch action") {
			return
		}
	}

	bestDistance = math.Inf(1)
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
		if distance <= radius && distance < bestDistance {
			bestDistance = distance
			bestActor = actor
		}
	}
	if bestDistance < math.Inf(1) {
		m.clearLockedAttack()
		m.requestAttack(ctx, bestActor, "touch action")
	}
}
