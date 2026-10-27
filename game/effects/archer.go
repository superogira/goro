package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

const ArrowShowerReleaseDelay = 18 * 24 * time.Millisecond

func (m *System) AddArrowShower(ctx client.Context, sourceID uint32, x, y float64, ground bool, now time.Time) {
	if !m.AddAt(ctx, ArrowShower, sourceID, now.Add(ArrowShowerReleaseDelay)) {
		return
	}
	effect := &m.Active[len(m.Active)-1]
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, now)
	effect.HasActorTransform = true
	effect.ArrowTargets = new([9]scene.Point3)
	// LaunchSkillSpecialEffect aims at the victim and its eight neighbouring
	// cells. ZC_NOTIFY_GROUNDSKILL instead aims the first arrow at the caster.
	for i, offset := range [9][2]int{{0, 0}, {1, 0}, {1, 1}, {1, -1}, {-1, 0}, {-1, 1}, {-1, -1}, {0, 1}, {0, -1}} {
		tx, ty := x, y
		if i > 0 {
			tx, ty = math.Floor(scene.CellCenter(x))+float64(offset[0]), math.Floor(scene.CellCenter(y))+float64(offset[1])
		}
		target := scene.Point3{X: scene.CellCenter(tx), Y: scene.TerrainHeightAt(ctx.World, tx, ty) + .07, Z: scene.CellCenter(ty)}
		if i == 0 && ground {
			target = effect.ActorOrigin
		}
		// CArrowEffect's positional constructor and first update each raise
		// the endpoint by ten native units; the source is raised only once.
		target.Y += 4
		effect.ArrowTargets[i] = target
	}
	effect.ActorOrigin.Y += 2
}

func ArrowShowerPosition(origin, target scene.Point3, elapsed time.Duration) (scene.Point3, bool) {
	delta := scene.Sub3(target, origin)
	distance := math.Sqrt(scene.Dot3(delta, delta))
	flight := float64(scene.ReferenceBowArrowFlightDuration) * math.Min(1, distance/8)
	if elapsed < 0 || flight <= 0 || float64(elapsed) >= flight {
		return scene.Point3{}, false
	}
	return scene.Add3(origin, scene.Mul3(delta, float64(elapsed)/flight)), true
}

func (m *System) DrawArrowShower(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if effect.ArrowTargets == nil || now.Before(effect.Starts) {
		return
	}
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return
	}
	key := scene.SpriteBillboardKey{IgnoreLayerAngles: true}
	billboard, ok := view.Billboards[key]
	if !ok {
		billboard, ok = scene.ComposeSingleSpriteBillboardWithOptions(view, view.Act.Actions[0].Animations[0], true)
		if !ok {
			return
		}
		view.Billboards[key] = billboard
	}
	for _, target := range effect.ArrowTargets {
		position, visible := ArrowShowerPosition(effect.ActorOrigin, target, now.Sub(effect.Starts))
		if !visible {
			continue
		}
		delta := scene.Sub3(target, effect.ActorOrigin)
		angle := math.Atan2(delta.Z, delta.X) + math.Pi/2 - scene.DegreesToRadians(projection.CameraYaw)
		scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.X, position.Z, position.Y,
			scene.PixelRatio, angle, 1, 1, color.RGBA{255, 255, 255, 255}, scene.SpriteBillboardTriangleDrawOptions())
	}
}
