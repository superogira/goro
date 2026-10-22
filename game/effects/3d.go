package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *System) draw3DEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, componentIndex int, worldX, worldY, worldZ float64, now time.Time) {
	if component.TextureFile == "" && len(component.TextureFiles) == 0 && component.SpriteFile == "" {
		return
	}
	duplicates := scene.MaxInt(component.Duplicate, 1)
	componentDuration := component.Duration
	if componentDuration <= 0 {
		componentDuration = 500 * time.Millisecond
	}
	for i := 0; i < duplicates; i++ {
		starts := effect.Starts.Add(worldEffectComponentStartOffset(component, i))
		progress, active := WorldEffectComponentDuplicateProgressForDraw(effect.Starts, component, i, componentDuration, now)
		if !active {
			continue
		}
		alpha := EffectBillboardAlphaForDuplicate(progress, component, i)
		if alpha <= 0 {
			continue
		}
		salt := componentIndex*1009 + i*37
		offsetX, offsetY, offsetZ := m.Effect3DOffset(ctx, component, effect, salt, i, progress, worldX, worldY, worldZ)
		sizeX, sizeY := Effect3DSize(component, effect, salt, progress, i)
		if sizeX <= 0 || sizeY <= 0 {
			continue
		}
		drawX := worldX + offsetX
		drawY := worldY + offsetY
		drawZ := worldZ + offsetZ
		if component.ShadowTexture {
			drawZ = scene.TerrainHeightAtRenderPoint(ctx.World, drawX, drawY) + scene.ActorShadowTerrainLift
		}
		if component.TextureFile != "" || len(component.TextureFiles) > 0 {
			texture := m.effectTextureFrame(ctx.Resources, component, progress)
			if texture == nil {
				continue
			}
			angle := WorldEffectBillboardAngleForEffect(component, projection, effect, salt, progress)
			DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, drawX, drawY, drawZ, sizeX, sizeY, angle, effectComponentTint(component, alpha), TexturedEffectBillboardDrawOptions(component.BlendAdditive, component.Overlay))
			continue
		}
		size := (sizeX + sizeY) * 0.5
		m.draw3DSpriteEffect(screen, ctx, projection, effect, component, drawX, drawY, drawZ, size, alpha, progress, starts, now)
	}
}

func (m *System) effectTextureFrame(manager *res.Manager, component Component, progress float64) *render.Image {
	if len(component.TextureFiles) == 0 {
		return m.EffectFileTexture(manager, component.TextureFile)
	}
	index := int(scene.ClampFloat(progress, 0, 0.999999) * float64(len(component.TextureFiles)))
	if component.FrameDelay > 0 {
		duration := component.Duration
		if duration <= 0 {
			duration = 500 * time.Millisecond
		}
		elapsed := time.Duration(scene.ClampFloat(progress, 0, 0.999999) * float64(duration))
		index = int(elapsed/component.FrameDelay) % len(component.TextureFiles)
	}
	return m.EffectFileTexture(manager, component.TextureFiles[index])
}

func effectBillboardAlpha(progress float64, component Component) float64 {
	return EffectBillboardAlphaForDuplicate(progress, component, 0)
}

func EffectBillboardAlphaForDuplicate(progress float64, component Component, duplicateIndex int) float64 {
	alphaMax := component.AlphaMax
	if alphaMax <= 0 {
		alphaMax = 1
	}
	alphaMax += component.AlphaMaxDelta * float64(duplicateIndex)
	alphaMax = scene.ClampFloat(alphaMax, 0, 1)
	switch {
	case component.FadeIn && progress < 0.25:
		return progress / 0.25 * alphaMax
	case component.FadeOut && progress > 0.75:
		return (1 - progress) / 0.25 * alphaMax
	case component.Sparkling:
		sparkNumber := component.SparkNumber
		if sparkNumber <= 0 {
			sparkNumber = 1
		}
		steps := progress * 100
		return alphaMax * ((math.Cos((steps*11*float64(sparkNumber)*math.Pi)/180) + 1) / 2)
	default:
		return alphaMax
	}
}

func effectBillboardSize(progress float64, component Component) float64 {
	start := component.SizeStart
	end := component.SizeEnd
	if start <= 0 {
		start = 1
	}
	if end <= 0 {
		end = start
	}
	if !component.SizeSmooth {
		return start + (end-start)*progress
	}
	return start + (end-start)*math.Log10(progress*9+1)
}

func (m *System) Effect3DOffset(ctx client.Context, component Component, effect Instance, salt int, duplicateIndex int, progress float64, worldX, worldY, worldZ float64) (float64, float64, float64) {
	staticX := deterministicSigned(effect, salt+1) * component.PosXRand
	staticY := deterministicSigned(effect, salt+2) * component.PosYRand
	staticZ := deterministicSigned(effect, salt+10) * component.PosZRand
	startX := component.PosX + staticX + component.posXStartMiddle + deterministicSigned(effect, salt+11)*component.PosXStartRand
	startY := component.PosY + staticY + component.posYStartMiddle + deterministicSigned(effect, salt+12)*component.PosYStartRand
	startZ := component.PosZ + staticZ + component.PosZStartMiddle + deterministicSigned(effect, salt+3)*component.PosZStartRand
	endX := component.PosXEnd + staticX + component.posXEndMiddle + deterministicSigned(effect, salt+4)*component.PosXEndRand
	endY := component.PosYEnd + staticY + component.posYEndMiddle + deterministicSigned(effect, salt+5)*component.PosYEndRand
	endZ := component.PosZEnd + staticZ + component.PosZEndMiddle + deterministicSigned(effect, salt+6)*component.PosZEndRand
	if component.circlePattern {
		angle := effectComponentAngleDegrees(component, effect, salt)
		radians := angle * math.Pi / 180
		outer := deterministicFloatRange(effect, salt+23, component.circleOuterRandMin, component.circleOuterRandMax)
		startX = math.Sin(radians)*component.circleInnerSize + staticX
		startY = math.Cos(radians)*component.circleInnerSize + staticY
		endX = math.Sin(radians)*outer + staticX
		endY = math.Cos(radians)*outer + staticY
	}
	if !component.circlePattern && component.PosXEnd == 0 && component.PosXEndRand == 0 && component.posXEndMiddle == 0 && component.PosXStartRand == 0 && component.posXStartMiddle == 0 {
		endX = startX
	}
	if !component.circlePattern && component.PosYEnd == 0 && component.PosYEndRand == 0 && component.posYEndMiddle == 0 && component.PosYStartRand == 0 && component.posYStartMiddle == 0 {
		endY = startY
	}
	if component.PosZEnd == 0 && component.PosZEndRand == 0 && component.PosZEndMiddle == 0 {
		endZ = startZ
	}
	if component.fromSrc || component.ToSrc {
		otherX, otherY, otherZ, ok := effectOtherEndpoint(ctx, effect, worldX, worldY, worldZ)
		if ok {
			dx := otherX - worldX
			dy := otherY - worldY
			dz := otherZ - worldZ
			if component.fromSrc {
				endX += dx
				endY += dy
				endZ += dz
			}
			if component.ToSrc {
				startX += dx
				startY += dy
				startZ += dz
			}
		}
	}
	x := effectPositionAxis(progress, startX, endX, component.PosXSmooth)
	y := effectPositionAxis(progress, startY, endY, component.PosYSmooth)
	z := effectPositionAxis(progress, startZ, endZ, component.PosZSmooth)
	if component.Retreat != 0 {
		dx := endX - startX
		dy := endY - startY
		dist := math.Hypot(dx, dy)
		if dist > 0.001 {
			factor := math.Sin(progress*math.Pi) * component.Retreat
			x -= dx / dist * factor
			y -= dy / dist * factor
		}
	}
	if component.Arc != 0 {
		z += math.Sin(progress*math.Pi) * component.Arc
	}
	if component.OrbitRadiusX != 0 || component.OrbitRadiusY != 0 || component.orbitRadiusZ != 0 {
		angle := progress*component.OrbitRotations*350*math.Pi/180 - (component.OrbitPhase+component.orbitPhaseDelta*float64(duplicateIndex))*math.Pi/2
		if component.OrbitRadiusX != 0 {
			x = math.Cos(angle) * component.OrbitRadiusX
			if component.OrbitClockwise {
				x = -x
			}
		}
		if component.OrbitRadiusY != 0 {
			y = math.Sin(angle) * component.OrbitRadiusY
		}
	}
	return x, y, z
}

func effectOtherEndpoint(ctx client.Context, effect Instance, fallbackX, fallbackY, fallbackZ float64) (float64, float64, float64, bool) {
	if effect.TargetID == 0 || ctx.World == nil {
		return fallbackX, fallbackY, fallbackZ, false
	}
	if actor, ok := ctx.World.Actors[effect.TargetID]; ok {
		x, y := scene.ActorRenderPosition(actor, time.Now())
		return scene.CellCenter(x), scene.CellCenter(y), scene.TerrainHeightAt(ctx.World, x, y) + 0.07, true
	}
	if scene.IsLocalActor(ctx, effect.TargetID) {
		x, y := scene.ActorRenderPosition(ctx.World.Player, time.Now())
		return scene.CellCenter(x), scene.CellCenter(y), scene.TerrainHeightAt(ctx.World, x, y) + 0.07, true
	}
	return fallbackX, fallbackY, fallbackZ, false
}

func effectPositionAxis(progress, start, end float64, smooth bool) float64 {
	if smooth {
		return start + (end-start)*math.Log10(progress*9+1)
	}
	return start + (end-start)*progress
}

func WorldEffectBillboardAngleForEffect(component Component, projection scene.Projection, effect Instance, salt int, progress float64) float64 {
	angle := effectComponentAngleDegrees(component, effect, salt)
	if component.angleRandMax <= component.angleRandMin {
		angle = component.AngleStart + (component.AngleEnd-component.AngleStart)*progress
	}
	angle *= math.Pi / 180
	if component.RotateWithCamera {
		angle += scene.DegreesToRadians(projection.CameraYaw)
	}
	return angle
}

func effectComponentAngleDegrees(component Component, effect Instance, salt int) float64 {
	if component.angleRandMax > component.angleRandMin {
		return deterministicFloatRange(effect, salt+21, component.angleRandMin, component.angleRandMax)
	}
	return component.AngleStart
}

func Effect3DSize(component Component, effect Instance, salt int, progress float64, duplicateIndex int) (float64, float64) {
	if effect.Size > 0 {
		return effect.Size, effect.Size
	}
	size := effectBillboardSize(progress, component)
	sizeX := size
	sizeY := size
	if component.SizeStartX > 0 || component.SizeEndX > 0 {
		sizeX = effectAxisSize(progress, component.SizeStartX, component.SizeEndX, component.SizeSmooth)
	}
	if component.SizeStartY > 0 || component.SizeEndY > 0 {
		sizeY = effectAxisSize(progress, component.SizeStartY, component.SizeEndY, component.SizeSmooth)
	}
	if component.sizeStartXRandMax > component.sizeStartXRandMin || component.sizeEndXRandMax > component.sizeEndXRandMin {
		start := component.SizeStartX
		if component.sizeStartXRandMax > component.sizeStartXRandMin {
			start = deterministicFloatRange(effect, salt+31, component.sizeStartXRandMin, component.sizeStartXRandMax)
		}
		end := component.SizeEndX
		if component.sizeEndXRandMax > component.sizeEndXRandMin {
			end = deterministicFloatRange(effect, salt+32, component.sizeEndXRandMin, component.sizeEndXRandMax)
		}
		sizeX = effectAxisSize(progress, start, end, component.SizeSmooth)
	}
	if component.sizeStartYRandMax > component.sizeStartYRandMin || component.sizeEndYRandMax > component.sizeEndYRandMin {
		start := component.SizeStartY
		if component.sizeStartYRandMax > component.sizeStartYRandMin {
			start = deterministicFloatRange(effect, salt+33, component.sizeStartYRandMin, component.sizeStartYRandMax)
		}
		end := component.SizeEndY
		if component.sizeEndYRandMax > component.sizeEndYRandMin {
			end = deterministicFloatRange(effect, salt+34, component.sizeEndYRandMin, component.sizeEndYRandMax)
		}
		sizeY = effectAxisSize(progress, start, end, component.SizeSmooth)
	}
	if component.SizeDelta != 0 {
		delta := component.SizeDelta * float64(duplicateIndex) * scene.PixelRatio
		sizeX += delta
		sizeY += delta
	}
	if component.SizeRand != 0 {
		sizeX += deterministicSigned(effect, salt+7) * component.SizeRand
		sizeY = sizeX
	}
	if component.sizeRandX != 0 {
		if component.sizeRandXMiddle != 0 {
			sizeX = component.sizeRandXMiddle + deterministicSigned(effect, salt+8)*component.sizeRandX
		} else {
			sizeX += deterministicSigned(effect, salt+8) * component.sizeRandX
		}
	}
	if component.SizeRandY != 0 {
		if component.SizeRandYMiddle != 0 {
			sizeY = component.SizeRandYMiddle + deterministicSigned(effect, salt+9)*component.SizeRandY
		} else {
			sizeY += deterministicSigned(effect, salt+9) * component.SizeRandY
		}
	}
	return sizeX, sizeY
}

func effectAxisSize(progress, start, end float64, smooth bool) float64 {
	if start <= 0 && end > 0 {
		start = end
	}
	if end <= 0 && start > 0 {
		end = start
	}
	if smooth {
		factor := math.Log10(progress*9 + 1)
		return start + (end-start)*factor
	}
	return start + (end-start)*progress
}

func drawTexturedEffectBillboardRotatedXY(screen *render.Frame, projection scene.Projection, texture *render.Image, worldX, worldY, worldZ, sizeX, sizeY, angle float64, tint color.RGBA, additive bool) {
	DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, worldX, worldY, worldZ, sizeX, sizeY, angle, tint, TexturedEffectBillboardDrawOptions(additive, false))
}

func TexturedEffectBillboardDrawOptions(additive, overlay bool) *render.DrawTrianglesOptions {
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	if additive {
		options.Blend = render.BlendLighter
	}
	if overlay {
		options.DepthTest = false
	}
	return options
}

func DrawTexturedEffectBillboardRotatedXYWithOptions(screen *render.Frame, projection scene.Projection, texture *render.Image, worldX, worldY, worldZ, sizeX, sizeY, angle float64, tint color.RGBA, options *render.DrawTrianglesOptions) {
	if screen == nil || texture == nil || tint.A == 0 {
		return
	}
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	center := scene.Point3{X: worldX, Y: worldZ, Z: worldY}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	axisScaleX := sizeX / float64(w)
	axisScaleY := sizeY / float64(h)
	rightAxis := scene.Mul3(right, axisScaleX)
	upAxis := scene.Mul3(up, -axisScaleY)
	if angle != 0 {
		sinA, cosA := math.Sin(angle), math.Cos(angle)
		rightAxis = scene.Add3(scene.Mul3(right, cosA*axisScaleX), scene.Mul3(up, -sinA*axisScaleX))
		upAxis = scene.Add3(scene.Mul3(right, -sinA*axisScaleY), scene.Mul3(up, -cosA*axisScaleY))
	}
	if options == nil {
		options = TexturedEffectBillboardDrawOptions(false, false)
	}
	screen.DrawWorldBillboard(render.WorldBillboardCommand{
		Texture:     texture,
		Options:     *options,
		Center:      [3]float32{float32(center.X), float32(center.Y), float32(center.Z)},
		RightAxis:   [3]float32{float32(rightAxis.X), float32(rightAxis.Y), float32(rightAxis.Z)},
		UpAxis:      [3]float32{float32(upAxis.X), float32(upAxis.Y), float32(upAxis.Z)},
		DepthUpAxis: [3]float32{float32(upAxis.X), float32(upAxis.Y), float32(upAxis.Z)},
		Width:       w,
		Height:      h,
		AnchorX:     w * 0.5,
		AnchorY:     h * 0.5,
		ColorR:      float32(tint.R) / 255,
		ColorG:      float32(tint.G) / 255,
		ColorB:      float32(tint.B) / 255,
		ColorA:      float32(tint.A) / 255,
	})
}

func groundEffectRingMesh(texture *render.Image, center scene.Point3, inner, outer float64, alpha uint8) ([]render.Vertex3D, []uint16) {
	vertices := make([]render.Vertex3D, 0, 22)
	indices := make([]uint16, 0, 60)
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	tint := color.RGBA{255, 255, 255, alpha}
	for i := 0; i <= 10; i++ {
		angle := float64(i) * math.Pi / 5
		axis := scene.Point3{X: math.Cos(angle), Z: math.Sin(angle)}
		for j, radius := range [2]float64{inner, outer} {
			point := scene.Add3(center, scene.Mul3(axis, radius))
			vertices = append(vertices, scene.TexturedSurfaceVertex3D(point, scene.UV{U: float32(i) / 4, V: float32(1 - j)}, tint, w, h))
		}
		if i < 10 {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	return vertices, indices
}

func appendEffectQuad(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, corners [4]scene.Point3, tint color.RGBA) {
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	base := uint16(len(*vertices))
	for i, uv := range [...]scene.UV{{U: 0, V: 0}, {U: 1, V: 0}, {U: 1, V: 1}, {U: 0, V: 1}} {
		*vertices = append(*vertices, scene.TexturedSurfaceVertex3D(corners[i], uv, tint, w, h))
	}
	*indices = append(*indices, base, base+1, base+2, base, base+2, base+3)
}

// The native PP_INTIMIDATE and PP_TEIHIT1 rotate in XY, then XZ.
// Native Y points down; our Y points up. Distances here use native units.
func effectParticlePoint(x, y, pitch, yaw float64) scene.Point3 {
	s, c := math.Sincos(pitch)
	rx, ry := c*x-s*y, s*x+c*y
	s, c = math.Sincos(yaw)
	return scene.Point3{X: c * rx / 5, Y: -ry / 5, Z: s * rx / 5}
}

func effectSphereMesh(texture *render.Image, center scene.Point3, radius, rotation float64, alpha uint8) ([]render.Vertex3D, []uint16) {
	vertices := make([]render.Vertex3D, 0, 200)
	indices := make([]uint16, 0, 300)
	w, h := float32(texture.Bounds().Dx()), float32(texture.Bounds().Dy())
	tint := color.RGBA{255, 255, 255, alpha}
	point := func(latitude, longitude float64) scene.Point3 {
		pitch := scene.DegreesToRadians(latitude)
		yaw := scene.DegreesToRadians(longitude + rotation)
		return scene.Add3(center, scene.Point3{X: radius * math.Cos(pitch) * math.Sin(yaw), Y: radius * math.Sin(pitch), Z: radius * math.Cos(pitch) * math.Cos(yaw)})
	}
	for row := 0; row < 5; row++ {
		for col := 0; col < 10; col++ {
			latitude, longitude := float64(-90+row*36), float64(col*36)
			// GetRadian takes integer degrees; the source's extra .25 is truncated.
			corners := [4]scene.Point3{
				point(latitude, longitude), point(latitude+36, longitude),
				point(latitude+36, longitude+36), point(latitude, longitude+36),
			}
			u, v := float32(col)/10, float32(row)/5
			base := uint16(len(vertices))
			for i, uv := range [4]scene.UV{{U: u, V: v}, {U: u, V: v + .2}, {U: u + .1, V: v + .2}, {U: u + .1, V: v}} {
				vertices = append(vertices, scene.TexturedSurfaceVertex3D(corners[i], uv, tint, w, h))
			}
			indices = append(indices, base, base+1, base+3, base+2, base+3, base+1)
		}
	}
	return vertices, indices
}

func effectTexturedVertex3D(x, y, z float64, srcX, srcY float32, c color.RGBA) render.Vertex3D {
	point := scene.Point3{X: x, Y: z, Z: y}
	return render.Vertex3D{
		X:      float32(point.X),
		Y:      float32(point.Y),
		Z:      float32(point.Z),
		SrcX:   srcX,
		SrcY:   srcY,
		ColorR: float32(c.R) / 255,
		ColorG: float32(c.G) / 255,
		ColorB: float32(c.B) / 255,
		ColorA: float32(c.A) / 255,
		DepthX: float32(point.X),
		DepthY: float32(point.Y),
		DepthZ: float32(point.Z),
	}
}

func effectVertex3D(x, y, z float64, c color.RGBA) render.Vertex3D {
	return effectTexturedVertex3D(x, y, z, 0, 0, c)
}
