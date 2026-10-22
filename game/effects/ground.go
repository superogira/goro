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

// Sanctuary, Magnus, Basilica, and Fog Wall share the native rectangular walls.
func (m *System) drawGroundWallsEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	tick := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	vertices := make([]render.Vertex3D, 0, 96)
	indices := make([]uint16, 0, 144)
	center := scene.Point3{X: x, Y: z + .2, Z: y}
	if effect.EffectID == BottomBasilica {
		appendBasilicaWalls(&vertices, &indices, texture, center, tick)
	} else {
		half, height, alpha, fade := 5.0, 50.0, 30, 864
		tint := color.RGBA{255, 255, 255, 255}
		if effect.EffectID == BottomSanc {
			half, height, alpha, fade = 2.5, 16, 120, 1880
			tint.R, tint.B = 235, 235
		}
		if effect.EffectID == FogWallGround {
			half, height, alpha, fade = 2.5, 32, 30, 29999
			tint.R, tint.G, tint.B = 80, 80, 80
		}
		phase := scene.DegreesToRadians(float64((HitEffectRandom(effect, 0, 360) + tick) % 360))
		height *= (.65 + .35*math.Sin(phase)) * math.Sin(scene.DegreesToRadians(float64(min(tick, 90))))
		tint.A = uint8(max(0, alpha-2*max(0, tick-fade)))
		appendSquareEffectBand(&vertices, &indices, texture, center, half/5, height/5, 0, 8*half/5, tint)
		if effect.EffectID == FogWallGround {
			tint.A *= 2
			for _, sign := range []float64{-1, 1} {
				a := scene.Add3(center, scene.Point3{X: half / 5, Z: sign * half / 5})
				b := scene.Add3(center, scene.Point3{X: -half / 5, Z: -sign * half / 5})
				up := scene.Point3{Y: height / 5}
				appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{scene.Add3(a, up), scene.Add3(b, up), b, a}, tint)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(effect.EffectID != BottomSanc && effect.EffectID != FogWallGround, false))
	}
}

// A band follows the square perimeter, splitting at corners. The native
// RenderTeiRect restarts the texture on each face, including partial faces.
func appendSquareEffectBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, center scene.Point3, half, height, start, length float64, tint color.RGBA) {
	if height <= 0 || tint.A == 0 {
		return
	}
	corners := [4]scene.Point3{{X: half, Z: half}, {X: half, Z: -half}, {X: -half, Z: -half}, {X: -half, Z: half}}
	position := math.Mod(start/(2*half), 4)
	side := int(position)
	offset := (position - float64(side)) * 2 * half
	for segment := 0; segment < 5 && length > 1e-9; segment++ {
		step := math.Min(length, 2*half-offset)
		a, b := corners[side], corners[(side+1)%4]
		direction := scene.Mul3(scene.Sub3(b, a), 1/(2*half))
		a = scene.Add3(center, scene.Add3(a, scene.Mul3(direction, offset)))
		b = scene.Add3(a, scene.Mul3(direction, step))
		up := scene.Point3{Y: height}
		appendEffectQuad(vertices, indices, texture, [4]scene.Point3{scene.Add3(a, up), scene.Add3(b, up), b, a}, tint)
		length -= step
		side = (side + 1) % 4
		offset = 0
	}
}

func (m *System) drawPropertyGroundEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	if n <= 0 {
		return
	}
	radius, rise, alpha := propertyGroundRing(n, HitEffectRandom(effect, 1, 100))
	if alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	rise = scene.DegreesToRadians(rise)
	origin := scene.Point3{X: x, Y: z, Z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	tint := color.RGBA{255, 255, 255, uint8(alpha)}
	if effect.EffectID == BottomSuiton {
		tint.R, tint.G = 120, 120
	}
	appendEffectBand(&vertices, &indices, texture, origin, radius/5,
		scene.DegreesToRadians(float64(HitEffectRandom(effect, 0, 360))), 2*math.Pi, 1/math.Tan(rise), 10,
		tint, func(j int) float64 {
			return 3 * math.Sin(rise) * (1 + .3*math.Sin(scene.DegreesToRadians(float64(j*9)))*math.Sin(scene.DegreesToRadians(float64(n%360))))
		})
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
}

// PrimGI4 resets when its rising angle falls below ten degrees. After the
// random initial fade, its 65-tick cycle can be sampled without replaying it.
func propertyGroundRing(tick, initialAlpha int) (radius, rise float64, alpha int) {
	reset := max(1, (initialAlpha+1)/2)
	if tick < reset {
		return 2.6 + .04*float64(tick), float64(80 - tick), max(0, initialAlpha-2*tick)
	}
	phase := (tick - reset) % 65
	alpha = 20*min(phase, 9) - 2*max(0, phase-9)
	return 2.14 + .04*float64(phase), float64(74 - phase), alpha
}

func (m *System) drawGroundSigilEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	if n <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	speed, size := 3, 7.0
	if effect.EffectID == BottomSpider {
		speed, size = 1, 12
	}
	phase := (HitEffectRandom(effect, 0, 360) + speed*n) % 360
	radius := size * (.9 + .1*math.Sin(scene.DegreesToRadians(float64(phase)))) / 5
	origin := scene.Point3{X: x, Y: z + 2.0/5, Z: y}
	vertices, indices := []render.Vertex3D{}, []uint16{}
	appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
		scene.Add3(origin, scene.Point3{Z: -radius}), scene.Add3(origin, scene.Point3{X: -radius}),
		scene.Add3(origin, scene.Point3{Z: radius}), scene.Add3(origin, scene.Point3{X: radius}),
	}, color.RGBA{255, 255, 255, 100})
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

const groundTextureHoverPeriodSeconds = 10.8

func (m *System) drawFlatColorTileEffect(screen *render.Frame, component Component, x, y, z float64) {
	size := component.SizeStart
	if size <= 0 {
		size = 1
	}
	tint := component.Color
	if tint.A == 0 {
		tint.A = 128
	}
	drawGroundTextureQuad(screen, m.whitePixel, x, y, z+component.PosZ+0.02, size, size, 0, tint, false)
}

func (m *System) drawGroundTextureEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, componentIndex int, x, y, z, progress float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	size := component.SizeStart
	if size <= 0 {
		size = 1
	}
	if component.SizeEnd > 0 && component.SizeEnd != size {
		size += (component.SizeEnd - size) * progress
	}
	alpha := effectComponentAlpha(progress, component)
	tint := effectComponentTint(component, alpha)
	zOffset := groundTextureZOffset(component, effect, componentIndex, now)
	drawGroundTextureQuad(screen, texture, x, y, z+zOffset, size, size, component.AngleStart, tint, component.BlendAdditive)
}

func groundTextureZOffset(component Component, effect Instance, componentIndex int, now time.Time) float64 {
	if component.PosZEnd == 0 || component.PosZEnd == component.PosZ || now.IsZero() {
		return component.PosZ
	}
	elapsed := math.Max(0, now.Sub(effect.Starts).Seconds())
	phase := deterministicAngle(effect, componentIndex+733) + elapsed*2*math.Pi/groundTextureHoverPeriodSeconds
	return component.PosZ + (component.PosZEnd-component.PosZ)*(0.5+0.5*math.Sin(phase))
}

func drawGroundTextureQuad(screen *render.Frame, texture *render.Image, x, y, z, sizeX, sizeY, angle float64, tint color.RGBA, additive bool) {
	if screen == nil || texture == nil || sizeX <= 0 || sizeY <= 0 || tint.A == 0 {
		return
	}
	halfX := sizeX * 0.5
	halfY := sizeY * 0.5
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	point := func(dx, dy float64) scene.Point3 {
		px := dx*cosA - dy*sinA
		py := dx*sinA + dy*cosA
		return scene.Point3{X: x + px, Y: z, Z: y + py}
	}
	verts := [4]scene.Point3{
		point(-halfX, -halfY),
		point(halfX, -halfY),
		point(halfX, halfY),
		point(-halfX, halfY),
	}
	uvs := [4]scene.UV{{U: 0, V: 0}, {U: 1, V: 0}, {U: 1, V: 1}, {U: 0, V: 1}}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	if additive {
		options.Blend = render.BlendLighter
	}
	scene.DrawTexturedSurface3DWithOptions(screen, texture, verts, uvs, scene.QuadIndices012023, [4]color.RGBA{tint, tint, tint, tint}, options)
}

func drawWorldSoftRing(screen *render.Frame, white *render.Image, x, y, z, radius, width float64, c color.RGBA, segments int) {
	inner := math.Max(0, radius-width*0.5)
	mid := math.Max(inner+0.01, radius)
	outer := math.Max(mid+0.01, radius+width*0.5)
	drawWorldRingBand(screen, white, x, y, z, inner, mid, 0, c.A, c, segments)
	drawWorldRingBand(screen, white, x, y, z, mid, outer, c.A, 0, c, segments)
}

func drawWorldRingBand(screen *render.Frame, white *render.Image, x, y, z, innerRadius, outerRadius float64, innerAlpha, outerAlpha uint8, c color.RGBA, segments int) {
	if segments < 3 || outerRadius <= innerRadius {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	innerColor := c
	outerColor := c
	innerColor.A = innerAlpha
	outerColor.A = outerAlpha
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		vertices = append(vertices,
			effectVertex3D(x+cosine*innerRadius, y+sine*innerRadius, z, innerColor),
			effectVertex3D(x+cosine*outerRadius, y+sine*outerRadius, z, outerColor),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := scene.TriangleDrawOptions(render.FilterNearest, render.AddressUnsafe)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, white, options)
}
