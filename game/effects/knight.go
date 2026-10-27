package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func (m *System) drawBowlingBashEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	origin := effect.ActorOrigin
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		for _, birth := range []int{0, 5} {
			age, life := frame-birth, 20-birth
			if age < 0 || age >= life {
				continue
			}
			n := float64(age + 1)
			growth := (.5*n - .03*n*(n+1)/2) / 5
			alpha := float64(240-7*birth) * math.Min(1, float64(life-age-1)/float64(life/2))
			yaw := -45*float64(scene.NormalizeDirectionIndex(effect.ActorDirection)) + float64(20*birth)
			right := RotateEffectCylinderVector(scene.Point3{X: 1}, 0, yaw, 0)
			depth := RotateEffectCylinderVector(scene.Point3{Z: 1}, 0, yaw, 0)
			drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, origin.X, origin.Z, origin.Y,
				.6+growth, 1.6+growth, .7, color.RGBA{255, 255, 255, uint8(alpha)},
				10, right, depth, scene.Point3{Y: 1}, 2.5, render.BlendSourceOver)
		}
	}
	if frame < 0 || frame >= 50 {
		return
	}
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]); texture != nil {
		n := float64(min(frame+1, 20))
		radius := (8 + .7*n - (.7/30)*n*(n+1)/2 + .08*float64(max(0, frame+1-20))) / 5
		alpha := uint8(45 * math.Min(1, float64(49-frame)/15))
		vertices, indices := groundEffectRingMesh(texture, origin, radius-1, radius, alpha)
		screen.DrawTriangles3DOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
	}
}

func initSpearBoomerang(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	origin := destination
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		origin = EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, effect.Starts)
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ProjectileDirection = scene.Sub3(destination, origin)
	effect.ProjectileDirection.Y = 0
	origin.Y += 3 // Fifteen native units above the caster.
	effect.ActorOrigin = origin
	effect.HasActorTransform = true
}

func SpearBoomerangPosition(effect Instance, age int) (scene.Point3, bool) {
	if age < 0 || age > 15 {
		return scene.Point3{}, false
	}
	delta := effect.ProjectileDirection
	fraction := float64(age+1) / 15
	// Prim3DParticle stops when it comes within three native units of the target.
	if math.Hypot(delta.X, delta.Z)*(1-fraction) <= .6 {
		return scene.Point3{}, false
	}
	return scene.Add3(effect.ActorOrigin, scene.Mul3(delta, fraction)), true
}

func (m *System) drawSpearBoomerangEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	delta := effect.ProjectileDirection
	angle := math.Atan2(delta.Z, delta.X) - math.Pi/2 + scene.DegreesToRadians(projection.CameraYaw)
	options := scene.SpriteBillboardTriangleDrawOptions()
	for i, alpha := range []uint8{254, 180, 130, 80} {
		age := frame - i*4
		position, visible := SpearBoomerangPosition(effect, age)
		if !visible {
			continue
		}
		motion := (age + 1) % len(view.Act.Actions[0].Animations)
		key := scene.SpriteBillboardKey{Motion: motion}
		billboard, ok := view.Billboards[key]
		if !ok {
			billboard, ok = scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[0].Animations[motion])
			if !ok {
				continue
			}
			view.Billboards[key] = billboard
		}
		scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard, position.X, position.Z, position.Y,
			scene.PixelRatio, angle, float64(alpha)/255, 1, color.RGBA{255, 255, 255, 255}, options)
	}
}
