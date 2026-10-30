package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

const (
	level99AuraLevel    = 99
	Level99AuraLifetime = 24 * time.Hour
)

var (
	Level99AuraNormalEffects = []int{Level99, Level99Ground, Level99Bubble}
	Level99AuraSimpleEffects = []int{Level99Bubble}
)

func (m *System) SyncLevel99AuraEffects(ctx client.Context, now time.Time) {
	if m == nil || ctx.World == nil {
		return
	}
	if ctx.Session != nil {
		m.syncActorLevel99AuraEffects(ctx, scene.LocalSkillTarget(ctx), ctx.Session.Progress.BaseLevel, true, now)
	}
	for _, actor := range ctx.World.Actors {
		m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, now)
	}
}

func (m *System) SyncActorEntryLevel99Aura(ctx client.Context, entryID uint32) {
	if m == nil || ctx.World == nil || entryID == 0 {
		return
	}
	actor, ok := ctx.World.Actors[entryID]
	if !ok {
		m.RemoveLevel99AuraEffects(entryID)
		return
	}
	m.syncActorLevel99AuraEffects(ctx, actor.ID, actor.Level, actor.HasLevel, time.Now())
}

func (m *System) syncActorLevel99AuraEffects(ctx client.Context, actorID uint32, level int, hasLevel bool, now time.Time) {
	if actorID == 0 {
		return
	}
	if !hasLevel {
		return
	}
	if actor, ok, _ := scene.ActorForCombatID(ctx, actorID); ok && scene.ActorHasStealth(actor) {
		m.RemoveLevel99AuraEffects(actorID)
		return
	}
	var wanted []int
	if level >= level99AuraLevel {
		wanted = level99AuraEffectIDs(ctx)
		for _, effectID := range wanted {
			if !m.HasActive(effectID, actorID, now) {
				m.addLevel99AuraEffect(ctx, effectID, actorID, now)
			}
		}
	}
	for _, effectID := range Level99AuraNormalEffects {
		if !level99AuraWantsEffect(wanted, effectID) {
			m.Remove(effectID, actorID)
		}
	}
}

func (m *System) addLevel99AuraEffect(ctx client.Context, effectID int, actorID uint32, now time.Time) bool {
	x, y, ok := EffectAnchor(ctx, actorID)
	if !ok {
		return false
	}
	return m.AddCellLifetime(ctx, effectID, actorID, x, y, now, Level99AuraLifetime, true)
}

func (m *System) RemoveLevel99AuraEffects(actorID uint32) {
	for _, effectID := range Level99AuraNormalEffects {
		m.Remove(effectID, actorID)
	}
}

func level99AuraEffectIDs(ctx client.Context) []int {
	simple := ctx.Config.Gameplay.SimpleAura
	if ctx.Session != nil {
		simple = ctx.Session.SimpleAura
	}
	if simple || lessEffectsEnabled(ctx) {
		return Level99AuraSimpleEffects
	}
	return Level99AuraNormalEffects
}

func level99AuraWantsEffect(wanted []int, effectID int) bool {
	for _, wantedID := range wanted {
		if wantedID == effectID {
			return true
		}
	}
	return false
}

func (m *System) drawLevel99AuraEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	elapsedFrames := math.Max(0, now.Sub(effect.Starts).Seconds()*60)
	build := math.Sin(math.Min(elapsedFrames, 90) * math.Pi / 180)
	if build <= 0 {
		return
	}
	tint := color.RGBA{R: 100, G: 100, B: 255, A: 120}
	for band := 0; band < 3; band++ {
		drawLevel99AuraBand(screen, m.whitePixel, texture, x, y, z+0.02, band, elapsedFrames, build, tint)
	}
}

func drawLevel99AuraBand(screen *render.Frame, white, texture *render.Image, x, y, z float64, band int, elapsedFrames, build float64, tint color.RGBA) {
	const (
		divisions        = 21
		fullDisplayAngle = 315.0
		gameToWorld      = 0.1 * 2.2
		innerCircleScale = 0.6
	)
	if screen == nil || white == nil {
		return
	}
	source := white
	srcW, srcH := float32(1), float32(1)
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	maxHeight := float64(15-2*band) * gameToWorld
	distance := (3.9 + 0.2*float64(band)) * gameToWorld * innerCircleScale
	riseAngle := float64(55-5*band) * math.Pi / 180
	rotStart := float64(band)*90 + elapsedFrames*float64(band+3)
	basicAngle := fullDisplayAngle / float64(divisions-1)
	cosRise := math.Cos(riseAngle)
	sinRise := math.Sin(riseAngle)
	center := scene.Point3{X: x, Y: z, Z: y}
	vertices := make([]render.Vertex3D, 0, divisions*2)
	indices := make([]uint16, 0, (divisions-1)*6)
	for k := 0; k < divisions; k++ {
		angle := (rotStart + float64(k)*basicAngle) * math.Pi / 180
		cosAngle := math.Cos(angle)
		sinAngle := math.Sin(angle)
		sinLimit := math.Sin((90 + float64(k-10)*9) * math.Pi / 180)
		height := math.Max(0, maxHeight*sinLimit*build)
		radialRise := cosRise * height
		verticalRise := sinRise * height
		u := float32(k) / float32(divisions-1)
		base := scene.Point3{
			X: center.X + distance*cosAngle,
			Y: center.Y,
			Z: center.Z + distance*sinAngle,
		}
		top := scene.Point3{
			X: center.X + (distance+radialRise)*cosAngle,
			Y: center.Y + verticalRise,
			Z: center.Z + (distance+radialRise)*sinAngle,
		}
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(base, scene.UV{U: u, V: 1}, tint, srcW, srcH),
			scene.TexturedSurfaceVertex3D(top, scene.UV{U: u, V: 0}, tint, srcW, srcH),
		)
		if k == divisions-1 {
			continue
		}
		baseIndex := uint16(k * 2)
		indices = append(indices, baseIndex, baseIndex+1, baseIndex+2, baseIndex+1, baseIndex+3, baseIndex+2)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, source, options)
}

func (m *System) drawLevel99BubbleEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	elapsedFrames := math.Max(0, now.Sub(effect.Starts).Seconds()*60)
	size := component.SizeStart
	if size <= 0 {
		size = 2.4 * level99BubbleGameToWorld * 0.6 * 2
	}
	for column := 0; column < level99BubbleColumns; column++ {
		for anchor := 0; anchor < level99BubbleAnchors; anchor++ {
			sample := level99BubbleSample(effect, column, anchor, elapsedFrames)
			if !sample.visible || sample.alpha <= 0 {
				continue
			}
			tint := component.Color
			if tint.A == 0 {
				tint = color.RGBA{R: 80, G: 80, B: 255, A: 250}
			}
			tint.A = uint8(sample.alpha * 255)
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, x+sample.offsetX, y+sample.offsetY, z+component.PosZ+sample.height, size, size, 0, tint, true)
		}
	}
}

const (
	level99BubbleGameToWorld = 0.1 * 2.2
	level99BubbleColumns     = 4
	level99BubbleAnchors     = 4
	level99BubbleSeedMax     = 99.0
	level99BubbleResetY      = -30.0
	level99BubbleRiseSpeed   = 0.15
	level99BubbleDrift       = 0.15
	level99BubblePhaseXRate  = 0.045
	level99BubblePhaseYRate  = 0.052
)

type level99BubbleParticle struct {
	visible bool
	offsetX float64
	offsetY float64
	height  float64
	alpha   float64
}

func level99BubbleSample(effect Instance, column, anchor int, elapsedFrames float64) level99BubbleParticle {
	elapsedFrames = math.Max(0, elapsedFrames)
	salt := column*97 + anchor*31
	seed := deterministicUnit(effect, salt+501) * level99BubbleSeedMax
	cycleSpan := level99BubbleSeedMax - level99BubbleResetY
	cycle := math.Mod(elapsedFrames*level99BubbleRiseSpeed+seed, cycleSpan)
	clientY := level99BubbleSeedMax - cycle
	if clientY > 0 {
		return level99BubbleParticle{}
	}
	heightUnits := -clientY
	visibleFrames := heightUnits / level99BubbleRiseSpeed
	visibleStartFrame := elapsedFrames - visibleFrames
	phaseX := visibleStartFrame*level99BubblePhaseXRate + deterministicUnit(effect, salt+601)*2*math.Pi
	phaseY := visibleStartFrame*level99BubblePhaseYRate + deterministicUnit(effect, salt+701)*2*math.Pi
	signX, signY := level99BubbleDriftSigns(anchor)
	return level99BubbleParticle{
		visible: true,
		offsetX: signX * level99BubbleIntegratedDrift(phaseX, level99BubblePhaseXRate, visibleFrames) *
			level99BubbleGameToWorld,
		offsetY: signY * level99BubbleIntegratedDrift(phaseY, level99BubblePhaseYRate, visibleFrames) *
			level99BubbleGameToWorld,
		height: heightUnits * level99BubbleGameToWorld,
		alpha:  level99BubbleAlpha(clientY),
	}
}

func level99BubbleDriftSigns(anchor int) (float64, float64) {
	switch anchor % level99BubbleAnchors {
	case 1:
		return -1, -1
	case 2:
		return 1, -1
	case 3:
		return -1, 1
	default:
		return 1, 1
	}
}

func level99BubbleIntegratedDrift(phase, rate, frames float64) float64 {
	if frames <= 0 || rate == 0 {
		return 0
	}
	return level99BubbleDrift / rate * (math.Cos(phase) - math.Cos(phase+rate*frames))
}

func level99BubbleAlpha(clientY float64) float64 {
	alpha := 250.0
	if clientY < -20 {
		alpha = 250 + (clientY+20)*25
	}
	return scene.ClampFloat(alpha/255, 0, 250.0/255.0)
}

func (m *System) drawLevel99GroundEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	elapsed := now.Sub(effect.Starts).Seconds()
	sizes := []float64{component.SizeStart, component.SizeEnd}
	for i, size := range sizes {
		if size <= 0 {
			continue
		}
		phase := elapsed*6 + float64(i)*23*math.Pi/180
		size += math.Sin(phase) * 0.08
		drawGroundTextureQuad(screen, texture, x, y, z+component.PosZ+0.01*float64(i), size, size, float64(i)*23*math.Pi/180, color.RGBA{R: 255, G: 255, B: 255, A: 204}, true)
	}
}
