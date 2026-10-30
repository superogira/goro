package actors

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) DrawVendingBoardLabels(screen *render.Frame, ctx client.Context, entries []scene.ActorDrawEntry) {
	icon := m.vendingShopIcon(ctx.Resources)

	for _, entry := range entries {
		if !social.ActorHasVending(entry.Actor) {
			continue
		}
		label := scene.SanitizeActorName(entry.Actor.VendingName)
		labelY := ActorSpriteTopY(entry.ScreenY, entry.Scale) - boardLabelGap - boardLabelHeight(label)
		drawBoardLabel(screen, entry.Actor.VendingName, entry.ScreenX, labelY, icon)
	}
}

func (m *System) DrawChatRoomBoardLabels(screen *render.Frame, ctx client.Context, entries []scene.ActorDrawEntry) {
	for _, entry := range entries {
		if !social.ActorHasChatRoom(entry.Actor) {
			continue
		}
		label := chatRoomBoardLabel(entry.Actor)
		icon := m.chatRoomBoardIcon(ctx.Resources, entry.Actor.ChatRoomPublic)
		labelY := ActorSpriteTopY(entry.ScreenY, entry.Scale) - boardLabelGap - boardLabelHeight(label)
		if social.ActorHasVending(entry.Actor) {
			vendingLabel := scene.SanitizeActorName(entry.Actor.VendingName)
			labelY -= boardLabelHeight(vendingLabel) + boardLabelGap
		}
		drawBoardLabel(screen, label, entry.ScreenX, labelY, icon)
	}
}

func chatRoomBoardLabel(actor worldstate.Actor) string {
	title := scene.SanitizeActorName(actor.ChatRoomTitle)
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
	X float64
	Y float64
	W float64
	H float64
}

func (b vendingBoardBounds) contains(x, y float64) bool {
	return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
}

func drawBoardLabel(screen *render.Frame, label string, centerX, topY float64, icon *render.Image) {
	label = scene.SanitizeActorName(label)
	if label == "" {
		return
	}
	bounds, ok := boardLabelBounds(label, centerX, topY)
	if !ok {
		return
	}
	bounds.X, bounds.Y = render.SnapScreenPoint(screen, bounds.X, bounds.Y)
	drawBoardSurface(screen, bounds)

	contentInset := boardLabelOutlineW + boardLabelBorderW
	contentX := bounds.X + float64(contentInset+boardLabelPadX)
	textX := contentX
	if icon != nil && !icon.Bounds().Empty() {
		drawBoardIcon(screen, icon, contentX, bounds.Y+float64(boardLabelH-boardLabelIcon)/2, boardLabelIcon)
		textX += float64(boardLabelIcon + boardLabelIconGap)
	}
	maxTextWidth := int(bounds.X+bounds.W) - int(math.Round(textX)) - contentInset - boardLabelPadX
	text := trimBoardLabel(label, maxTextWidth)
	textY := bounds.Y + float64(boardLabelH-boardLabelTextH)/2 - 2
	render.DrawUITextAt(screen, text, textX, textY, boardLabelTextColor)
}

func drawBoardSurface(screen *render.Frame, bounds vendingBoardBounds) {
	if screen == nil || bounds.W <= 0 || bounds.H <= 0 {
		return
	}
	surface := cachedBoardSurface()
	if surface == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(bounds.X, bounds.Y)
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
		float32(scene.MaxInt(0, int(boardLabelRadius)-blueInset)),
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
		float32(scene.MaxInt(0, int(boardLabelRadius)-fillInset)),
	)
	boardSurfaceCache = img
	return boardSurfaceCache
}

func drawBoardIcon(screen *render.Frame, icon *render.Image, x, y float64, size int) {
	if screen == nil || icon == nil {
		return
	}
	src := scene.VisibleImageBounds(icon)
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
	screen.DrawTrianglesOwned(vertices, scene.QuadIndices012213, icon, &render.DrawTrianglesOptions{Filter: scene.SpriteDrawFilter(), Address: render.AddressClampToZero})
}

func (m *System) HoveredVendingBoard(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.ActorDeaths[actor.ID]; dead {
			continue
		}
		if actor.ID == 0 || scene.IsLocalActor(ctx, actor.ID) || !social.ActorHasVending(actor) {
			continue
		}
		bounds, ok := VendingBoardActorBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			continue
		}
		dx := bounds.X + bounds.W/2 - float64(mouseX)
		dy := bounds.Y + bounds.H/2 - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func (m *System) HoveredChatRoomBoard(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := m.ActorDeaths[actor.ID]; dead {
			continue
		}
		if actor.ID == 0 || scene.IsLocalActor(ctx, actor.ID) || !social.ActorHasChatRoom(actor) {
			continue
		}
		bounds, ok := m.chatRoomBoardActorBounds(ctx, projection, actor, now)
		if !ok || !bounds.contains(float64(mouseX), float64(mouseY)) {
			continue
		}
		dx := bounds.X + bounds.W/2 - float64(mouseX)
		dy := bounds.Y + bounds.H/2 - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func VendingBoardActorBounds(ctx client.Context, projection scene.Projection, actor worldstate.Actor, now time.Time) (vendingBoardBounds, bool) {
	label := scene.SanitizeActorName(actor.VendingName)
	if label == "" || ctx.World == nil {
		return vendingBoardBounds{}, false
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	topY := ActorSpriteTopY(float64(point.Y), scale) - boardLabelGap - boardLabelHeight(label)
	return boardLabelBounds(label, float64(point.X), topY)
}

func (m *System) chatRoomBoardActorBounds(ctx client.Context, projection scene.Projection, actor worldstate.Actor, now time.Time) (vendingBoardBounds, bool) {
	label := chatRoomBoardLabel(actor)
	if label == "" || ctx.World == nil {
		return vendingBoardBounds{}, false
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	topY := ActorSpriteTopY(float64(point.Y), scale) - boardLabelGap - boardLabelHeight(label)
	if social.ActorHasVending(actor) {
		topY -= boardLabelHeight(actor.VendingName) + boardLabelGap
	}
	return boardLabelBounds(label, float64(point.X), topY)
}

func boardLabelBounds(label string, centerX, topY float64) (vendingBoardBounds, bool) {
	label = scene.SanitizeActorName(label)
	if label == "" {
		return vendingBoardBounds{}, false
	}
	return vendingBoardBounds{
		X: math.Round(centerX - float64(boardLabelW)/2),
		Y: math.Round(topY),
		W: boardLabelW,
		H: boardLabelH,
	}, true
}

func boardLabelHeight(label string) float64 {
	if scene.SanitizeActorName(label) == "" {
		return 0
	}
	return boardLabelH
}

func trimBoardLabel(label string, maxWidth int) string {
	label = scene.SanitizeActorName(label)
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

func (m *System) vendingShopIcon(manager *res.Manager) *render.Image {
	return m.roomBoardIcon(manager, "__interface_basic_interface_shop", "basic_interface\\shop")
}

func (m *System) chatRoomBoardIcon(manager *res.Manager, public bool) *render.Image {
	if public {
		return m.roomBoardIcon(manager, "__interface_basic_interface_chat_open", "basic_interface\\chat_open")
	}
	return m.roomBoardIcon(manager, "__interface_basic_interface_chat_close", "basic_interface\\chat_close")
}

func (m *System) roomBoardIcon(manager *res.Manager, key, resource string) *render.Image {
	if manager == nil {
		return nil
	}
	if m.bindings.Map.Textures == nil {
		m.bindings.Map.Textures = make(map[string]*render.Image)
	}
	if m.bindings.Map.TextureMiss == nil {
		m.bindings.Map.TextureMiss = make(map[string]struct{})
	}
	if texture, ok := m.bindings.Map.Textures[key]; ok {
		return texture
	}
	if _, ok := m.bindings.Map.TextureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.InterfaceTextureCandidates(resource))
	if err != nil {
		m.bindings.Map.TextureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.bindings.Map.Textures[key] = texture
	return texture
}
