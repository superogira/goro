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

// PP_3DQUADHORN uses a square base and four consecutive texture strips.
// Sizes and travel are native units; origin is already in world coordinates.
func appendEffectSpike(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, origin scene.Point3, size, height, pitch, yaw, travel float64, tint color.RGBA) {
	base := [4]scene.Point3{{X: -size, Y: -size}, {X: -size, Y: size}, {X: size, Y: size}, {X: size, Y: -size}}
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	for side, pair := range [4][2]int{{0, 3}, {0, 1}, {1, 2}, {2, 3}} {
		start := uint16(len(*vertices))
		for j, p := range [3]scene.Point3{base[pair[0]], {Z: height}, base[pair[1]]} {
			p.Z += travel
			p = rotateModelPointY(rotateModelPointX(p, pitch), yaw)
			p.Y = -p.Y
			uv := scene.UV{U: float32(side) * .2, V: 1}
			if j == 1 {
				uv.V = 0
			} else if j == 2 {
				uv.U += .2
			}
			*vertices = append(*vertices, scene.TexturedSurfaceVertex3D(scene.Add3(origin, scene.Mul3(p, .2)), uv, tint, w, h))
			*indices = append(*indices, start+uint16(j))
		}
	}
}

var QuadHornUVs = []scene.UV{
	{U: 0.0, V: 0}, {U: 0.0, V: 1}, {U: 0.2, V: 1},
	{U: 0.2, V: 0}, {U: 0.2, V: 1}, {U: 0.4, V: 1},
	{U: 0.4, V: 0}, {U: 0.4, V: 1}, {U: 0.6, V: 1},
	{U: 0.6, V: 0}, {U: 0.6, V: 1}, {U: 0.8, V: 1},
}

func (m *System) drawQuadHornEffect(screen *render.Frame, ctx client.Context, effect Instance, component Component, componentIndex int, worldX, worldY, worldZ, progress float64, componentDuration time.Duration) {
	if screen == nil || ctx.Resources == nil || component.TextureFile == "" {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}

	salt := componentIndex * 1009
	height := QuadHornRange(effect, salt+1, component.QuadHornHeightMin, component.QuadHornHeightMax)
	bottomSize := QuadHornRange(effect, salt+2, component.QuadHornBottomMin, component.QuadHornBottomMax)
	if height <= 0 || bottomSize <= 0 {
		return
	}
	offsetX := QuadHornRange(effect, salt+3, component.QuadHornOffsetXMin, component.QuadHornOffsetXMax)
	offsetY := QuadHornRange(effect, salt+4, component.QuadHornOffsetYMin, component.QuadHornOffsetYMax)
	offsetZ := component.QuadHornOffsetZ
	offsetX = QuadHornDefaultOffset(offsetX)
	offsetY = QuadHornDefaultOffset(offsetY)
	offsetZ = QuadHornDefaultOffset(offsetZ)
	height, offsetZ = quadHornAnimation(component, progress, componentDuration, height, offsetZ)

	rotateX := QuadHornRange(effect, salt+5, component.QuadHornRotateXMin, component.QuadHornRotateXMax)
	rotateY := QuadHornRange(effect, salt+6, component.QuadHornRotateYMin, component.QuadHornRotateYMax)
	rotateZ := QuadHornRange(effect, salt+7, component.QuadHornRotateZMin, component.QuadHornRotateZMax)
	x := worldX + component.PosX
	y := worldY + component.PosY
	z := worldZ + component.PosZ
	if ctx.World != nil && (component.PosX != 0 || component.PosY != 0) {
		z = scene.TerrainHeightAtRenderPoint(ctx.World, x, y) + 0.07 + component.PosZ
	}
	origin := scene.Point3{X: x + offsetX, Y: z + height*0.9 + offsetZ, Z: y + offsetY}

	locals := []scene.Point3{
		{X: 0, Y: height, Z: 0},
		{X: -bottomSize, Y: -height, Z: bottomSize},
		{X: bottomSize, Y: -height, Z: bottomSize},

		{X: 0, Y: height, Z: 0},
		{X: bottomSize, Y: -height, Z: bottomSize},
		{X: bottomSize, Y: -height, Z: -bottomSize},

		{X: 0, Y: height, Z: 0},
		{X: bottomSize, Y: -height, Z: -bottomSize},
		{X: -bottomSize, Y: -height, Z: -bottomSize},

		{X: 0, Y: height, Z: 0},
		{X: -bottomSize, Y: -height, Z: -bottomSize},
		{X: -bottomSize, Y: -height, Z: bottomSize},
	}

	bounds := texture.Bounds()
	textureWidth := float32(bounds.Dx())
	textureHeight := float32(bounds.Dy())
	tint := effectComponentTint(component, alpha)
	vertices := make([]render.Vertex3D, 0, len(locals))
	indices := make([]uint16, 0, len(locals))
	for i, local := range locals {
		point := scene.Add3(origin, RotateEffectCylinderVector(local, rotateX, rotateY, rotateZ))
		vertices = append(vertices, scene.TexturedSurfaceVertex3D(point, QuadHornUVs[i], tint, textureWidth, textureHeight))
		indices = append(indices, uint16(i))
	}

	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	if component.BlendAdditive || component.BlendMode == 2 {
		options.Blend = render.BlendLighter
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

func QuadHornRange(effect Instance, salt int, min, max float64) float64 {
	if max < min {
		return max
	}
	return deterministicFloatRange(effect, salt, min, max)
}

func QuadHornDefaultOffset(value float64) float64 {
	if value == 0 {
		return 0.5
	}
	return value
}

func quadHornAnimation(component Component, progress float64, componentDuration time.Duration, height, offsetZ float64) (float64, float64) {
	speed := component.QuadHornAnimSpeed
	if speed <= 0 {
		speed = 100 * time.Millisecond
	}
	elapsed := time.Duration(progress * float64(componentDuration))
	units := float64(elapsed) / float64(speed)

	switch component.Animation {
	case 1:
		if units < height {
			height = units
		}
	case 2:
		if units <= offsetZ {
			offsetZ = units
		}
	case 3:
		if units < height/2 {
			offsetZ = units
		}
	}

	if component.QuadHornAnimOut && componentDuration > speed {
		outStarts := componentDuration - speed
		if elapsed > outStarts {
			outUnits := float64(elapsed-outStarts) / float64(speed)
			offsetZ = -(outUnits * height)
		}
	}
	return height, offsetZ
}

// Wizard ground spikes and Ninja Ice Spear share this native spike animation.
func (m *System) drawGroundSpikes(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	if effect.EffectID == StormGust && lessEffectsEnabled(ctx) {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	if effect.EffectID == Hyousensou {
		origin = effect.ActorOrigin
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	count := 7
	switch effect.EffectID {
	case IceWall:
		count = 3
	case HeavenDrive:
		count = 25
	case StormGust:
		count = 8
	}
	vertices := make([]render.Vertex3D, 0, count*12)
	indices := make([]uint16, 0, count*12)
	for i := 0; i < count; i++ {
		salt := i * 11
		rand := func(n, limit int) int { return HitEffectRandom(effect, salt+n, limit) }
		age, life := frame, 240
		position := origin
		pitch, yaw := float64(80+rand(0, 20)), float64(rand(1, 360))
		size, height := float64(300+rand(2, 50))/100, 18.0
		alpha := 250.0
		travel := earthSpikeTravel(age, 12, 15)
		position.Y -= 2
		switch effect.EffectID {
		case IceWall:
			life, alpha = 9999, 200
			pitch = float64(87 + rand(0, 6))
			size, height = 2.5, float64(300+rand(2, 100))/10
			position.Y = origin.Y - 6
			position.Z -= float64(-2+2*i) * math.Cos(scene.DegreesToRadians(float64(rand(3, 360)))) / 5
			n := float64(min(age+1, 21))
			travel = 3*n - .15*n*(n+1)/2
		case HeavenDrive:
			life = 300 + rand(4, 30) - 10
			height = float64(1+rand(3, 60))/10 + 9
			position.X += float64(i%5 - 2)
			position.Z += float64(i/5 - 2)
			travel = earthSpikeTravel(age, 11, 14)
			n := float64(max(0, age-(life-30)))
			travel -= n + .01*n*(n+1)/2
			alpha = 254
		case StormGust:
			age -= (i/2 + 2) * 29
			life = 215 - (i/2+2)*29
			radius, angle := float64(3+rand(3, 25))/5, scene.DegreesToRadians(float64(rand(4, 360)))
			position.X += radius * math.Sin(angle)
			position.Z -= radius * math.Cos(angle)
			position.Y = origin.Y - 4
			size, height, pitch = float64(400+rand(2, 100))/100, 20, 100
			travel = 1.5 * float64(min(age+1, 4))
			alpha *= scene.ClampFloat(float64(life-1-age)/10, 0, 1)
		default:
			if i > 0 {
				angle := scene.DegreesToRadians(float64(rand(3, 360)))
				position.X += .7 * math.Sin(angle)
				position.Z -= .7 * math.Cos(angle)
				position.Y = origin.Y - 4
				yaw = float64(min(360, rand(4, i*60)+(i-1)*60))
				size, height, pitch = float64(400+rand(2, 100))/100, 20, 100
				travel = 1.5 * float64(min(age+1, 3))
				alpha *= scene.ClampFloat(float64(life-1-age)/10, 0, 1)
			}
		}
		if age < 0 || (age >= life && effect.EffectID != IceWall) {
			continue
		}
		appendEffectSpike(&vertices, &indices, texture, position, size, height, scene.DegreesToRadians(pitch), scene.DegreesToRadians(yaw), travel, color.RGBA{255, 255, 255, uint8(alpha)})
	}
	if len(indices) > 0 {
		options := TexturedEffectBillboardDrawOptions(false, false)
		options.AfterBillboards = effect.EffectID == IceWall
		screen.DrawTriangles3DOwned(vertices, indices, texture, options)
	}
}

func earthSpikeTravel(age, reverse, stop int) float64 {
	n := float64(min(age+1, reverse))
	return n + .01*n*(n+1)/2 - 1.2*float64(max(0, min(age, stop)-reverse+1))
}
