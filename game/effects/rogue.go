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

func rogueBurstParticle(effect Instance, index, frame int) ([4]scene.Point3, color.RGBA) {
	salt := index * 5
	delay, rise, tint := 70, 15, color.RGBA{100, 150, 255, 0}
	if effect.EffectID == RogueCoin {
		delay, rise, tint = 18, 25, color.RGBA{250, 250, 155, 0}
	}
	pitch := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 360)))
	yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2, 360)))
	var corners [4]scene.Point3
	if effect.EffectID == TeiHit3 {
		age := frame + 1 - 20 - HitEffectRandom(effect, salt, 10)
		alpha := 15*scene.ClampInt(age-10, 0, 10) - 5*max(0, age-20)
		if alpha <= 0 {
			return corners, tint
		}
		tint = color.RGBA{200, 200, 255, uint8(alpha)}
		distance := float64(-10 + age*2)
		for j, p := range [4][2]float64{{16, -1}, {16, 1}, {-16, 1}, {-16, -1}} {
			corners[j] = effectParticlePoint(distance+p[0], p[1], pitch, yaw)
		}
		return corners, tint
	}
	age := frame + 1 - delay - HitEffectRandom(effect, salt, 30)
	alpha := rise*scene.ClampInt(age, 0, 10) - 7*max(0, age-25)
	if alpha <= 0 {
		return corners, tint
	}
	tint.A = uint8(alpha)
	center := effectParticlePoint(12+1.5*float64(age), 0, pitch, yaw)
	roll := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+3, 360)))
	spin := scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+4, 360) + 5*(frame+1)))
	for j := range corners {
		corners[j] = scene.Add3(center, effectParticlePoint(3.5, 0, roll+float64(j)*math.Pi/2, spin))
	}
	return corners, tint
}

func (m *System) drawRogueBurst(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	count := 80
	if effect.EffectID == RogueCoin {
		count = 120
	}
	vertices := make([]render.Vertex3D, 0, count*4)
	indices := make([]uint16, 0, count*6)
	for i := 0; i < count; i++ {
		corners, tint := rogueBurstParticle(effect, i, frame)
		if tint.A == 0 {
			continue
		}
		// RenderTeiRect maps its first edge to the bottom of the texture.
		for j := range corners {
			corners[j] = scene.Add3(corners[j], effect.ActorOrigin)
		}
		appendEffectQuad(&vertices, &indices, texture,
			[4]scene.Point3{corners[3], corners[2], corners[1], corners[0]}, tint)
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(effect.EffectID != TeiHit3, false))
	}
}
