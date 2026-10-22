package effects

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func bashFlash(effect Instance, index, frame int) (angle, halfAngle, radius float64) {
	n := float64(frame + 1)
	speed := float64(10+HitEffectRandom(effect, index*5+1, 60)) / 10
	angle = float64(HitEffectRandom(effect, index*5, 360)) + speed*n - speed/60*n*(n+1)/2
	halfAngle = float64(5+HitEffectRandom(effect, index*5+2, 25)) / 10
	radius = float64(20+HitEffectRandom(effect, index*5+3, 40)) + float64(20+HitEffectRandom(effect, index*5+4, 30))/10*n
	return
}

func (m *System) drawBashEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	center, ok := projection.ProjectPoint(x, y, z)
	if !ok {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	center.Y -= 40
	options := &render.DrawTrianglesOptions{Filter: render.FilterLinear, Address: render.AddressRepeat}
	vertex := func(angle, radius float64, u, v, alpha float32) render.Vertex {
		return render.Vertex{
			DstX: center.X + float32(math.Cos(angle)*radius),
			DstY: center.Y - float32(math.Sin(angle)*radius),
			SrcX: u, SrcY: v, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: alpha,
		}
	}
	// Native PP_2D primitives overlay actors and geometry at a fixed pixel size.
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		vertices := make([]render.Vertex, 0, 30)
		indices := make([]uint16, 0, 30)
		w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
		alpha := float32(effectParticleAlpha(frame, 40, 6, 30, 170)) / 255
		for i := 0; i < 10; i++ {
			u := float32(i%4) * w / 4
			base := uint16(len(vertices))
			vertices = append(vertices,
				vertex(0, 0, u, h, alpha),
				vertex(float64(i+1)*math.Pi/5, 100, u+w/4, 0, alpha),
				vertex(float64(i)*math.Pi/5, 100, u, 0, alpha))
			indices = append(indices, base, base+1, base+2)
		}
		screen.DrawTrianglesOwned(vertices, indices, texture, options)
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	vertices := make([]render.Vertex, 0, 60)
	indices := make([]uint16, 0, 60)
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	alpha := float32(effectParticleAlpha(frame, 40, 10, 27, 200)) / 255
	for i := 0; i < 20; i++ {
		angle, halfAngle, radius := bashFlash(effect, i, frame)
		base := uint16(len(vertices))
		// The native renderer indexes integer-degree sine/cosine tables.
		vertices = append(vertices,
			vertex(0, 0, w/2, 0, alpha),
			vertex(scene.DegreesToRadians(float64(int(angle-halfAngle))), radius, w, h, alpha),
			vertex(scene.DegreesToRadians(float64(int(angle+halfAngle))), radius, 0, h, alpha))
		indices = append(indices, base, base+1, base+2)
	}
	screen.DrawTrianglesOwned(vertices, indices, texture, options)
}

func bashHitParticle(effect Instance, index, frame int) effectScreenParticle {
	life := 10 + HitEffectRandom(effect, index*6, 20)
	if frame < 0 || frame >= life {
		return effectScreenParticle{}
	}
	n := float64(frame + 1)
	axis := scene.DegreesToRadians(float64(index * 45))
	angle := scene.DegreesToRadians(float64(index*45 - 15 + HitEffectRandom(effect, index*6+1, 30)))
	speed := float64(5+HitEffectRandom(effect, index*6+2, 45)) / 10
	distance := speed*n - speed/float64(2*life)*n*(n+1)/2
	halfWidth := float64(5+HitEffectRandom(effect, index*6+3, 15)) * (1 - n/float64(life))
	halfHeight := float64(20+HitEffectRandom(effect, index*6+4, 20)) + 1.5*n + .25*n*(n+1)/2
	radius := float64(HitEffectRandom(effect, index*6+5, 5))
	// SIZEUP anchors the bottom edge, so shift the centered quad outward.
	return effectScreenParticle{
		x:         math.Sin(axis)*radius + math.Sin(angle)*(distance+halfHeight),
		Y:         -math.Cos(axis)*radius - math.Cos(angle)*(distance+halfHeight) - 20,
		halfWidth: halfWidth, halfHeight: halfHeight, angle: angle,
		Alpha: uint8(254 * math.Min(1, n/8)),
	}
}

func (m *System) drawBashHitEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	origin, ok := projection.ProjectPoint(x, y, z)
	if !ok {
		return
	}
	textures := [2]*render.Image{m.EffectFileTexture(ctx.Resources, component.TextureFile), m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for i := 0; i < 8; i++ {
		drawEffectScreenParticle(screen, textures[i%2], origin, bashHitParticle(effect, i, frame))
	}
}

func endureStreak(effect Instance, birth, frame int) effectScreenParticle {
	age := frame - birth
	// The parent increments its tick before ProcessAlpha runs. A new primitive
	// with more remaining life is immediately faded from zero and stays invisible.
	if birth < 0 || birth >= 50 || 40 > 80-birth || age < 0 || age >= 40 || frame >= 80 {
		return effectScreenParticle{}
	}
	n := float64(age + 1)
	angle := scene.DegreesToRadians(float64(HitEffectRandom(effect, birth*4, 360)))
	radius := float64(100+HitEffectRandom(effect, birth*4+3, 40)) - 4*n + .1*n*(n+1)/2
	return effectScreenParticle{
		x: math.Sin(angle) * radius, Y: -math.Cos(angle)*radius - 20,
		halfWidth:  float64(60+HitEffectRandom(effect, birth*4+1, 100)) / 100,
		halfHeight: float64(30 + HitEffectRandom(effect, birth*4+2, 20)),
		angle:      angle,
		Alpha:      effectParticleAlpha(age, 40, 20, 27, 254),
	}
}

func (m *System) drawEndureEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	origin, ok := projection.ProjectPoint(x, y, z)
	if !ok {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		n := float64(min(frame+1, 45))
		halfSize := 150 - 5*n + (5.0/45)*n*(n+1)/2
		drawEffectScreenParticle(screen, texture, origin, effectScreenParticle{
			Y: -20, halfWidth: halfSize, halfHeight: halfSize,
			Alpha: effectParticleAlpha(frame, 80, 20, 54, 254),
		})
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	for birth := max(0, frame-39); birth <= min(frame, 49); birth++ {
		drawEffectScreenParticle(screen, texture, origin, endureStreak(effect, birth, frame))
	}
}

func magnumRadius(speed float64, frame int) float64 {
	n := float64(frame + 1)
	return (speed*n - speed/60*n*(n+1)/2) / 5
}

func (m *System) drawMagnumBreakEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	alpha := effectParticleAlpha(frame, 30, 15, 15, 254)
	if alpha == 0 {
		return
	}
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0]); texture != nil {
		vertices, indices := effectSphereMesh(texture, effect.ActorOrigin, magnumRadius(1.15, frame), 3*float64(frame+1), alpha)
		screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil {
		radius := magnumRadius(1.75, frame)
		vertices, indices := groundEffectRingMesh(texture, effect.ActorOrigin, math.Max(0, radius-2.4), radius, alpha)
		options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
		options.DepthTest = false
		// The 2008 client deliberately launches two identical ground rings.
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
	}
}
