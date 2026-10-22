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

// Each petal captures its target's position when it appears, then drifts on its own.
type NinjaPetals struct {
	tick      int
	Particles [20]wanderingEffectParticle
}

func (p *NinjaPetals) Advance(ctx client.Context, effect Instance, now time.Time) {
	end := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 110)
	for p.tick < end {
		if p.tick < len(p.Particles) {
			seed := p.tick * 512
			particle := &p.Particles[p.tick]
			particle.seed, particle.kind = seed, HitEffectRandom(effect, seed+4, 3)
			particle.speed = float64(2+HitEffectRandom(effect, seed+5, 3)) * .15 / 5
			particle.radius = float64(22+HitEffectRandom(effect, seed+6, 6)) / 70
			particle.Position = EffectWorldAnchor(ctx, effect, effect.Starts.Add(time.Duration(p.tick)*db.EffectFrameDuration))
			s, c := math.Sincos(scene.DegreesToRadians(float64(HitEffectRandom(effect, seed+7, 360))))
			particle.Position.X += 1.6 * c
			particle.Position.Z += 1.6 * s
			for i := range particle.angles {
				particle.angles[i] = HitEffectRandom(effect, seed+i, 360)
			}
		}
		p.tick++
		for i := 0; i < min(p.tick, len(p.Particles)); i++ {
			particle := &p.Particles[i]
			if particle.Age >= 83 {
				continue
			}
			particle.Age++
			advanceWanderingAngles(particle, effect, particle.Age+20)
			particle.Position.Y += particle.speed
			particle.Position.X += .03 * math.Sin(scene.DegreesToRadians(float64(particle.angles[0])))
			particle.Position.Z += .03 * math.Sin(scene.DegreesToRadians(float64(particle.angles[2])))
		}
	}
}

func (m *System) drawNinjaPetals(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, now time.Time) {
	if effect.NinjaPetals == nil {
		return
	}
	effect.NinjaPetals.Advance(ctx, effect, now)
	for i := 0; i < min(effect.NinjaPetals.tick, 20); i++ {
		p := effect.NinjaPetals.Particles[i]
		alpha := 250 - 3*p.Age
		if alpha < 3 {
			continue
		}
		motion := 1 + (p.Age-1)/4
		m.drawEffectParticleSpriteAction(screen, ctx, projection, c.SpriteFile, p.Position, p.kind, motion, p.radius, 0, color.RGBA{255, 255, 255, uint8(alpha)})
	}
}

type ninjaWindSample struct {
	Position    scene.Point3
	Alpha, spin int
}

type ninjaWindHead struct {
	Samples        [4]ninjaWindSample
	heading, speed float64
	fading         bool
}

type NinjaWind struct {
	tick    int
	Heads   [36]ninjaWindHead
	origins [5]scene.Point3
	born    int
}

func (p *NinjaWind) Advance(ctx client.Context, effect Instance, now time.Time) {
	end := min(int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1, 200)
	for p.tick < end {
		p.tick++
		at := effect.Starts.Add(time.Duration(p.tick-1) * db.EffectFrameDuration)
		target := EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, at)
		_, exists, _ := scene.ActorForCombatID(ctx, effect.TargetID)
		for i := 0; i < max(1, min(effect.HitCount, 36)); i++ {
			h := &p.Heads[i]
			turn := 1.0
			if i%2 == 0 {
				turn = -1
			}
			if p.tick == 1 {
				for j := range h.Samples {
					h.Samples[j].Position = effect.ActorOrigin
					h.Samples[j].Position.Y += 2
				}
				h.heading = math.Atan2(target.Z-effect.ActorOrigin.Z, target.X-effect.ActorOrigin.X)*180/math.Pi - turn*float64(25+HitEffectRandom(effect, i*8, 21))
				h.speed = float64(10+HitEffectRandom(effect, i*8+1, 21)) / 50
			}
			if !exists || p.tick > 20 {
				h.fading = true
			}
			s := &h.Samples[0]
			if h.fading && s.Alpha <= 0 {
				continue
			}
			s.spin = p.tick * 30
			if p.tick > 1 && p.tick <= 20 {
				s.Alpha = min(s.Alpha+8, 120)
			}
			if p.tick > 5 {
				if !h.fading {
					toward := math.Mod(math.Atan2(target.Z-s.Position.Z, target.X-s.Position.X)*180/math.Pi+360, 360)
					if math.Abs(h.heading-toward) <= 3 {
						h.fading = true
					} else {
						h.heading += 5 * turn
					}
				} else {
					for j := range h.Samples {
						h.Samples[j].Alpha = max(0, h.Samples[j].Alpha-5)
					}
					h.heading += turn
				}
			}
			h.heading = math.Mod(h.heading+360, 360)
			sin, cos := math.Sincos(scene.DegreesToRadians(h.heading))
			s.Position.X += cos * h.speed
			s.Position.Z += sin * h.speed
			if p.tick%3 == 0 {
				for j := 3; j > 0; j-- {
					h.Samples[j] = h.Samples[j-1]
					h.Samples[j].Alpha = h.Samples[j].Alpha * 8 / 10
				}
			}
		}
	}
}

func (m *System) drawNinjaWind(screen *render.Frame, ctx client.Context, c Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	if texture == nil || effect.NinjaWind == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	quad := func(position scene.Point3, spin, alpha int, radius, heading float64, vertical bool) {
		if alpha <= 0 {
			return
		}
		var corners [4]scene.Point3
		for i := range corners {
			sin, cos := math.Sincos(scene.DegreesToRadians(float64(spin + (i+1)*90)))
			offset := scene.Point3{X: cos * radius, Z: sin * radius}
			if vertical {
				offset = scene.Point3{X: cos * radius * math.Cos(heading), Z: cos * radius * math.Sin(heading), Y: -sin * radius}
			}
			corners[3-i] = scene.Add3(position, offset)
		}
		appendEffectQuad(&vertices, &indices, texture, corners, color.RGBA{80, 255, 80, uint8(alpha)})
	}
	if effect.EffectID == Stin4 {
		effect.NinjaWind.Advance(ctx, effect, now)
		for i := 0; i < max(1, min(effect.HitCount, 36)); i++ {
			for j, s := range effect.NinjaWind.Heads[i].Samples {
				quad(s.Position, s.spin, s.Alpha, float64(20+HitEffectRandom(effect, i*8+2+j, 11))/10, 0, false)
			}
		}
	} else {
		tick := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
		for i := 0; i < 5; i++ {
			process := tick - i - 3
			if process <= 0 {
				continue
			}
			for effect.NinjaWind.born <= i {
				birth := effect.NinjaWind.born
				origin := effect.ActorOrigin
				if effect.ActorID != 0 {
					origin = EffectWorldAnchor(ctx, Instance{ActorID: effect.ActorID}, effect.Starts.Add(time.Duration(birth)*db.EffectFrameDuration))
				}
				effect.NinjaWind.origins[birth] = origin
				effect.NinjaWind.born++
			}
			origin := effect.NinjaWind.origins[i]
			direction := scene.Sub3(scene.Add3(effect.ActorOrigin, effect.ProjectileDirection), origin)
			direction.Y = 0
			if length := math.Hypot(direction.X, direction.Z); length > 0 {
				direction = scene.Mul3(direction, 1/length)
			}
			heading := math.Atan2(direction.Z, direction.X)
			switch i {
			case 1:
				origin.X++
			case 2:
				origin.X--
			case 3:
				origin.Z++
			case 4:
				origin.Z--
			}
			origin.Y += float64(10-HitEffectRandom(effect, 30+i, 5)) / 5

			for j := 0; j < 4; j++ {
				sample := process
				if j > 0 {
					sample = min(process, 29)/3*3 - (j-1)*3
				}
				if sample <= 0 {
					continue
				}
				rise, fall := min(sample-1, 9), max(0, sample-10)
				alpha := max(0, 6*rise-3*fall)
				for n := 0; n < j; n++ {
					alpha = alpha * 8 / 10
				}
				distance := 1.5 * (1 + 1.02*(math.Pow(1.02, float64(rise))-1)/.02 + math.Pow(1.02, 9)*1.04*(math.Pow(1.04, float64(fall))-1)/.04) / 5
				position := scene.Add3(origin, scene.Mul3(direction, distance))
				angle := heading
				if j > 0 {
					angle = scene.DegreesToRadians(float64(HitEffectRandom(effect, 40+i*4+j, 360)))
				}
				quad(position, sample*30, alpha, 4, angle, true)
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
