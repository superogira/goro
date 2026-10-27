package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func appendEffectBand(vertices *[]render.Vertex3D, indices *[]uint16, texture *render.Image, origin scene.Point3, radius, angle, sweep, flare float64, segments int, tint color.RGBA, height func(int) float64) {
	point := func(i int) (scene.Point3, scene.Point3) {
		a, h := angle+sweep*float64(i)/float64(segments), height(i)
		c, s := math.Cos(a), math.Sin(a)
		bottom := scene.Add3(origin, scene.Point3{X: radius * c, Z: radius * s})
		top := scene.Add3(origin, scene.Point3{X: (radius + h*flare) * c, Y: h, Z: (radius + h*flare) * s})
		return bottom, top
	}
	for i := 0; i < segments; i++ {
		b1, t1 := point(i)
		b2, t2 := point(i + 1)
		start := len(*vertices)
		appendEffectQuad(vertices, indices, texture, [4]scene.Point3{t1, t2, b2, b1}, tint)
		for j, u := range []float32{float32(i) / 21, float32(i+1) / 21, float32(i+1) / 21, float32(i) / 21} {
			(*vertices)[start+j].SrcX = u * float32(texture.Bounds().Dx())
		}
	}
}

func (m *System) drawCylinderEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, componentIndex int, x, y, z float64, componentDuration time.Duration, now time.Time) {
	texture := m.effectTexture(ctx.Resources, component.TextureName)
	if texture == nil {
		return
	}
	duplicates := scene.MaxInt(component.Duplicate, 1)
	baseDuration := componentDuration - worldEffectComponentMaxStartOffset(component)
	if baseDuration <= 0 {
		baseDuration = component.Duration
	}
	if baseDuration <= 0 {
		baseDuration = 500 * time.Millisecond
	}
	for i := 0; i < duplicates; i++ {
		instanceProgress, active := WorldEffectComponentDuplicateProgressForDraw(effect.Starts, component, i, baseDuration, now)
		if !active {
			continue
		}
		m.drawCylinderEffectInstance(screen, projection, texture, effect, component, componentIndex, x, y, z, instanceProgress, i)
	}
}

func (m *System) drawCylinderEffectInstance(screen *render.Frame, projection scene.Projection, texture *render.Image, effect Instance, component Component, componentIndex int, x, y, z, progress float64, duplicateIndex int) {
	alpha := effectComponentAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	bottomSize, topSize, height := EffectCylinderAnimatedDimensions(component, progress)
	if !component.FixedPerspective {
		drawWorldCylinderBandOriented(screen, m.whitePixel, texture, projection, component, x, y, z+component.PosZ, bottomSize, topSize, height, effectComponentTint(component, alpha), scene.MaxInt(component.CircleSides, component.TotalCircleSides), progress)
		return
	}
	angle := 0.0
	if component.Rotate {
		angle += progress * 2 * math.Pi
		angle += deterministicAngle(effect, componentIndex*101+duplicateIndex+31) * 0.08
	}
	if component.AngleZRandom != 0 {
		angle += deterministicAngle(effect, componentIndex*101+duplicateIndex) * component.AngleZRandom / 360
	}
	drawTexturedEffectCylinder(screen, projection, texture, x, y, z+component.PosZ, effectCylinderDraw{
		bottomSize:       bottomSize,
		topSize:          topSize,
		totalCircleSides: component.TotalCircleSides,
		circleSides:      component.CircleSides,
		alpha:            alpha,
		angle:            angle,
	})
}

func EffectCylinderAnimatedDimensions(component Component, progress float64) (float64, float64, float64) {
	bottomSize := component.BottomSize
	topSize := component.TopSize
	height := component.Height
	switch component.Animation {
	case 1:
		height *= progress
	case 2:
		topSize *= progress
	case 3:
		scale := 1 - progress
		bottomSize *= scale
		topSize *= scale
		if progress < 0.5 {
			height *= progress * 2
		} else if progress > 0.5 {
			height *= (1 - progress) * 2
		}
	case 4:
		bottomSize *= progress
		topSize *= progress
	case 5:
		if progress < 0.5 {
			height *= progress * 2
		} else {
			height *= (1 - progress) * 2
		}
	}
	return bottomSize, topSize, height
}

type effectCylinderDraw struct {
	bottomSize       float64
	topSize          float64
	totalCircleSides int
	circleSides      int
	alpha            float64
	angle            float64
}

func drawTexturedEffectCylinder(screen *render.Frame, projection scene.Projection, texture *render.Image, worldX, worldY, worldZ float64, draw effectCylinderDraw) {
	if screen == nil || texture == nil || draw.alpha <= 0 || draw.topSize <= 0 || draw.totalCircleSides <= 0 || draw.circleSides <= 0 {
		return
	}
	right, up, _, ok := projection.BillboardBasis(worldX, worldY, worldZ)
	if !ok {
		return
	}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	center := scene.Point3{X: worldX, Y: worldZ, Z: worldY}
	tint := color.RGBA{R: 255, G: 255, B: 255, A: uint8(scene.ClampFloat(draw.alpha, 0, 1) * 255)}
	vertices := make([]render.Vertex3D, 0, (draw.circleSides+1)*2)
	indices := make([]uint16, 0, draw.circleSides*6)
	point := func(radius, angle float64) scene.Point3 {
		return scene.Add3(scene.Add3(center, scene.Mul3(right, math.Sin(angle)*radius)), scene.Mul3(up, math.Cos(angle)*radius))
	}
	for i := 0; i <= draw.circleSides; i++ {
		a := float64(i) / float64(draw.totalCircleSides)
		angle := draw.angle + a*2*math.Pi
		u := float32(a * float64(draw.totalCircleSides) / float64(draw.circleSides))
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(point(draw.bottomSize, angle), scene.UV{U: u, V: 1}, tint, w, h),
			scene.TexturedSurfaceVertex3D(point(draw.topSize, angle), scene.UV{U: u, V: 0}, tint, w, h),
		)
		if i < draw.circleSides {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
}

func drawWorldCylinderBandOriented(screen *render.Frame, white, texture *render.Image, projection scene.Projection, component Component, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, progress float64) {
	right := scene.Point3{X: 1}
	depth := scene.Point3{Z: 1}
	up := scene.Point3{Y: 1}
	if component.AngleX != 0 || component.AngleY != 0 || component.angleZ != 0 || component.RotateWithCamera || component.Rotate {
		angleY := component.AngleY
		if component.Rotate {
			angleY += progress * 360
		}
		if component.RotateWithCamera {
			angleY += projection.CameraYaw
		}
		right = RotateEffectCylinderVector(right, component.AngleX, angleY, component.angleZ)
		depth = RotateEffectCylinderVector(depth, component.AngleX, angleY, component.angleZ)
		up = RotateEffectCylinderVector(up, component.AngleX, angleY, component.angleZ)
	}
	DrawWorldCylinderBandWithBasis(screen, white, texture, x, y, z, bottomRadius, topRadius, height, c, segments, right, depth, up)
}

func RotateEffectCylinderVector(v scene.Point3, angleX, angleY, angleZ float64) scene.Point3 {
	v = rotateModelPointX(v, scene.DegreesToRadians(angleX))
	v = rotateModelPointY(v, scene.DegreesToRadians(angleY))
	v = rotateModelPointZ(v, scene.DegreesToRadians(angleZ))
	return v
}

func rotateModelPointX(v scene.Point3, angle float64) scene.Point3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return scene.Point3{X: v.X, Y: v.Y*cosA - v.Z*sinA, Z: v.Y*sinA + v.Z*cosA}
}

func rotateModelPointY(v scene.Point3, angle float64) scene.Point3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return scene.Point3{X: v.X*cosA + v.Z*sinA, Y: v.Y, Z: -v.X*sinA + v.Z*cosA}
}

func rotateModelPointZ(v scene.Point3, angle float64) scene.Point3 {
	if angle == 0 {
		return v
	}
	sinA, cosA := math.Sin(angle), math.Cos(angle)
	return scene.Point3{X: v.X*cosA - v.Y*sinA, Y: v.X*sinA + v.Y*cosA, Z: v.Z}
}

func DrawWorldCylinderBandWithBasis(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, right, depth, up scene.Point3) {
	drawWorldCylinderBandWithOptions(screen, white, texture, x, y, z, bottomRadius, topRadius, height, c, segments, right, depth, up, 1, render.BlendLighter)
}

func drawWorldCylinderBandWithOptions(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, right, depth, up scene.Point3, textureRepeats float64, blend render.Blend) {
	if segments < 3 || bottomRadius <= 0.01 || topRadius <= 0.01 || math.Abs(height) <= 0.01 || c.A == 0 {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	tint := c
	srcW, srcH := float32(1), float32(1)
	source := white
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	for i := 0; i <= segments; i++ {
		u := float32(float64(i) / float64(segments) * textureRepeats)
		angle := float64(i) * 2 * math.Pi / float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		center := scene.Point3{X: x, Y: z, Z: y}
		bottom := scene.Add3(center, scene.Add3(scene.Mul3(right, cosine*bottomRadius), scene.Mul3(depth, sine*bottomRadius)))
		top := scene.Add3(center, scene.Add3(scene.Add3(scene.Mul3(right, cosine*topRadius), scene.Mul3(depth, sine*topRadius)), scene.Mul3(up, height)))
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(bottom, scene.UV{U: u, V: 1}, tint, srcW, srcH),
			scene.TexturedSurfaceVertex3D(top, scene.UV{U: u, V: 0}, tint, srcW, srcH),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = blend
	screen.DrawTriangles3DOwned(vertices, indices, source, options)
}

func drawWorldCylinderBandRotated(screen *render.Frame, white, texture *render.Image, x, y, z, bottomRadius, topRadius, height float64, c color.RGBA, segments int, angleOffset float64) {
	if segments < 3 || bottomRadius <= 0.01 || topRadius <= 0.01 || math.Abs(height) <= 0.01 || c.A == 0 {
		return
	}
	vertices := make([]render.Vertex3D, 0, (segments+1)*2)
	indices := make([]uint16, 0, segments*6)
	tint := c
	srcW, srcH := float32(1), float32(1)
	source := white
	if texture != nil {
		source = texture
		bounds := texture.Bounds()
		srcW = float32(bounds.Dx())
		srcH = float32(bounds.Dy())
	}
	for i := 0; i <= segments; i++ {
		u := float32(i) / float32(segments)
		angle := angleOffset + float64(i)*2*math.Pi/float64(segments)
		cosine := math.Cos(angle)
		sine := math.Sin(angle)
		topAngle := angleOffset + (float64(i)+0.5)*2*math.Pi/float64(segments)
		topCosine := math.Cos(topAngle)
		topSine := math.Sin(topAngle)
		vertices = append(vertices,
			effectTexturedVertex3D(x+cosine*bottomRadius, y+sine*bottomRadius, z, u*srcW, srcH, tint),
			effectTexturedVertex3D(x+topCosine*topRadius, y+topSine*topRadius, z+height, u*srcW, 0, tint),
		)
		if i == segments {
			continue
		}
		base := uint16(i * 2)
		indices = append(indices, base, base+1, base+3, base, base+3, base+2)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendLighter
	screen.DrawTriangles3D(vertices, indices, source, options)
}
