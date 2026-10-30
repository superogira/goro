package game

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/game/pet"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

const (
	cursorActionDefault = 0
	cursorActionTalk    = 1
	cursorActionClick   = 2
	cursorActionLock    = 3
	cursorActionRotate  = 4
	cursorActionAttack  = 5
	cursorActionWarp    = 7
	cursorActionPick    = 9
	cursorActionTarget  = 10
	cursorActionTarget2 = 11
	cursorActionNoWalk  = 13
)

// cursorSnapTriggerScale sizes the snap/magnet circle relative to the pick
// bounds. At 1.0 the circle circumscribes the whole pick box, so the pick
// cursor and magnet engage everywhere a click would land — enabling snap can
// only make targeting feel bigger, never shrink it to the box's inner half
// (which is what the previous 0.5 did: hovering a floor item showed the walk
// cursor everywhere except within a few pixels of its center).
const cursorSnapTriggerScale = 1.0

// inputPickMultiplier returns the player's pick/snap area enlargement. Touch
// input has no pixel precision, so the gameplay settings expose a multiplier
// that grows both the pick bounds (which target a click finds) and the cursor
// snap distance (which decides click-vs-walk) for actors and floor items.
func inputPickMultiplier(ctx client.Context) float64 {
	multiplier := 0.0
	if ctx.Session != nil && ctx.Session.SnapRadius > 0 {
		multiplier = ctx.Session.SnapRadius
	} else {
		multiplier = ctx.Config.Gameplay.SnapRadius
	}
	if multiplier < 0.5 {
		return 0.5
	}
	if multiplier > 3 {
		return 3
	}
	return multiplier
}

type cursorActionInfo struct {
	drawX     float64
	drawY     float64
	delayMult float64
}

type roCursorState struct {
	view      *scene.SpriteView
	viewMiss  bool
	fallback  *render.Image
	action    int
	started   time.Time
	loadTried bool
}

var cursorActionInfos = map[int]cursorActionInfo{
	cursorActionDefault: {drawX: 1, drawY: 19, delayMult: 2.0},
	cursorActionTalk:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionLock:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionRotate:  {drawX: 18, drawY: 26, delayMult: 1.0},
	cursorActionWarp:    {drawX: 10, drawY: 32, delayMult: 1.0},
	cursorActionPick:    {drawX: 20, drawY: 40, delayMult: 1.0},
	cursorActionTarget:  {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionTarget2: {drawX: 20, drawY: 50, delayMult: 0.5},
	cursorActionNoWalk:  {drawX: 13, drawY: 25, delayMult: 1.0},
}

func (m *WorldMode) drawROCursor(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if ctx.Input == nil {
		cursorWebHide()
		return
	}
	render.SetCursorMode(render.CursorModeHidden)
	action := m.cursorDesiredAction(ctx, projection, now)
	magnetX, magnetY := m.cursorMagnetOffset(ctx, projection, action, now)
	state := m.cursorState()
	state.draw(screen, ctx, action, now, magnetX, magnetY)
	m.storeCursorState(state)
	m.skills().DrawPendingSkillCursorLevel(screen, ctx, m.skillSystem.Pending.Skill, magnetX, magnetY)
}

func (m *WorldMode) cursorState() *roCursorState {
	return &roCursorState{
		view:     m.cursorView,
		viewMiss: m.cursorViewMiss,
		fallback: m.cursorFallback,
		action:   m.cursorAction,
		started:  m.cursorStarted,
	}
}

func (m *WorldMode) storeCursorState(state *roCursorState) {
	if state == nil {
		return
	}
	m.cursorView = state.view
	m.cursorViewMiss = state.viewMiss
	m.cursorFallback = state.fallback
	m.cursorAction = state.action
	m.cursorStarted = state.started
}

func (m *WorldMode) loadedCursorState(ctx client.Context) *roCursorState {
	state := m.cursorState()
	state.ensureLoaded(ctx)
	m.storeCursorState(state)
	return state
}

func (s *roCursorState) ensureLoaded(ctx client.Context) {
	if s == nil || s.loadTried || s.view != nil || s.viewMiss {
		return
	}
	s.loadTried = true
	if view, status := sprites.LoadCursor(ctx.Resources); view != nil {
		s.view = view
	} else {
		s.viewMiss = true
		glog.Warnf("cursor resources unavailable: %s", status)
	}
}

func (s *roCursorState) draw(screen *render.Frame, ctx client.Context, action int, now time.Time, magnetX, magnetY float64) {
	if s == nil || screen == nil || ctx.Input == nil {
		return
	}
	s.ensureLoaded(ctx)
	if action != s.action {
		s.action = action
		s.started = now
	}
	if s.started.IsZero() {
		s.started = now
	}
	frame, ok := s.frame(action, cursorInfo(action), now)
	if cursorWebEnabled() {
		// DOM twin: same billboard, same anchor/hotspot and magnet math —
		// only the destination changes from a canvas blit to a positioned
		// element, keeping the cursor crisp at any resolution scale.
		if ok {
			cursorWebSync(frame.Image, fmt.Sprintf("%p", frame.Image),
				float64(ctx.Input.MouseX)-frame.AnchorX-magnetX,
				float64(ctx.Input.MouseY)-frame.AnchorY-magnetY)
		} else {
			cursorWebSync(s.fallbackTexture(), "fallback",
				float64(ctx.Input.MouseX), float64(ctx.Input.MouseY))
		}
		return
	}
	if !ok {
		drawFallbackROCursor(screen, s.fallbackTexture(), ctx.Input.MouseX, ctx.Input.MouseY)
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(ctx.Input.MouseX)-frame.AnchorX-magnetX, float64(ctx.Input.MouseY)-frame.AnchorY-magnetY)
	opts.Filter = scene.SpriteDrawFilter()
	screen.DrawImage(frame.Image, &opts)
}

func (s *roCursorState) frame(action int, info cursorActionInfo, now time.Time) (*scene.Billboard, bool) {
	return s.frameAt(action, info, s.started, now)
}

func (s *roCursorState) frameAt(action int, info cursorActionInfo, start, now time.Time) (*scene.Billboard, bool) {
	if s == nil || s.view == nil || s.viewMiss || s.view.Act == nil {
		return nil, false
	}
	if action < 0 || action >= len(s.view.Act.Actions) || len(s.view.Act.Actions[action].Animations) == 0 {
		action = cursorActionDefault
		info = cursorInfo(action)
	}
	actionDef := s.view.Act.Actions[action]
	delay := float64(actionDef.DelayMS) * info.delayMult
	// Cycle every motion of the action — the roBrowser behaviour this
	// cursor pipeline is ported from (anim = frame % animations.length):
	// cursor acts encode their animation as the full motion list, and the
	// classic look plays through it.
	motion := scene.SpriteMotionIndexWithDelay(actionDef, start, now, true, delay)
	return sprites.CursorFrameBillboard(s.view, action, motion, info.drawX, info.drawY)
}

func (m *WorldMode) cursorDesiredAction(ctx client.Context, projection scene.Projection, now time.Time) int {
	mouseX, mouseY := ctx.Input.MouseX, ctx.Input.MouseY
	if m.petSystem.SlotMachine.Active {
		return cursorActionClick
	}
	if m.npc().CutinPointerBlocked(ctx) {
		return cursorActionDefault
	}
	if action, ok := uiCursorAction(ctx); ok {
		return action
	}
	if ctx.Input.MousePressed(input.MouseButtonRight) {
		return cursorActionRotate
	}
	if m.skillSystem.Pending.Skill.ID != 0 {
		actor, ok := m.skills().ClickedSkillTarget(ctx, projection, m.skillSystem.Pending.Skill, mouseX, mouseY, now)
		if ok && (!cursorSnapTargets(ctx) || m.cursorActorSnapEligible(ctx, projection, actor, now)) {
			return cursorActionTarget2
		}
		return cursorActionTarget
	}
	if m.petSystem.PendingCapture.Active {
		actor, ok := clickedSkillTarget(ctx, projection, pet.CaptureTargetSkill(), mouseX, mouseY, now, m.actorSystem.ActorDeaths)
		if ok && (!cursorSnapTargets(ctx) || m.cursorActorSnapEligible(ctx, projection, actor, now)) {
			return cursorActionTarget2
		}
		return cursorActionTarget
	}
	if _, ok := m.actors().HoveredVendingBoard(ctx, projection, mouseX, mouseY, now); ok {
		return cursorActionClick
	}
	if _, ok := m.actors().HoveredChatRoomBoard(ctx, projection, mouseX, mouseY, now); ok {
		return cursorActionClick
	}
	if item, ok := clickedGroundItem(ctx, projection, mouseX, mouseY, now); ok && (!cursorSnapItems(ctx) || cursorGroundItemSnapEligible(ctx, projection, item, now)) {
		return cursorActionPick
	}
	if actor, ok := hoveredCursorActor(ctx, projection, mouseX, mouseY, now, m.actorSystem.ActorDeaths); ok {
		switch {
		case npc.IsWarpActor(actor):
			return cursorActionWarp
		case actors.ActorCanBeAttackClicked(ctx, actor):
			if !cursorSnapTargets(ctx) || m.cursorActorSnapEligible(ctx, projection, actor, now) {
				return cursorActionAttack
			}
		case npc.CanTalk(actor):
			return cursorActionTalk
		}
	}
	if ctx.Input.MousePressed(input.MouseButtonLeft) {
		return cursorActionClick
	}
	if ctx.World != nil && ctx.World.GAT != nil {
		if _, _, ok := m.hoveredWalkCell(ctx, projection, mouseX, mouseY); !ok {
			return cursorActionNoWalk
		}
	}
	return cursorActionDefault
}

func (m *WorldMode) cursorMagnetOffset(ctx client.Context, projection scene.Projection, action int, now time.Time) (float64, float64) {
	if ctx.Input == nil {
		return 0, 0
	}
	switch action {
	case cursorActionPick:
		if !cursorSnapItems(ctx) {
			return 0, 0
		}
		item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now)
		if !ok {
			return 0, 0
		}
		targetX, targetY, scale, ok := cursorGroundItemMagnetTarget(ctx, projection, item, now)
		if !ok || !pointInCursorSnapDistance(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), targetX, targetY, groundItemCursorSnapRadius(scale)*inputPickMultiplier(ctx)) {
			return 0, 0
		}
		return float64(ctx.Input.MouseX) - targetX, float64(ctx.Input.MouseY) - targetY
	case cursorActionAttack:
		if !cursorSnapTargets(ctx) {
			return 0, 0
		}
		actor, ok := clickedAttackTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths)
		if !ok {
			return 0, 0
		}
		return m.cursorActorMagnetOffset(ctx, projection, actor, now)
	case cursorActionTarget2:
		if !cursorSnapTargets(ctx) {
			return 0, 0
		}
		skill, ok := m.pendingTargetCursorSkill()
		if !ok {
			return 0, 0
		}
		actor, ok := m.skills().ClickedSkillTarget(ctx, projection, skill, ctx.Input.MouseX, ctx.Input.MouseY, now)
		if !ok {
			return 0, 0
		}
		return m.cursorActorMagnetOffset(ctx, projection, actor, now)
	default:
		return 0, 0
	}
}

func (m *WorldMode) pendingTargetCursorSkill() (session.Skill, bool) {
	if m.skillSystem.Pending.Skill.ID != 0 {
		return m.skillSystem.Pending.Skill, true
	}
	if m.petSystem.PendingCapture.Active {
		return pet.CaptureTargetSkill(), true
	}
	return session.Skill{}, false
}

func (m *WorldMode) drawPendingTargetBanner(screen *render.Frame, ctx client.Context) {
	skill, ok := m.pendingTargetCursorSkill()
	if !ok {
		return
	}
	name := skills.SkillLabel(skill)
	if ctx.Resources != nil && skill.ID != pet.CaptureSkillID {
		if displayName, ok := ctx.Resources.SkillDisplayName(int(skill.ID)); ok {
			name = displayName
		}
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	// Scale the skill name's vertical position from a 480-pixel viewport.
	y := float64(46 * height / 480)
	render.DrawUITextBanner(screen, name, float64(width)/2, y, float64(scene.MinInt(520, width)), color.RGBA{R: 50, G: 255, B: 50, A: 255}, 12, false)
}

func (m *WorldMode) cursorActorMagnetOffset(ctx client.Context, projection scene.Projection, actor world.Actor, now time.Time) (float64, float64) {
	targetX, targetY, scale, ok := m.cursorActorMagnetTarget(ctx, projection, actor, now)
	if !ok || !pointInCursorSnapDistance(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), targetX, targetY, actorCursorSnapRadius(scale)*inputPickMultiplier(ctx)) {
		return 0, 0
	}
	return float64(ctx.Input.MouseX) - targetX, float64(ctx.Input.MouseY) - targetY
}

func (m *WorldMode) cursorActorSnapEligible(ctx client.Context, projection scene.Projection, actor world.Actor, now time.Time) bool {
	if ctx.Input == nil {
		return false
	}
	targetX, targetY, scale, ok := m.cursorActorMagnetTarget(ctx, projection, actor, now)
	return ok && pointInCursorSnapDistance(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), targetX, targetY, actorCursorSnapRadius(scale)*inputPickMultiplier(ctx))
}

func (m *WorldMode) cursorActorMagnetTarget(ctx client.Context, projection scene.Projection, actor world.Actor, now time.Time) (float64, float64, float64, bool) {
	if ctx.Input == nil || ctx.World == nil {
		return 0, 0, 0, false
	}
	actorX, actorY := scene.ActorRenderPosition(actor, now)
	terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
	point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
	if _, ok := m.skillSystem.Units[actor.ID]; ok {
		return float64(point.X), float64(point.Y), scale, true
	}
	if targetX, targetY, ok := m.cursorActorSpriteCenter(ctx, projection, actor, point, scale, now); ok {
		return targetX, targetY, scale, true
	}
	targetX, targetY := actors.ActorPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	return targetX, targetY, scale, true
}

func cursorGroundItemSnapEligible(ctx client.Context, projection scene.Projection, item world.FloorItem, now time.Time) bool {
	if ctx.Input == nil {
		return false
	}
	targetX, targetY, scale, ok := cursorGroundItemMagnetTarget(ctx, projection, item, now)
	return ok && pointInCursorSnapDistance(float64(ctx.Input.MouseX), float64(ctx.Input.MouseY), targetX, targetY, groundItemCursorSnapRadius(scale)*inputPickMultiplier(ctx))
}

func cursorGroundItemMagnetTarget(ctx client.Context, projection scene.Projection, item world.FloorItem, now time.Time) (float64, float64, float64, bool) {
	if ctx.World == nil {
		return 0, 0, 0, false
	}
	x, y := floorItemWorldPosition(item)
	z := floorItemRenderHeight(ctx.World, item, now)
	point := projection.Project(scene.CellCenter(x), scene.CellCenter(y), z)
	scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(x), scene.CellCenter(y), z) * 0.42
	targetX, targetY := groundItemPickBoundsCenter(float64(point.X), float64(point.Y), scale)
	return targetX, targetY, scale, true
}

func pointInCursorSnapDistance(mouseX, mouseY, centerX, centerY, radius float64) bool {
	dx := mouseX - centerX
	dy := mouseY - centerY
	return dx*dx+dy*dy <= radius*radius
}

func actorCursorSnapRadius(scale float64) float64 {
	scale = actors.NormalizePickScale(scale)
	halfWidth := 44 * scale * cursorSnapTriggerScale
	halfHeight := (float64(scene.HumanoidBillboardAnchorY) + 20) / 2 * scale * cursorSnapTriggerScale
	return math.Hypot(halfWidth, halfHeight)
}

func groundItemCursorSnapRadius(scale float64) float64 {
	scale = actors.NormalizePickScale(scale)
	halfWidth := 18 * scale * cursorSnapTriggerScale
	halfHeight := 20 * scale * cursorSnapTriggerScale
	return math.Hypot(halfWidth, halfHeight)
}

func (m *WorldMode) cursorActorSpriteCenter(ctx client.Context, projection scene.Projection, actor world.Actor, point scene.Point2, scale float64, now time.Time) (float64, float64, bool) {
	view := m.actors().NonPCSpriteView(ctx, actor)
	if view == nil {
		return 0, 0, false
	}
	state := m.actors().NonPCSpriteState(actor, now)
	state.CameraYaw = projection.CameraYaw
	billboard, ok := sprites.BillboardForState(view, state, now)
	if !ok {
		return 0, 0, false
	}
	return spriteBillboardScreenCenter(billboard, point, scale)
}

func spriteBillboardScreenCenter(billboard *scene.Billboard, point scene.Point2, scale float64) (float64, float64, bool) {
	if billboard == nil || billboard.Image == nil {
		return 0, 0, false
	}
	bounds := billboard.Image.Bounds()
	if bounds.Empty() {
		return 0, 0, false
	}
	scale = actors.NormalizePickScale(scale)
	x := float64(point.X) + (float64(bounds.Dx())/2-billboard.AnchorX)*scale
	y := float64(point.Y) + (float64(bounds.Dy())/2-billboard.AnchorY)*scale
	return x, y, true
}

func cursorSnapTargets(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapTargets
	}
	return ctx.Config.Gameplay.SnapTargets
}

func cursorSnapItems(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapItems
	}
	return ctx.Config.Gameplay.SnapItems
}

func uiCursorAction(ctx client.Context) (int, bool) {
	if ctx.UIApp != nil && ctx.UIApp.Cursor() == widget.CursorPointer {
		return cursorActionClick, true
	}
	if uiPointerBlocked(ctx) {
		return cursorActionDefault, true
	}
	return 0, false
}

func uiPointerBlocked(ctx client.Context) bool {
	if ctx.Input == nil || ctx.UIManager == nil {
		return false
	}
	blocker, ok := ctx.UIManager.(interface {
		PointerBlocked(x, y int) bool
	})
	return ok && blocker.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY)
}

func (m *WorldMode) mapPointerBlocked(ctx client.Context) bool {
	// The pet slot machine and NPC cut-ins are drawn outside the gogpu UI
	// manager, so include them explicitly when gating map interactions.
	return m.petSystem.SlotMachine.Active || m.npc().CutinPointerBlocked(ctx) || uiPointerBlocked(ctx)
}

func hoveredCursorActor(ctx client.Context, projection scene.Projection, mouseX, mouseY int, now time.Time, deadActors map[uint32]time.Time) (world.Actor, bool) {
	if ctx.World == nil {
		return world.Actor{}, false
	}
	bestDistance := math.Inf(1)
	var best world.Actor
	for _, actor := range ctx.World.Actors {
		if _, dead := deadActors[actor.ID]; dead {
			continue
		}
		if scene.ActorHasStealth(actor) || scene.IsLocalActor(ctx, actor.ID) || actor.ID == 0 {
			continue
		}
		if actor.Job == scene.ActorJobClearNPC || actor.Job == scene.ActorJobHiddenWarpNPC {
			continue
		}
		actorX, actorY := scene.ActorRenderPosition(actor, now)
		terrainZ := scene.TerrainHeightAt(ctx.World, actorX, actorY)
		point := projection.Project(scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		scale := scene.ActorBillboardScreenScale(projection, scene.CellCenter(actorX), scene.CellCenter(actorY), terrainZ)
		if !actors.PointInActorPickBounds(float64(mouseX), float64(mouseY), float64(point.X), float64(point.Y), scale*inputPickMultiplier(ctx)) {
			continue
		}
		dx := float64(point.X) - float64(mouseX)
		dy := float64(point.Y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = actor
		}
	}
	return best, bestDistance < math.Inf(1)
}

func cursorInfo(action int) cursorActionInfo {
	if info, ok := cursorActionInfos[action]; ok {
		return info
	}
	return cursorActionInfos[cursorActionDefault]
}

func (s *roCursorState) fallbackTexture() *render.Image {
	if s.fallback != nil {
		return s.fallback
	}
	const width = 18
	const height = 24
	mask := []string{
		"X.................",
		"XX................",
		"XOX...............",
		"XOOX..............",
		"XOOOX.............",
		"XOOOOX............",
		"XOOOOOX...........",
		"XOOOOOOX..........",
		"XOOOOOOOX.........",
		"XOOOOOOOOX........",
		"XOOOOOOOOOX.......",
		"XOOOOOOOOOOX......",
		"XOOOOXXXXXXXX.....",
		"XOOXOX............",
		"XOX..OX...........",
		"XX....OX..........",
		"X......OX.........",
		"........X.........",
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y, row := range mask {
		for x, ch := range row {
			switch ch {
			case 'X':
				img.SetRGBA(x, y, color.RGBA{A: 255})
			case 'O':
				img.SetRGBA(x, y, color.RGBA{R: 246, G: 246, B: 246, A: 255})
			}
		}
	}
	s.fallback = render.NewImageFromImage(img)
	return s.fallback
}

func drawFallbackROCursor(screen *render.Frame, img *render.Image, mouseX, mouseY int) {
	if img == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(mouseX), float64(mouseY))
	opts.Filter = scene.SpriteDrawFilter()
	screen.DrawImage(img, &opts)
}

func (m *WorldMode) drawTileCursor(screen *render.Frame, ctx client.Context, projection scene.Projection) {
	if ctx.Input == nil || ctx.World == nil || ctx.World.GAT == nil {
		return
	}
	if m.mapPointerBlocked(ctx) {
		return
	}
	x, y, ok := m.hoveredWalkCell(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY)
	if !ok {
		return
	}
	m.mapScene.DrawTileCursor(screen, ctx.Resources, ctx.World.GAT, projection, x, y)
}
