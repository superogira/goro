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

func (m *System) drawChainCrush(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 20*min(n, 10) - 2*max(0, n-20)
	if n <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	s, c := math.Sincos(math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X))
	point := func(along, across float64) scene.Point3 {
		return scene.Add3(effect.ActorOrigin, scene.Point3{X: (along*c - across*s) / 5, Z: (along*s + across*c) / 5, Y: 1.2})
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 32; i++ {
		salt := i * 6
		speed := 1 + .1*float64(HitEffectRandom(effect, salt, 11))
		phase := math.Mod(float64(HitEffectRandom(effect, salt+1, 200))+speed*float64(n)+.025*float64(n*(n+1)), 200)
		length := float64(40 + HitEffectRandom(effect, salt+2, 31))
		offset := float64(15 + HitEffectRandom(effect, salt+3, 11))
		if i%2 == 0 {
			offset = -offset
		}
		width := .5 + .1*float64(HitEffectRandom(effect, salt+4, 6))
		front, back := phase-100+length, phase-100
		appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
			point(front, offset+width), point(back, offset+width), point(back, offset-width), point(front, offset-width),
		}, color.RGBA{255, 255, uint8(225 - 20*HitEffectRandom(effect, salt+5, 3)), uint8(alpha)})
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}
