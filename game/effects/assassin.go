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

func (m *System) drawSonicBlowEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	right, up, units, ok := projection.BillboardBasis(x, y, z)
	if texture == nil || !ok {
		return
	}
	// Sakexe's PP_2DCIRCLE is a ten-pixel band, twenty pixels above the caster.
	n := float64(now.Sub(effect.Starts)/db.EffectFrameDuration + 1)
	radius := 7*n - .175*n*(n+1)/2
	alpha := scene.ClampFloat((40-n)/40, 0, 1) * 254 / 255
	center := scene.Add3(scene.Point3{X: x, Y: z, Z: y}, scene.Mul3(up, 20*units))
	tint := color.RGBA{255, 255, 255, uint8(alpha * 255)}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 74)
	indices := make([]uint16, 0, 216)
	for i := 0; i <= 36; i++ {
		angle := float64(i) * math.Pi / 18
		axis := scene.Add3(scene.Mul3(right, math.Cos(angle)), scene.Mul3(up, math.Sin(angle)))
		for j, r := range [2]float64{math.Max(0, radius-10), radius} {
			point := scene.Add3(center, scene.Mul3(axis, r*units))
			uv := scene.UV{U: float32(i) / 4, V: float32(1 - j)}
			vertices = append(vertices, scene.TexturedSurfaceVertex3D(point, uv, tint, float32(bounds.Dx()), float32(bounds.Dy())))
		}
		if i < 36 {
			base := uint16(i * 2)
			indices = append(indices, base, base+1, base+2, base+1, base+3, base+2)
		}
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.DepthTest = false
	screen.DrawTriangles3DOwned(vertices, indices, texture, options)
}

type grimtoothHorn struct {
	Origin, right, up, axis scene.Point3
	size, height, Alpha     float64
}

func GrimtoothSpike(effect Instance, index, frame int) grimtoothHorn {
	pitch, yaw := 75.0, float64((6-index)*120)
	size, height, speed, accel := .18, 5.0, .7, .0002
	life, stop, age, fade := 1000, 11, frame, 990
	offset := [3]scene.Point3{{Y: -8, Z: -2.4}, {X: 2.4, Y: -8, Z: 1.2}, {X: -2.4, Y: -8, Z: 1.2}}[index%3]
	if effect.EffectID == Grimtooth {
		birth := index * 3
		age = frame - birth
		life, stop, fade = min(40, 250-birth), 7, 30
		if life < 40 {
			fade = 0
		}
		speed, accel, height = .6, -.03, 2
		size = float64(60+HitEffectRandom(effect, index*3+1, 40)) / 500
		pitch = float64(75 + HitEffectRandom(effect, index*3+2, 30))
		yaw = float64(HitEffectRandom(effect, index*3+3, 360))
		offset = scene.Add3(scene.Point3{Y: -4}, scene.Mul3(effect.ProjectileDirection, float64(birth+1)))
	}
	if age < 0 || age >= life {
		return grimtoothHorn{}
	}
	// Native horns have a square base in XY and their tip on Z, rotated X then Y.
	// Convert the native downward Y axis to our upward one after the rotation.
	p, q := scene.DegreesToRadians(pitch), scene.DegreesToRadians(yaw)
	right := scene.Point3{X: math.Cos(q), Z: -math.Sin(q)}
	up := scene.Point3{X: math.Sin(p) * math.Sin(q), Y: -math.Cos(p), Z: math.Sin(p) * math.Cos(q)}
	axis := scene.Point3{X: math.Cos(p) * math.Sin(q), Y: math.Sin(p), Z: math.Cos(p) * math.Cos(q)}
	n := float64(min(age+1, stop))
	distance := n*speed + accel*n*(n+1)/2
	return grimtoothHorn{
		Origin: scene.Add3(effect.ActorOrigin, scene.Add3(offset, scene.Mul3(axis, distance))),
		right:  right, up: up, axis: axis, size: size, height: height,
		Alpha: scene.ClampFloat(float64(life-age-1)/float64(life-fade), 0, 1) * 254 / 255,
	}
}

func (m *System) drawGrimtoothEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	first, last := 0, 2
	if effect.EffectID == Grimtooth {
		first, last = max(0, (frame-39+2)/3), min(frame/3, 83)
		if effect.ProjectileDirection == (scene.Point3{}) {
			last = 0
		}
	}
	bounds := texture.Bounds()
	vertices := make([]render.Vertex3D, 0, 168)
	indices := make([]uint16, 0, 168)
	for i := first; i <= last; i++ {
		horn := GrimtoothSpike(effect, i, frame)
		if horn.Alpha <= 0 {
			continue
		}
		tint := color.RGBA{255, 255, 255, uint8(horn.Alpha * 255)}
		corners := [4]scene.Point3{
			scene.Mul3(scene.Add3(horn.right, horn.up), -horn.size),
			scene.Mul3(scene.Sub3(horn.right, horn.up), horn.size),
			scene.Mul3(scene.Add3(horn.right, horn.up), horn.size),
			scene.Mul3(scene.Sub3(horn.up, horn.right), horn.size),
		}
		for side, edge := range [4][2]int{{0, 1}, {0, 3}, {3, 2}, {2, 1}} {
			for j, local := range [3]scene.Point3{scene.Mul3(horn.axis, horn.height), corners[edge[0]], corners[edge[1]]} {
				indices = append(indices, uint16(len(vertices)))
				vertices = append(vertices, scene.TexturedSurfaceVertex3D(scene.Add3(horn.Origin, local), QuadHornUVs[side*3+j], tint, float32(bounds.Dx()), float32(bounds.Dy())))
			}
		}
	}
	if len(vertices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, scene.TriangleDrawOptions(render.FilterLinear, render.AddressRepeat))
	}
}
