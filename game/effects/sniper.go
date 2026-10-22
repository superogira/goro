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

func SharpShootingLine(effect Instance, index, frame int) (from, to scene.Point3, alpha int) {
	salt := index * 6
	age := frame + 1 - index/4 - 45 - HitEffectRandom(effect, salt, 20)
	alpha = max(0, 2*scene.ClampInt(age, 0, 5)-max(0, age-5))
	angle := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X)
	s, c := math.Sincos(angle)
	back := .15 * float64(HitEffectRandom(effect, salt+1, 100))
	from = scene.Add3(effect.ActorOrigin, scene.Point3{
		X: (.08*float64(HitEffectRandom(effect, salt+2, 100)) - 4 - back*c) / 5,
		Z: (.08*float64(HitEffectRandom(effect, salt+3, 100)) - 4 - back*s) / 5,
		Y: (8 + .04*float64(HitEffectRandom(effect, salt+4, 100))) / 5,
	})
	length := math.Hypot(effect.ProjectileDirection.X, effect.ProjectileDirection.Z) + .12*float64(HitEffectRandom(effect, salt+5, 100))
	to = scene.Add3(from, scene.Point3{X: length * c, Z: length * s})
	return
}

func (m *System) drawSniperEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	n := frame + 1
	var vertices []render.Vertex3D
	var indices []uint16
	switch effect.EffectID {
	case TrueSight:
		alpha := 6*min(n, 10) - max(0, n-11)
		if alpha <= 0 {
			return
		}
		origin := EffectWorldAnchor(ctx, effect, now)
		origin.Y += 2.4
		distance := float64(2+3*n) / 5
		for layer, width := range []float64{2, 5} {
			vertices, indices = nil, nil
			for i := 0; i < 48; i++ {
				yaw := scene.DegreesToRadians(float64(90*(i%4) + 7*(i/4)))
				roll := deterministicAngle(effect, i)
				point := func(angle float64) scene.Point3 {
					r := distance * math.Cos(angle)
					return scene.Add3(origin, scene.Point3{X: r * math.Cos(roll), Y: -r * math.Sin(roll), Z: distance * math.Sin(angle)})
				}
				appendEffectQuad(&vertices, &indices, texture,
					[4]scene.Point3{origin, origin, point(yaw + scene.DegreesToRadians(width)), point(yaw - scene.DegreesToRadians(width))},
					color.RGBA{250, 250, 250, uint8(alpha)})
			}
			screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer == 0, false))
		}
		return
	case Portal4, Portal5:
		alpha := 10*min(n, 11) - 2*max(0, n-20)
		if alpha <= 0 {
			return
		}
		// PP_WIND stays at its creation point, unlike the caster's body flash.
		fullAngle := min(120, 30+3*n)
		step := fullAngle / 21
		segments := (fullAngle - 1) / step
		height, radiusBase, radiusRandom := 5, 4.5, .03
		tint := color.RGBA{255, 255, 255, uint8(alpha)}
		if effect.EffectID == Portal5 {
			height, radiusBase, radiusRandom = 3, 2.5, .01
			tint.B = 170
		}
		for i := 0; i < 4; i++ {
			origin := effect.ActorOrigin
			origin.Y += float64(height+2*i) / 5
			rise := scene.DegreesToRadians(float64(80 + HitEffectRandom(effect, 2*i, 21)))
			radius := (radiusBase + radiusRandom*float64(HitEffectRandom(effect, 2*i+1, 101)) + .1*float64(max(0, n-20))) / 5
			start := len(vertices)
			appendEffectBand(&vertices, &indices, texture, origin, radius,
				scene.DegreesToRadians(float64(i*90+n*5)), scene.DegreesToRadians(float64(segments*step)), 1/math.Tan(rise), segments,
				tint, func(int) float64 { return math.Sin(rise) / 5 })
			for j := 0; j < segments; j++ {
				for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
					vertices[start+4*j+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
				}
			}
		}
	case TripleAttack2:
		for layer := 0; layer < 2; layer++ {
			vertices, indices = nil, nil
			for i := 0; i < 16; i++ {
				from, to, alpha := SharpShootingLine(effect, i, frame)
				if alpha > 0 {
					appendEffectTube(&vertices, &indices, texture, from, to, .01+.05*float64(layer), 5, color.RGBA{255, 255, 255, uint8(alpha)})
				}
			}
			if len(indices) > 0 {
				screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer == 1, false))
			}
		}
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(effect.EffectID != Portal4, false))
	}
}

func SniperBodyFlashActive(effectID int, frame int) bool {
	switch effectID {
	case FalconAssault:
		return frame >= 30 && frame <= 50 && frame%10 < 5
	case Portal4:
		return frame >= 5 && frame <= 25
	}
	return false
}
