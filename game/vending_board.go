package game

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) drawVendingBoardLabels(screen *render.Frame, ctx client.Context, entries []sceneActorDrawEntry) {
	icon := m.vendingShopIcon(ctx.Resources)
	for _, entry := range entries {
		if !actorHasVending(entry.actor) {
			continue
		}
		label := sanitizeActorName(entry.actor.VendingName)
		labelY := actorSpriteTopY(entry.screenY, entry.scale) - boardLabelGap - boardLabelHeight(label)
		drawBoardLabel(screen, entry.actor.VendingName, entry.screenX, labelY, icon)
	}
}

func (m *WorldMode) drawChatRoomBoardLabels(screen *render.Frame, ctx client.Context, entries []sceneActorDrawEntry) {
	bounds := make(map[uint32]vendingBoardBounds, len(entries))
	for _, entry := range entries {
		if !actorHasChatRoom(entry.actor) {
			continue
		}
		label := chatRoomBoardLabel(entry.actor)
		icon := m.chatRoomBoardIcon(ctx.Resources, entry.actor.ChatRoomPublic)
		labelY := actorSpriteTopY(entry.screenY, entry.scale) - boardLabelGap - boardLabelHeight(label)
		if actorHasVending(entry.actor) {
			vendingLabel := sanitizeActorName(entry.actor.VendingName)
			labelY -= boardLabelHeight(vendingLabel) + boardLabelGap
		}
		drawBoardLabel(screen, label, entry.screenX, labelY, icon)
		if box, ok := boardLabelBounds(label, entry.screenX, labelY); ok {
			bounds[entry.actor.ID] = box
		}
	}
	// Hover and click hit-testing read these drawn bounds: recomputing them
	// from world state drifts, because the draw path anchors the sprite and
	// scales the local player's body while the projection shortcut does not.
	m.chatBoardBounds = bounds
}

func chatRoomBoardLabel(actor worldstate.Actor) string {
	title := sanitizeActorName(actor.ChatRoomTitle)
	if title == "" {
		return ""
	}
	if actor.ChatRoomLimit == 0 {
		return title
	}
	return fmt.Sprintf("%s (%d/%d)", title, actor.ChatRoomCount, actor.ChatRoomLimit)
}

const (
	boardLabelW        = 158
	boardLabelH        = 36
	boardLabelPadX     = 5
	boardLabelIconGap  = 5
	boardLabelGap      = 4
	boardLabelIcon     = 24
	boardLabelTextH    = 20
	boardLabelRadius   = 4
	boardLabelOutlineW = 3
	boardLabelBorderW  = 1
)

var (
	boardLabelFillColor    = color.RGBA{R: 255, G: 255, B: 255, A: 245}
	boardLabelOutlineColor = color.RGBA{R: 255, G: 255, B: 255, A: 245}
	boardLabelBorderColor  = color.RGBA{R: 74, G: 138, B: 202, A: 245}
	boardLabelTextColor    = color.RGBA{R: 30, G: 34, B: 40, A: 255}
)

var boardSurfaceCache *render.Image

type vendingBoardBounds struct {
	x float64
	y float64
	w float64
	h float64
}

func (b vendingBoardBounds) contains(x, y float64) bool {
	return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h
}

func drawBoardLabel(screen *render.Frame, label string, centerX, topY float64, icon *render.Image) {
	label = sanitizeActorName(label)
	if label == "" {
		return
	}
	bounds, ok := boardLabelBounds(label, centerX, topY)
	if !ok {
		return
	}
	bounds.x, bounds.y = render.SnapScreenPoint(screen, bounds.x, bounds.y)
	drawBoardSurface(screen, bounds)

	contentInset := boardLabelOutlineW + boardLabelBorderW
	contentX := bounds.x + float64(contentInset+boardLabelPadX)
	textX := contentX
	if icon != nil && !icon.Bounds().Empty() {
		drawBoardIcon(screen, icon, contentX, bounds.y+float64(boardLabelH-boardLabelIcon)/2, boardLabelIcon)
		textX += float64(boardLabelIcon + boardLabelIconGap)
	}
	maxTextWidth := int(bounds.x+bounds.w) - int(math.Round(textX)) - contentInset - boardLabelPadX
	text := trimBoardLabel(label, maxTextWidth)
	textY := bounds.y + float64(boardLabelH-boardLabelTextH)/2 - 2
	render.DrawUITextAt(screen, text, textX, textY, boardLabelTextColor)
}

func drawBoardSurface(screen *render.Frame, bounds vendingBoardBounds) {
	if screen == nil || bounds.w <= 0 || bounds.h <= 0 {
		return
	}
	surface := cachedBoardSurface()
	if surface == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(bounds.x, bounds.y)
	opts.Filter = render.FilterLinear
	screen.DrawImage(surface, &opts)
}

func cachedBoardSurface() *render.Image {
	if boardSurfaceCache != nil {
		return boardSurfaceCache
	}
	img := render.NewImage(boardLabelW, boardLabelH)
	gameui.DrawRoundedSurface(img, 0, 0, boardLabelW, boardLabelH, boardLabelOutlineColor, color.RGBA{}, boardLabelRadius)
	blueInset := boardLabelOutlineW
	gameui.DrawRoundedSurface(
		img,
		blueInset,
		blueInset,
		boardLabelW-blueInset*2,
		boardLabelH-blueInset*2,
		boardLabelBorderColor,
		color.RGBA{},
		float32(maxInt(0, int(boardLabelRadius)-blueInset)),
	)
	fillInset := boardLabelOutlineW + boardLabelBorderW
	gameui.DrawRoundedSurface(
		img,
		fillInset,
		fillInset,
		boardLabelW-fillInset*2,
		boardLabelH-fillInset*2,
		boardLabelFillColor,
		color.RGBA{},
		float32(maxInt(0, int(boardLabelRadius)-fillInset)),
	)
	boardSurfaceCache = img
	return boardSurfaceCache
}

func drawBoardIcon(screen *render.Frame, icon *render.Image, x, y float64, size int) {
	if screen == nil || icon == nil {
		return
	}
	src := visibleImageBounds(icon)
	if src.Empty() {
		return
	}
	srcW, srcH := float64(src.Dx()), float64(src.Dy())
	if srcW <= 0 || srcH <= 0 {
		return
	}
	if size <= 0 {
		size = boardLabelIcon
	}
	scale := math.Min(float64(size)/srcW, float64(size)/srcH)
	dstW, dstH := srcW*scale, srcH*scale
	dstX := x + (float64(size)-dstW)/2
	dstY := y + (float64(size)-dstH)/2
	vertices := []render.Vertex{
		{DstX: float32(dstX), DstY: float32(dstY), SrcX: float32(src.Min.X), SrcY: float32(src.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY), SrcX: float32(src.Max.X), SrcY: float32(src.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX), DstY: float32(dstY + dstH), SrcX: float32(src.Min.X), SrcY: float32(src.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY + dstH), SrcX: float32(src.Max.X), SrcY: float32(src.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	screen.DrawTrianglesOwned(vertices, quadIndices012213, icon, &render.DrawTrianglesOptions{Filter: spriteDrawFilter(), Address: render.AddressClampToZero})
}

func (m *WorldMode) hoveredVendingBoard(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if actor.ID == 0 || isLocalActor(ctx, actor.ID) || !actorHasVending(actor) {
			continue
		}
		bounds, ok := vendingBoardActorBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			continue
		}
		dx := bounds.x + bounds.w/2 - float64(mouseX)
		dy := bounds.y + bounds.h/2 - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func (m *WorldMode) hoveredChatRoomBoard(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.actorDeaths[actor.ID]; dead {
			continue
		}
		if actor.ID == 0 || isLocalActor(ctx, actor.ID) || !actorHasChatRoom(actor) {
			continue
		}
		bounds, ok := m.chatRoomBoardHitBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			continue
		}
		dx := bounds.x + bounds.w/2 - float64(mouseX)
		dy := bounds.y + bounds.h/2 - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func vendingBoardActorBounds(ctx client.Context, projection sceneProjection, actor worldstate.Actor, now time.Time) (vendingBoardBounds, bool) {
	label := sanitizeActorName(actor.VendingName)
	if label == "" || ctx.World == nil {
		return vendingBoardBounds{}, false
	}
	actorX, actorY := actorRenderPosition(actor, now)
	terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
	topY := actorSpriteTopY(float64(point.y), scale) - boardLabelGap - boardLabelHeight(label)
	return boardLabelBounds(label, float64(point.x), topY)
}

func (m *WorldMode) chatRoomBoardActorBounds(ctx client.Context, projection sceneProjection, actor worldstate.Actor, now time.Time) (vendingBoardBounds, bool) {
	label := chatRoomBoardLabel(actor)
	if label == "" || ctx.World == nil {
		return vendingBoardBounds{}, false
	}
	actorX, actorY := actorRenderPosition(actor, now)
	terrainZ := terrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(cellCenter(actorX), cellCenter(actorY), terrainZ)
	scale := actorBillboardScreenScale(projection, cellCenter(actorX), cellCenter(actorY), terrainZ)
	topY := actorSpriteTopY(float64(point.y), scale) - boardLabelGap - boardLabelHeight(label)
	if actorHasVending(actor) {
		topY -= boardLabelHeight(actor.VendingName) + boardLabelGap
	}
	return boardLabelBounds(label, float64(point.x), topY)
}

// chatRoomBoardHitBounds prefers the bounds captured while drawing the
// board this frame and falls back to the projection shortcut before the
// first draw (and in tests).
func (m *WorldMode) chatRoomBoardHitBounds(ctx client.Context, projection sceneProjection, actor worldstate.Actor, now time.Time) (vendingBoardBounds, bool) {
	if bounds, ok := m.chatBoardBounds[actor.ID]; ok {
		return bounds, true
	}
	return m.chatRoomBoardActorBounds(ctx, projection, actor, now)
}

func boardLabelBounds(label string, centerX, topY float64) (vendingBoardBounds, bool) {
	label = sanitizeActorName(label)
	if label == "" {
		return vendingBoardBounds{}, false
	}
	return vendingBoardBounds{
		x: math.Round(centerX - float64(boardLabelW)/2),
		y: math.Round(topY),
		w: boardLabelW,
		h: boardLabelH,
	}, true
}

func boardLabelHeight(label string) float64 {
	if sanitizeActorName(label) == "" {
		return 0
	}
	return boardLabelH
}

func trimBoardLabel(label string, maxWidth int) string {
	label = sanitizeActorName(label)
	if label == "" || maxWidth <= 0 {
		return ""
	}
	maxWidth -= 14
	if maxWidth <= 0 {
		return "..."
	}
	if w, _ := render.BitmapTextSize(label); w <= maxWidth {
		return label
	}
	runes := []rune(label)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + "..."
		if w, _ := render.BitmapTextSize(candidate); w <= maxWidth {
			return candidate
		}
	}
	return "..."
}

func (m *WorldMode) vendingShopIcon(manager *res.Manager) *render.Image {
	return m.roomBoardIcon(manager, "__interface_basic_interface_shop", "basic_interface\\shop")
}

func (m *WorldMode) chatRoomBoardIcon(manager *res.Manager, public bool) *render.Image {
	if public {
		return m.roomBoardIcon(manager, "__interface_basic_interface_chat_open", "basic_interface\\chat_open")
	}
	return m.roomBoardIcon(manager, "__interface_basic_interface_chat_close", "basic_interface\\chat_close")
}

func (m *WorldMode) roomBoardIcon(manager *res.Manager, key, resource string) *render.Image {
	if manager == nil {
		return nil
	}
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.InterfaceTextureCandidates(resource))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture
}
