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

func HealEffectID(amount uint32) int {
	// HighPriest 2008 uses EXIT for small heals, not HEAL4 as in dhxj.
	switch {
	case amount < 200:
		return Exit
	case amount < 2000:
		return Heal
	case amount < 4000:
		return Heal2
	default:
		return Heal4
	}
}

func AgilityLabel(decrease bool, frame, duration int) effectScreenParticle {
	n := float64(frame + 1)
	label := effectScreenParticle{
		halfWidth: 40, halfHeight: 20,
		Y:     -(1.5*n - 1.8/float64(duration)*n*(n+1)/2),
		Alpha: effectParticleAlpha(frame, duration, 15, duration-15, 200),
	}
	if decrease {
		label.halfHeight = 10
		label.Y = -80 + n - .01*n*(n+1)/2
	}
	return label
}

func (m *System) DrawAgilityEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	if now.Before(effect.Starts) {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	duration := 100
	if effect.EffectID == IncAgiDex && effect.Duration > 0 {
		duration = max(1, int(effect.Duration/db.EffectFrameDuration))
	}
	if frame >= duration {
		return
	}
	decrease := effect.EffectID == DecAgility
	if origin, ok := projection.ProjectPoint(x, y, z); ok && component.TextureFile != "" {
		drawEffectScreenParticle(screen, m.EffectFileTexture(ctx.Resources, component.TextureFile), origin, AgilityLabel(decrease, frame, duration))
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	vertices := make([]render.Vertex3D, 0, 26*8)
	indices := make([]uint16, 0, 26*12)
	// Native Prim3DTexture follows the actor. One crossed streak is born every
	// two ticks; births whose 50-tick lifetime exceeds the parent fade from zero.
	for birth := max(0, frame/2*2-48); birth <= scene.MinInt(frame, duration-50); birth += 2 {
		age := frame - birth
		maxAlpha := 200.0
		if effect.EffectID == AgiUp {
			maxAlpha = 60
		}
		alpha := effectParticleAlpha(age, 50, 20, 30, maxAlpha)
		if alpha == 0 {
			continue
		}
		n := float64(age + 1)
		angle := scene.DegreesToRadians(float64(HitEffectRandom(effect, birth*4, 360)))
		radius := float64(2+HitEffectRandom(effect, birth*4+1, 7)) / 5
		height := float64(20+HitEffectRandom(effect, birth*4+2, 50)) / 100 * n / 5
		if decrease {
			height = 4 - .015*n*(n+1)/2/5
		}
		center := scene.Point3{X: x + math.Sin(angle)*radius, Y: z + height, Z: y + math.Cos(angle)*radius}
		length := float64(30+HitEffectRandom(effect, birth*4+3, 60)) / 10 / 5
		width := .18 / 5
		if effect.EffectID == AgiUp {
			length = float64(20+HitEffectRandom(effect, birth*4+3, 100)) / 10 / 5
			width = float64(1+HitEffectRandom(effect, birth+1000, 4)) / 10 / 5
		}
		tint := color.RGBA{R: 255, G: 255, B: 255, A: alpha}
		for _, side := range []scene.Point3{{X: width}, {Z: width}} {
			// The texture's U axis runs along the streak, as in Render3DCrossTexture.
			up := scene.Point3{Y: length}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				scene.Sub3(scene.Sub3(center, up), side), scene.Sub3(scene.Add3(center, up), side),
				scene.Add3(scene.Add3(center, up), side), scene.Add3(scene.Sub3(center, up), side),
			}, tint)
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
	}
}

func blessingCircleAlpha(component Component, elapsed time.Duration) float64 {
	if elapsed < 0 || elapsed >= component.Duration || component.FrameDelay <= 0 {
		return 0
	}
	n := float64(elapsed/component.FrameDelay + 1)
	frames := float64(component.Duration / component.FrameDelay)
	// Blessing's Prim3DCircle ramps for 30 ticks at both ends.
	return component.AlphaMax * scene.ClampFloat(math.Min(n/30, (frames-n)/30), 0, 1)
}

func (m *System) drawBlessingCircleEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	alpha := blessingCircleAlpha(component, now.Sub(effect.Starts)-component.Delay)
	if alpha <= 0 || component.CircleSides < 3 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.TextureName)
	if texture == nil {
		return
	}
	bounds := texture.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	tint := effectComponentTint(component, alpha)
	vertices := make([]render.Vertex3D, 0, component.CircleSides*3)
	indices := make([]uint16, 0, component.CircleSides*3)
	center := scene.Point3{X: x, Y: z + component.PosZ, Z: y}
	point := func(i int) scene.Point3 {
		angle := float64(i) * 2 * math.Pi / float64(component.CircleSides)
		return scene.Add3(center, scene.Point3{X: component.SizeStart * math.Cos(angle), Z: component.SizeStart * math.Sin(angle)})
	}
	// Render3DCircle with PT_FILLCIRCLE: a fan on the world X/Z plane,
	// alpha_down from centre (V=1) to rim (V=0), U advancing 1/4 per wedge.
	for i := 0; i < component.CircleSides; i++ {
		u := float32(i) / 4
		base := uint16(len(vertices))
		vertices = append(vertices,
			scene.TexturedSurfaceVertex3D(center, scene.UV{U: u + 0.25, V: 1}, tint, w, h),
			scene.TexturedSurfaceVertex3D(point(i), scene.UV{U: u, V: 0}, tint, w, h),
			scene.TexturedSurfaceVertex3D(point(i+1), scene.UV{U: u + 0.25, V: 0}, tint, w, h))
		indices = append(indices, base, base+1, base+2)
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = !component.Overlay // The original explicitly uses RF_NODEPTHCHECK.
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}
