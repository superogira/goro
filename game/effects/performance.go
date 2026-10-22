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

func (m *System) drawSongMusic(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	file := component.TextureFile
	if len(component.TextureFiles) > 0 {
		file = component.TextureFiles[HitEffectRandom(effect, 0, len(component.TextureFiles))]
	}
	texture := m.EffectFileTexture(ctx.Resources, file)
	if texture == nil {
		return
	}
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	variant := component.Animation
	radius, count := 3.0, 1
	tint := color.RGBA{130, 130, 250, 200}
	switch variant {
	case 1:
		radius = 5
	case 2:
		count = 4
	case 4:
		radius = 15
	case 3:
		tint = color.RGBA{200, 200, 100, 200}
	case 5, 6, 7:
		tint = color.RGBA{255, 255, 255, 200}
	}
	if variant == 6 {
		radius = 8
	}
	for i := 0; i < count; i++ {
		age := n
		if i > 0 {
			if n < 6*i {
				continue
			}
			age = 6*(n/6) - 1 - 6*(i-1)
		}
		p := origin
		p.Y += (6 - 3*math.Sin(scene.DegreesToRadians(float64(HitEffectRandom(effect, 1, 360)+age)))) / 5
		size := math.Sqrt2 * (radius + .5*float64(i)) / 5 * (1 + .05*math.Sin(scene.DegreesToRadians(float64(5*age))))
		if variant == 4 {
			right, _, _, _ := projection.BillboardBasis(p.X, p.Z, p.Y)
			p = scene.Add3(p, scene.Mul3(right, .8))
		}
		angle := 0.0
		if variant == 7 {
			angle = scene.DegreesToRadians(float64(10 * age))
		}
		tint.A = uint8(200 - 50*i)
		DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, p.X, p.Z, p.Y, size, size, angle, tint, TexturedEffectBillboardDrawOptions(variant != 5 && variant != 6, false))
	}
}

func SongOutParticle(n, initial int) (alpha int, height float64) {
	firstEnd := max(1, 6*(10-initial))
	if n < firstEnd {
		fade := max(0, n+initial-10)
		return 25*min(n, 10-initial) - 5*fade, .2 * float64(fade) / 5
	}
	phase := (n - firstEnd) % 60
	fade := max(0, phase-10)
	return 25*min(phase, 10) - 5*fade, .2 * float64(1+fade) / 5
}

func (m *System) drawSongGround(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, origin scene.Point3, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	var vertices []render.Vertex3D
	var indices []uint16
	variant := component.Animation
	additive := true
	switch component.FuncAdapter {
	case effectFuncSongVertical:
		count := 4
		if variant == 4 {
			count = 2
		}
		for i := 0; i < count; i++ {
			salt := i * 3
			from, to := origin, origin
			rate := 2
			tint := color.RGBA{255, 255, 255, 15}
			if variant == 1 || variant == 2 {
				x, z := float64(HitEffectRandom(effect, salt, 7)-3)/5, -.6
				if i >= 2 {
					x, z = z, x
				}
				from.X -= x
				from.Z -= z
				to.X += x
				to.Z += z
				rate, additive = 4, false
				tint = color.RGBA{255, 155, 155, 50}
				if variant == 2 {
					rate = 2
					tint = color.RGBA{155, 255, 155, 50}
				}
			} else if variant == 4 {
				if i == 0 {
					from.X -= 1.2
					to.X += 1.2
				} else {
					from.Z += 1.2
					to.Z -= 1.2
				}
				tint = color.RGBA{255, 150, 150, 100}
			} else {
				to.X += float64(HitEffectRandom(effect, salt, 40)-20) / 5
				to.Z += float64(HitEffectRandom(effect, salt+1, 40)-20) / 5
			}
			height := (20 + 10*math.Sin(scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+2, 360)+rate*n)))) / 5
			up := scene.Point3{Y: height}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{scene.Add3(from, up), scene.Add3(to, up), to, from}, tint)
		}
	case effectFuncSongLight:
		radius := (3 + math.Sin(scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360)+n)))) / 5
		height := float64(25+HitEffectRandom(effect, 1, 10)) / 5 * (1 + .3*math.Sin(scene.DegreesToRadians(float64(n))))
		tint := color.RGBA{100, 100, 255, uint8(50 - HitEffectRandom(effect, 2, 10))}
		if variant == 8 {
			tint.R, tint.G, tint.B = 255, 255, 170
		}
		appendEffectBand(&vertices, &indices, texture, origin, radius,
			scene.DegreesToRadians(float64(HitEffectRandom(effect, 3, 360)+3*n)), scene.DegreesToRadians(315), 0, 21, tint,
			func(j int) float64 { return height * math.Sin(scene.DegreesToRadians(float64(min(j, 20)*9))) })
		for j := 0; j < 21; j++ {
			for k, u := range []int{min(j, 19), min(j+1, 20), min(j+1, 20), min(j, 19)} {
				vertices[4*j+k].SrcX = float32(u) / 21 * float32(texture.Bounds().Dx())
			}
		}
	case effectFuncSongOut:
		for i := 0; i < 2; i++ {
			alpha, height := SongOutParticle(n, HitEffectRandom(effect, i*2, 11))
			p := origin
			p.Y += height
			size := math.Sqrt2 * float64(5+HitEffectRandom(effect, i*2+1, 6)) / 5 * (1 + .05*math.Sin(scene.DegreesToRadians(float64(5*n))))
			DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, p.X, p.Z, p.Y, size, size, 0, color.RGBA{130, 130, 250, uint8(alpha)}, TexturedEffectBillboardDrawOptions(true, false))
		}
		return
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(additive, false))
	}
}
