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

func stealParticleAt(effect Instance, index, frame int) effectParticle {
	if frame < 0 || frame >= 50 {
		return effectParticle{}
	}
	salt := index * 6
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt, 360)))
	pitch := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 100) - 50))
	radius := float64(4+HitEffectRandom(effect, salt+2, 4)) / 5
	position := rotateModelPointX(rotateModelPointY(scene.Point3{Z: radius}, yaw), pitch)
	position.Y = 2 - position.Y
	n := float64(frame + 1)
	speed := float64(50+HitEffectRandom(effect, salt+3, 50)) / 500
	gravity := float64(60+HitEffectRandom(effect, salt+4, 90)) / 500
	distance := speed * (n - n*(n+1)/200)
	position.X += math.Sin(yaw) * distance
	position.Z += math.Cos(yaw) * distance
	position.Y += gravity * (n - n*(n+1)/50)
	size := float64(100+HitEffectRandom(effect, salt+5, 100)) / 100 * (1 - n/50)
	return effectParticle{Position: position, size: size, Alpha: math.Min(1, (50-n)/16) * 254 / 255}
}

func (m *System) drawStealEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for i := 0; i < 10; i++ {
		particle := stealParticleAt(effect, i, frame)
		m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, scene.Add3(effect.ActorOrigin, particle.Position),
			frame/4+1, particle.size, 0, color.RGBA{255, 255, 255, uint8(particle.Alpha * 255)})
	}
}

func sprinkleSandParticleAt(effect Instance, index, frame int) effectParticle {
	salt := index * 7
	n := frame + 1 - (10 - HitEffectRandom(effect, salt, 10))
	alpha := 15*scene.ClampInt(n-1, 0, 9) - 3*max(0, n-10)
	if n <= 0 || alpha <= 0 {
		return effectParticle{}
	}
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 360)))
	speed := float64(HitEffectRandom(effect, salt+2, 10)) / 500
	rise := float64(20+HitEffectRandom(effect, salt+3, 40)) / 500
	return effectParticle{
		Position: scene.Point3{
			X: float64(HitEffectRandom(effect, salt+4, 9)-4)/5 + float64(n)*speed*math.Cos(yaw),
			Y: float64(n) * rise,
			Z: float64(HitEffectRandom(effect, salt+5, 9)-4)/5 + float64(n)*speed*math.Sin(yaw),
		},
		size:  (.1 + .03*float64(HitEffectRandom(effect, salt+6, 16))) / 5 * math.Sqrt2,
		Angle: scene.DegreesToRadians(float64(-5*n)) - 3*math.Pi/4,
		Alpha: float64(alpha) / 255,
	}
}

func (m *System) drawSprinkleSandEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	// PARTICLE_UP launches twenty groups of four gold particles on the victim.
	for i := 0; i < 80; i++ {
		particle := sprinkleSandParticleAt(effect, i, frame)
		if particle.Alpha <= 0 {
			continue
		}
		p := scene.Add3(effect.ActorOrigin, particle.Position)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, p.X, p.Z, p.Y,
			particle.size, particle.size, particle.Angle, color.RGBA{250, 250, 150, uint8(particle.Alpha * 255)}, true)
	}
}
