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

func (m *System) drawHPConversion(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame >= 64 {
		return
	}
	// EnergyDrain3 emits three PP_3DPARTICLESPLINE souls on each of five ticks.
	for i := 0; i < 15; i++ {
		age := frame - i/3
		if age < 0 || age >= 60 {
			continue
		}
		direction := scene.Normalize3(scene.Point3{X: float64(HitEffectRandom(effect, i*5, 7) - 3), Z: float64(HitEffectRandom(effect, i*5+1, 7) - 3)})
		if direction == (scene.Point3{}) {
			direction.Z = 1
		}
		tint := color.RGBA{uint8(160 + HitEffectRandom(effect, i*5+2, 81)), 255, uint8(160 + HitEffectRandom(effect, i*5+3, 81)), 254}
		size := 1 + .1*float64(HitEffectRandom(effect, i*5+4, 6))
		s, c := math.Sincos(scene.DegreesToRadians(float64(-90 + i%3*90)))
		for trail := 0; trail < 7; trail++ {
			n := float64(max(0, age+1-trail))
			p := effect.ActorOrigin
			if n > 0 {
				forward := (2*n - n*(n+1)/30) / 5
				arc := (2*n - n*(n-1)/30) / 5
				p = scene.Add3(p, scene.Mul3(direction, forward))
				p.Y += 2 + arc*c
				p.X += direction.Z * arc * s
				p.Z -= direction.X * arc * s
			}
			tint.A = uint8(254 * (1 - float64(trail)/7))
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, age/4+1, size*(1-float64(trail)/14), 0, tint)
		}
	}
}

func (m *System) drawSoulChangeLink(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 6*min(n, 19) - 3*max(0, n-40)
	if n <= 0 || alpha <= 0 {
		return
	}
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); !ok {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	from := EffectWorldAnchor(ctx, effect, now)
	target := effect
	target.ActorID = effect.TargetID
	to := EffectWorldAnchor(ctx, target, now)
	from.Y, to.Y = from.Y+1.6, to.Y+1.6
	if n < 15 {
		to = scene.Add3(from, scene.Mul3(scene.Sub3(to, from), math.Sin(scene.DegreesToRadians(float64(n*6)))))
	}
	var vertices []render.Vertex3D
	var indices []uint16
	appendEffectTube(&vertices, &indices, texture, from, to, .2/5, 10, color.RGBA{0, 100, 150, uint8(alpha)})
	appendEffectTube(&vertices, &indices, texture, from, to, .4/5, 10, color.RGBA{255, 50, 0, uint8(alpha * 2)})
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
