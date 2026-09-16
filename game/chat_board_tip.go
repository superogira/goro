package game

import (
	"fmt"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

// updateChatBoardTip mirrors the board under the cursor as a DOM tooltip
// (web build): the board pills trim long titles, so hovering shows the
// full label near the cursor. Chat rooms win over vending boards when the
// pills overlap (chat boards sit above). Unlike the click path, the local
// player's own board is included — reading your own title is the common
// case. Native builds have no DOM layer and skip this.
func (m *WorldMode) updateChatBoardTip(ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil || !gameui.BoardTipWebAvailable() {
		return
	}
	label := ""
	if actor, ok := m.hoveredChatBoardTipTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
		label = chatRoomBoardLabel(actor)
	} else if actor, ok := m.hoveredVendingBoardTipTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now); ok {
		label = sanitizeActorName(actor.VendingName)
	}
	if label == "" {
		if m.chatBoardTipSig == "" {
			return
		}
		m.chatBoardTipSig = ""
		gameui.BoardTipWebSync("", 0, 0)
		return
	}
	x, y := ctx.Input.MouseX, ctx.Input.MouseY
	sig := fmt.Sprintf("%s|%d|%d", label, x, y)
	if sig == m.chatBoardTipSig {
		return
	}
	m.chatBoardTipSig = sig
	gameui.BoardTipWebSync(label, x, y)
}

func (m *WorldMode) hoveredVendingBoardTipTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	check := func(actor worldstate.Actor) {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			return
		}
		if actor.ID == 0 || !actorHasVending(actor) {
			return
		}
		bounds, ok := m.vendingBoardHitBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			return
		}
		dx := bounds.x + bounds.w/2 - float64(mouseX)
		dy := bounds.y + bounds.h/2 - float64(mouseY)
		if distance := dx*dx + dy*dy; distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	if player := ctx.World.Player; player.ID != 0 || ctx.Session != nil {
		if ctx.Session != nil {
			player.ID = ctx.Session.CharID
		}
		check(player)
	}
	for _, actor := range ctx.World.Actors {
		check(actor)
	}
	return best, bestDistance < math.Inf(1)
}

func (m *WorldMode) hoveredChatBoardTipTarget(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	check := func(actor worldstate.Actor) {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			return
		}
		if actor.ID == 0 || !actorHasChatRoom(actor) {
			return
		}
		bounds, ok := m.chatRoomBoardHitBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			return
		}
		dx := bounds.x + bounds.w/2 - float64(mouseX)
		dy := bounds.y + bounds.h/2 - float64(mouseY)
		if distance := dx*dx + dy*dy; distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	// Same ID fix-up as the draw list: World.Player carries the chat room
	// state but not the session CharID, and the board bounds map is keyed
	// by the ID the entries were built with.
	if player := ctx.World.Player; player.ID != 0 || ctx.Session != nil {
		if ctx.Session != nil {
			player.ID = ctx.Session.CharID
		}
		check(player)
	}
	for _, actor := range ctx.World.Actors {
		check(actor)
	}
	return best, bestDistance < math.Inf(1)
}
