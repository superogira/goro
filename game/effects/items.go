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

func (m *System) DrawBerserkPotion(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	origin := EffectWorldAnchor(ctx, effect, now)
	if texture := m.EffectFileTexture(ctx.Resources, component.TextureFile); texture != nil && frame < 50 {
		n := float64(frame + 1)
		height := (10 + n + .01*n*(n+1)/2) / 5
		tint := color.RGBA{255, 255, 255, effectParticleAlpha(frame, 50, 20, 40, 90)}
		var vertices []render.Vertex3D
		var indices []uint16
		for i := 0; i < 10; i++ {
			a, b := scene.DegreesToRadians(float64(i*36)), scene.DegreesToRadians(float64((i+1)*36))
			left := scene.Add3(origin, scene.Point3{X: .9 * math.Cos(a), Z: .9 * math.Sin(a)})
			right := scene.Add3(origin, scene.Point3{X: .9 * math.Cos(b), Z: .9 * math.Sin(b)})
			up := scene.Point3{Y: height}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{scene.Add3(left, up), scene.Add3(right, up), right, left}, tint)
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for spawn := 16; spawn < 110 && spawn <= frame; spawn += 4 {
		age := frame - spawn
		if age >= 40 {
			continue
		}
		salt := spawn * 4
		angle := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt, 360)))
		radius := float64(2+HitEffectRandom(effect, salt+1, 7)) / 5
		speed := float64(20+HitEffectRandom(effect, salt+2, 50)) / (60 * 5)
		center := scene.Add3(origin, scene.Point3{X: radius * math.Sin(angle), Y: speed * float64(age+1), Z: radius * math.Cos(angle)})
		up := scene.Point3{Y: float64(30+HitEffectRandom(effect, salt+3, 60)) / 50}
		tint := color.RGBA{240, 0, 0, effectParticleAlpha(age, 40, 20, 20, 180)}
		for _, side := range []scene.Point3{{X: .04}, {Z: .04}} {
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				scene.Add3(scene.Add3(center, side), up), scene.Sub3(scene.Add3(center, side), up), scene.Sub3(scene.Sub3(center, side), up), scene.Add3(scene.Sub3(center, side), up),
			}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
