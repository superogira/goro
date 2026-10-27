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

func (m *System) drawHunterEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	switch effect.EffectID {
	case Detecting:
		if frame >= 57 {
			return
		}
		// Prim3DTexture replaces the initial growth speed on tick 17.
		n := float64(min(frame+1, 17))
		size := 1.5*n + (1.5/14)*n*(n+1)/2
		n = float64(max(0, frame-16))
		size += .1*n + (.1/30)*n*(n+1)/2
		texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
		drawGroundTextureQuad(screen, texture, origin.X, origin.Z, origin.Y+1, size*2/5, size*2/5, 0,
			color.RGBA{255, 255, 255, effectParticleAlpha(frame, 57, 1, 40, 254)}, false)
	case RemoveTrap:
		m.drawRemoveTrap(screen, ctx, projection, component, effect, origin, frame)
	case Flasher:
		m.drawFlasher(screen, ctx, projection, component, effect, origin, frame)
	case BlitzBeat:
		m.drawBlitzBeat(screen, ctx, component, effect, origin, frame)
	}
}

func removeTrapParticle(effect Instance, index, age int) effectParticle {
	if age < 0 || age >= 50 {
		return effectParticle{}
	}
	n := float64(age + 1)
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, index*2, 360)))
	accel := .08 + float64(HitEffectRandom(effect, index*2+1, 10))/1000
	return effectParticle{
		Position: scene.Point3{X: math.Sin(yaw) * .2 * n / 5, Z: math.Cos(yaw) * .2 * n / 5, Y: (2*n - accel*n*(n+1)/2) / 5},
		size:     .8,
		Alpha:    float64(effectParticleAlpha(age, 50, 1, 35, 254)) / 255,
	}
}

func (m *System) drawRemoveTrap(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, frame int) {
	for i := 0; i < 24; i++ {
		age := frame - (i/12)*7
		if age < 0 || age >= 50 {
			continue
		}
		p := removeTrapParticle(effect, i, age)
		m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, scene.Add3(origin, p.Position),
			age/4+1, p.size, 0, color.RGBA{255, 255, 255, uint8(p.Alpha * 255)})
	}
}

func (m *System) drawFlasher(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, frame int) {
	if frame < 0 || frame >= 70 {
		return
	}
	alpha := effectParticleAlpha(frame, 70, 6, 60, 200)
	if frame >= 40 && frame < 43 {
		alpha = uint8(math.Min(200, 120+float64(frame-39)*(200.0/6)))
	}
	drawTexturedEffectBillboardRotatedXY(screen, projection, m.EffectFileTexture(ctx.Resources, component.TextureFile),
		origin.X, origin.Z, origin.Y+2, 8, 8, 0, color.RGBA{255, 255, 255, alpha}, true)
	center, ok := projection.ProjectPoint(origin.X, origin.Z, origin.Y)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if !ok || texture == nil {
		return
	}
	center.Y -= 40
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	n := float64(frame + 1)
	a := float32(effectParticleAlpha(frame, 70, 10, 47, 200)) / 255
	vertex := func(angle, radius float64, u, v float32) render.Vertex {
		angle = scene.DegreesToRadians(angle)
		return render.Vertex{DstX: center.X + float32(math.Cos(angle)*radius), DstY: center.Y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: a}
	}
	vertices := make([]render.Vertex, 0, 60)
	indices := make([]uint16, 0, 60)
	for i := 0; i < 20; i++ {
		speed := float64(10+HitEffectRandom(effect, i*5+1, 60)) / 10
		angle := float64(HitEffectRandom(effect, i*5, 360)) + speed*n - speed/105*n*(n+1)/2
		halfAngle := float64(5+HitEffectRandom(effect, i*5+2, 25)) / 10
		radius := float64(20+HitEffectRandom(effect, i*5+3, 40)) + float64(50+HitEffectRandom(effect, i*5+4, 30))/10*n
		base := uint16(len(vertices))
		vertices = append(vertices, vertex(angle, 0, w/2, h), vertex(angle-halfAngle, radius, 0, 0), vertex(angle+halfAngle, radius, w, 0))
		indices = append(indices, base, base+1, base+2)
	}
	screen.DrawTrianglesOwned(vertices, indices, texture, &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat})
}

func blitzStreakDistance(frame int) float64 {
	n := float64(min(frame+1, 10))
	distance := 15 - 1.2*n - .1*n*(n+1)/2
	n = float64(max(0, frame-9))
	return (distance - .3*n + .001*n*(n+1)/2) / 5
}

func (m *System) drawBlitzBeat(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, frame int) {
	// The attacked-effect table launches one EF_BLITZBEAT, with count=1,
	// on the first hit. Further damage ticks do not repeat it.
	yaw := 180 + 45*float64(scene.NormalizeDirectionIndex(effect.ActorDirection))
	if frame < 20 {
		texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
		if texture == nil {
			return
		}
		forward := RotateEffectCylinderVector(scene.Point3{Z: 1}, 0, yaw, 0)
		right := RotateEffectCylinderVector(scene.Point3{X: .2 / 5}, 0, yaw, 0)
		n := float64(frame + 1)
		vertices := make([]render.Vertex3D, 0, 80)
		indices := make([]uint16, 0, 120)
		for i := 0; i < 10; i++ {
			angle := scene.DegreesToRadians(float64(HitEffectRandom(effect, i*2, 360)))
			center := scene.Add3(origin, scene.Point3{X: .7 * math.Sin(angle), Y: 1.4, Z: -.7 * math.Cos(angle)})
			center = scene.Add3(center, scene.Mul3(forward, blitzStreakDistance(frame)))
			length := (float64(20+HitEffectRandom(effect, i*2+1, 40))/10 + .1*n - .01*n*(n+1)/2) / 5
			along := scene.Mul3(forward, length)
			for _, side := range []scene.Point3{right, {Y: .2 / 5}} {
				appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
					scene.Sub3(scene.Add3(center, along), side), scene.Sub3(scene.Sub3(center, along), side),
					scene.Add3(scene.Sub3(center, along), side), scene.Add3(scene.Add3(center, along), side),
				}, color.RGBA{255, 255, 255, effectParticleAlpha(frame, 20, 2, 18, 250)})
			}
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, true))
	}
	age := frame - 36
	if age < 0 || age >= 12 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	// Convert native downward Y to world upward Y after the X/Y rotations.
	rotate := func(p scene.Point3) scene.Point3 {
		p = RotateEffectCylinderVector(p, -90, yaw, 0)
		p.Y = -p.Y
		return p
	}
	n := float64(age + 1)
	distance := .4*n - (.4/24)*n*(n+1)/2
	center := scene.Add3(origin, scene.Mul3(rotate(scene.Point3{Y: -2 - distance, Z: 1}), .2))
	center.X += float64((1+HitEffectRandom(effect, 50, 10))/5) / 5
	center.Y += float64(12-(1+HitEffectRandom(effect, 51, 20))/2) / 5
	drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, center.X, center.Z, center.Y,
		.5/5, 1.5/5, 3.5/5, color.RGBA{255, 255, 255, effectParticleAlpha(age, 12, 1, 6, 254)}, 10,
		rotate(scene.Point3{X: 1}), rotate(scene.Point3{Z: 1}), rotate(scene.Point3{Y: -1}), 2.5, render.BlendSourceOver)
}
