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

func (m *System) drawSoulBreaker(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame > 61 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	count := 4
	if effect.EffectID == SoulBreaker2 {
		count = 32
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < count; i++ {
		n := frame + 1 - 28 + i%4
		yaw := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X) + scene.DegreesToRadians(float64(HitEffectRandom(effect, i*2, 31)-15))
		if effect.EffectID == SoulBreaker2 {
			n -= HitEffectRandom(effect, 100+i/4, 6)
			yaw = scene.DegreesToRadians(float64(i / 4 * 45))
		}
		alpha := 10*scene.ClampInt(n-1, 0, 9) - 5*max(0, n-10)
		if alpha <= 0 {
			continue
		}
		// PP_SOULBREAKER: acceleration changes after the first ten ticks.
		distance, speed := 0.0, .8/5
		for tick := 1; tick <= n; tick++ {
			if tick > 10 {
				speed *= 1.04
			} else if tick > 1 {
				speed *= 1.02
			}
			distance += speed
		}
		center := scene.Add3(effect.ActorOrigin, scene.Point3{X: distance * math.Cos(yaw), Z: distance * math.Sin(yaw), Y: float64(10-HitEffectRandom(effect, i*2+1, 5)) / 5})
		var corners [4]scene.Point3
		for j := range corners {
			angle := yaw + math.Pi/4 + float64(3-j)*math.Pi/2
			corners[j] = scene.Add3(center, scene.Point3{X: 4 * math.Cos(angle), Z: 4 * math.Sin(angle)})
		}
		// The 2008 renderer uses normal alpha and pale blue, unlike later dhxj.
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{200, 200, 255, uint8(alpha)})
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
	}
}
