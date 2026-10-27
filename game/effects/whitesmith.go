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

func (m *System) drawCartTermination(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts)/db.EffectFrameDuration) - 30
	if frame < 0 {
		return
	}
	n := frame + 1
	alpha := max(0, 230-6*max(0, n-25))
	if alpha == 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	center := EffectWorldAnchor(ctx, effect, now)
	right, up, _, ok := projection.BillboardBasis(center.X, center.Z, center.Y)
	if texture == nil || !ok {
		return
	}
	// Four ParticleSpread primitives, each with four groups of three lights.
	distance := (float64(min(n, 26)) + 9*(1-math.Pow(.9, float64(max(0, n-26))))) / 5
	var vertices []render.Vertex3D
	var indices []uint16
	for group := 0; group < 16; group++ {
		salt := group * 8
		half := (2.5 + .1*float64(HitEffectRandom(effect, salt, 6))) / (5 * math.Sqrt2)
		pink := uint8(200 - 40*HitEffectRandom(effect, salt+7, 3))
		r, u := scene.Mul3(right, half), scene.Mul3(up, half)
		for i := 0; i < 3; i++ {
			yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1+i*2, 360)))
			rise := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2+i*2, 180)))
			offset := scene.Point3{X: distance * math.Cos(rise) * math.Sin(yaw), Y: distance * math.Sin(rise), Z: distance * math.Cos(rise) * math.Cos(yaw)}
			if offset.Y < half*up.Y {
				continue
			}
			position := scene.Add3(center, offset)
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				scene.Add3(scene.Sub3(position, r), u), scene.Add3(scene.Add3(position, r), u), scene.Sub3(scene.Add3(position, r), u), scene.Sub3(scene.Sub3(position, r), u),
			}, color.RGBA{255, pink, pink, uint8(alpha)})
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
