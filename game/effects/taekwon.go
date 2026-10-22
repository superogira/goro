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

func (m *System) drawTaekwonStorm(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := min(age, 40) - max(0, age-60)
	if alpha <= 0 || len(component.TextureFiles) != 2 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFiles[0])
	if texture == nil {
		return
	}
	rotation := HitEffectRandom(effect, 0, 360)
	for tick := 1; tick <= age; tick++ {
		rotation += 3 + HitEffectRandom(effect, tick, 3)
	}
	var vertices []render.Vertex3D
	var indices []uint16
	tint := color.RGBA{50, 50, 255, uint8(alpha)}
	if effect.EffectID == StormKick3 {
		tint.R, tint.G = 130, 130
	}
	radius := 40.0
	for layer := 0; layer < 10; layer++ {
		if layer > 0 {
			radius -= 2 * (2.2 - .2*float64(layer))
		}
		var corners [4]scene.Point3
		for corner := range corners {
			s, c := math.Sincos(scene.DegreesToRadians(float64(rotation + layer*45 + corner*90)))
			point := scene.Point3{X: c * radius / 5, Z: s * radius / 5, Y: float64(50-layer*5) / 5}
			if corner < 2 {
				point.Y += .2
			}
			corners[3-corner] = scene.Add3(effect.ActorOrigin, point)
		}
		appendEffectQuad(&vertices, &indices, texture, corners, tint)
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(effect.EffectID == StormKick, false))
	m.drawTaekwonWind(screen, ctx, component.TextureFiles[1], effect, age)
}

func (m *System) drawTaekwonWind(screen *render.Frame, ctx client.Context, file string, effect Instance, age int) {
	alpha := min(age, 11) - max(0, age-50)
	texture := m.EffectFileTexture(ctx.Resources, file)
	if alpha <= 0 || texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	span := min(30+age*3, 180)
	step := max(1, span/21)
	for ring := 0; ring < 8; ring++ {
		i := ring % 4
		height, radius := float64(48-i*5), float64(14-i*2)
		if ring >= 4 {
			height, radius = float64(28-i*5), float64(6-i)
		}
		radius += .1 * float64(max(0, age-12))
		pitch := scene.DegreesToRadians(float64(80 + HitEffectRandom(effect, 200+ring, 21)))
		point := func(angle int, top bool) scene.Point3 {
			s, c := math.Sincos(scene.DegreesToRadians(float64(angle + i*90 + age*5)))
			r, y := radius, height
			if top {
				r += math.Cos(pitch)
				y += math.Sin(pitch)
			}
			return scene.Add3(effect.ActorOrigin, scene.Point3{X: c * r / 5, Z: s * r / 5, Y: y / 5})
		}
		for angle, section := step, 0; angle < span; angle, section = angle+step, section+1 {
			start := len(vertices)
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				point(angle-step, true), point(angle, true), point(angle, false), point(angle-step, false),
			}, color.RGBA{255, 255, 255, uint8(alpha)})
			w := float32(texture.Bounds().Dx())
			for j := 0; j < 4; j++ {
				u := float32(min(section, 19)) / 21
				if j == 1 || j == 2 {
					u += 1.0 / 21
				}
				vertices[start+j].SrcX = u * w
			}
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

func (m *System) drawTaekwonStop(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 5*min(age, 5) - 2*max(0, age-9)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if alpha <= 0 || texture == nil {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	var vertices []render.Vertex3D
	var indices []uint16
	for ray := 0; ray < 8; ray++ {
		for layer := -1; layer <= 1; layer++ {
			s, c := math.Sincos(scene.DegreesToRadians(float64(ray*45 + layer)))
			bottom := scene.Add3(origin, scene.Point3{X: c * float64(1+age) / 5, Z: s * float64(1+age) / 5})
			tip := scene.Add3(origin, scene.Point3{X: c * float64(3+age*2) / 5, Z: s * float64(3+age*2) / 5})
			height := math.Min(2, .2*float64(age)) / 5
			low, high, a := height*.2, height, alpha
			if layer != 0 {
				low, high, a = height*.1, height*.4, alpha/3
			}
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{
				scene.Add3(bottom, scene.Point3{Y: low}), scene.Add3(tip, scene.Point3{Y: high}), tip, bottom,
			}, color.RGBA{255, 255, 255, uint8(a)})
		}
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}

type taekwonJumpParticles struct {
	tick   int
	origin scene.Point3
	lights [30]wanderingEffectParticle
}

func (p *taekwonJumpParticles) advance(ctx client.Context, effect Instance, now time.Time) {
	end := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 180)
	for p.tick < end {
		p.tick++
		if p.tick == 36 {
			p.origin = EffectWorldAnchor(ctx, effect, effect.Starts.Add(35*db.EffectFrameDuration))
			for i := range p.lights {
				particle := &p.lights[i]
				particle.seed = i * 10
				angle := deterministicAngle(effect, particle.seed+4)
				particle.Position = scene.Point3{X: .5 * math.Cos(angle), Z: .5 * math.Sin(angle), Y: float64(HitEffectRandom(effect, particle.seed+5, 8)) / 5}
				particle.Age = -HitEffectRandom(effect, particle.seed+6, 26)
				particle.radius = (3 + .1*float64(HitEffectRandom(effect, particle.seed+7, 11))) / 5
				particle.speed = (.1 + .01*float64(HitEffectRandom(effect, particle.seed+8, 16))) / 5
				for j := range particle.angles {
					particle.angles[j] = HitEffectRandom(effect, particle.seed+j, 360)
				}
			}
		}
		for i := range p.lights {
			particle := &p.lights[i]
			particle.Age++
			if particle.Age <= 0 || particle.Age >= 102 {
				continue
			}
			advanceWanderingAngles(particle, effect, p.tick)
			if particle.Position.Y > 0 {
				particle.Position.X += .03 / 5 * math.Sin(scene.DegreesToRadians(float64(particle.angles[0])))
				particle.Position.Z += .03 / 5 * math.Sin(scene.DegreesToRadians(float64(particle.angles[2])))
			}
			particle.Position.Y += particle.speed
		}
	}
}

func (m *System) drawTaekwonJump(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	if effect.taekwonJump == nil || len(component.TextureFiles) != 4 {
		return
	}
	p := effect.taekwonJump
	p.advance(ctx, effect, now)
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if frame < 35 {
		return
	}
	_, up, _, ok := projection.BillboardBasis(p.origin.X, p.origin.Z, p.origin.Y)
	if !ok {
		return
	}
	for i, file := range component.TextureFiles {
		texture := m.EffectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		if i == 0 {
			for _, light := range p.lights {
				alpha := max(0, 5*scene.ClampInt(light.Age, 0, 29)-2*max(0, light.Age-29))
				if alpha == 0 || light.Position.Y < light.radius/math.Sqrt2*up.Y {
					continue
				}
				pos := scene.Add3(p.origin, light.Position)
				size := math.Sqrt2 * light.radius
				drawTexturedEffectBillboardRotatedXY(screen, projection, texture, pos.X, pos.Z, pos.Y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, true)
			}
			continue
		}
		for particle := 0; particle < 64; particle++ {
			if [...]int{1, 2, 3, 2}[particle/4%4] != i {
				continue
			}
			seed := 1000 + particle*4
			age := frame - 34 - HitEffectRandom(effect, seed, 11)
			alpha := max(0, 2*scene.ClampInt(age, 0, 25)-max(0, age-70))
			if alpha == 0 {
				continue
			}
			angle := scene.DegreesToRadians(float64(particle/16*90 + HitEffectRandom(effect, seed+1, 90)))
			radius := (4 + .03*float64(min(age, 90))) / 5
			pos := scene.Add3(p.origin, scene.Point3{X: radius * math.Cos(angle), Z: radius * math.Sin(angle), Y: (4 + 1.5*math.Sin(scene.DegreesToRadians(float64(min(age, 90)*2)))) / 5})
			size := math.Sqrt2 * (2.5 + .2*float64(HitEffectRandom(effect, seed+2, 11))) / 5
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture, pos.X, pos.Z, pos.Y, size, size, 0, color.RGBA{255, 255, 255, uint8(alpha)}, false)
		}
	}
}

type taekwonRunTrail struct {
	tick     int
	position scene.Point3
}

func (m *System) AdvanceTaekwonFootprints(ctx client.Context, now time.Time) {
	var footprints []Instance
	for _, effect := range m.Active {
		p := effect.taekwonRun
		if p == nil || effect.Expired(now) || now.Before(effect.Starts) {
			continue
		}
		end := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
		p.tick = max(p.tick, (end-340)/7*7)
		for p.tick+7 <= end {
			p.tick += 7
			at := effect.Starts.Add(time.Duration(p.tick) * db.EffectFrameDuration)
			position := EffectWorldAnchor(ctx, effect, at)
			delta := scene.Sub3(position, p.position)
			p.position = position
			if p.tick <= 7 || delta.X == 0 && delta.Z == 0 {
				continue
			}
			id := Foot3
			if p.tick/7%2 == 0 {
				id = Foot4
			}
			footprints = append(footprints, Instance{
				EffectID: id, ActorOrigin: position, ProjectileDirection: delta, HasActorTransform: true,
				Starts: at, Expires: at.Add(340 * db.EffectFrameDuration), Duration: 340 * db.EffectFrameDuration,
			})
		}
	}
	m.Active = append(m.Active, footprints...)
}

func (m *System) drawTaekwonFoot(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	age := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	alpha := 150 - (age+1)/2
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if age <= 0 || alpha <= 0 || texture == nil {
		return
	}
	angle := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X) + 3*math.Pi/4
	var corners [4]scene.Point3
	for i := range corners {
		s, c := math.Sincos(angle + float64(i)*math.Pi/2)
		corners[3-i] = scene.Add3(effect.ActorOrigin, scene.Point3{X: c * 1.08, Z: s * 1.08, Y: .04})
	}
	var vertices []render.Vertex3D
	var indices []uint16
	for i := 0; i < 2; i++ {
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{255, 255, 255, uint8(alpha)})
	}
	screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
}
