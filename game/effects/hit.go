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

func (m *System) drawHitRingEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	distance, alpha := hitRingAnimation(component, now.Sub(effect.Starts)-component.Delay)
	if alpha <= 0 {
		return
	}
	texture := m.effectTexture(ctx.Resources, component.TextureName)
	if texture == nil {
		return
	}
	if effect.HasActorTransform {
		x, y, z = effect.ActorOrigin.X, effect.ActorOrigin.Z, effect.ActorOrigin.Y
	}
	// Hit1 captures 180 - master.roty; DirToDeg(dir) is 180 + dir*45.
	// Keep this orientation even if the target turns during the impact.
	yaw := -45 * float64(scene.NormalizeDirectionIndex(effect.ActorDirection))
	right := RotateEffectCylinderVector(scene.Point3{X: 1}, component.AngleX, yaw, 0)
	depth := RotateEffectCylinderVector(scene.Point3{Z: 1}, component.AngleX, yaw, 0)
	axis := RotateEffectCylinderVector(scene.Point3{Y: 1}, component.AngleX, yaw, 0)
	n := float64((now.Sub(effect.Starts)-component.Delay)/component.FrameDelay + 1)
	motion := component.hitRing
	height := component.Height + n*motion.HeightSpeed + motion.HeightAcceleration*n*(n+1)/2
	if motion.RandomImpactOffset {
		// SonicBlowHit divides integer random offsets before converting to world units.
		x += float64((1+HitEffectRandom(effect, 601, 10))/4) / 5
		z -= float64((1+HitEffectRandom(effect, 602, 20))/2) / 5
	}
	x += axis.X * distance
	y += axis.Z * distance
	z += component.PosZ + axis.Y*distance
	// Prim3DCylinder advances U by 1/4 per face and uses RF_ALPHA.
	drawWorldCylinderBandWithOptions(screen, m.whitePixel, texture, x, y, z,
		component.BottomSize, component.TopSize, height,
		effectComponentTint(component, alpha), component.CircleSides, right, depth, axis,
		float64(component.CircleSides)/4, render.BlendSourceOver)
}

func hitRingAnimation(component Component, elapsed time.Duration) (distance, alpha float64) {
	if elapsed < 0 || elapsed >= component.Duration || component.FrameDelay <= 0 || component.hitRing == nil {
		return 0, 0
	}
	frames := float64(component.Duration / component.FrameDelay)
	if frames < 2 {
		return 0, 0
	}
	// Sakexe Prim3DCylinder: acceleration is -speed/(2*duration).
	// Acceleration and movement run before the first draw. One native unit is .2 cells.
	n := float64(elapsed/component.FrameDelay + 1)
	speed := component.hitRing.Speed
	acceleration := -speed / (2 * frames)
	distance = component.hitRing.InitialDistance + n*speed + acceleration*n*(n+1)/2
	// ProcessAlpha applies its first decrement on FadeOutFrame, before drawing.
	alpha = component.AlphaMax * scene.ClampFloat((frames-n)/math.Max(1, frames-float64(component.hitRing.FadeOutFrame)), 0, 1)
	return distance, alpha
}

func HitEffectRandom(effect Instance, salt, count int) int {
	return min(count-1, int(deterministicUnit(effect, salt)*float64(count)))
}

type hitParticle struct {
	axis                scene.Point3
	radius, speed, size float64
	frames              int
}

func newHitParticle(effect Instance, component Component, index int) hitParticle {
	rand := func(salt, count int) int { return HitEffectRandom(effect, index*17+salt, count) }
	yaw := scene.DegreesToRadians(float64(-45*scene.NormalizeDirectionIndex(effect.ActorDirection) - 40 + rand(1, 80)))
	pitch := scene.DegreesToRadians(float64(-50 + rand(2, 100)))
	// The original multiplies Y then X, then we flip its downward vertical axis.
	return hitParticle{
		axis:   scene.Point3{X: math.Sin(yaw), Y: math.Cos(yaw) * math.Sin(pitch), Z: math.Cos(yaw) * math.Cos(pitch)},
		radius: float64(4+rand(3, 4)) / 5,
		speed:  float64(component.hitParticles.SpeedMin+rand(4, component.hitParticles.SpeedRange)) / 500,
		size:   float64(60+rand(5, 100)) / 100,
		frames: 6 + rand(6, 24),
	}
}

func (p hitParticle) segment(frame, trail int, height float64) (position scene.Point3, size, alpha float64) {
	n := float64(frame + 1)
	life := scene.ClampFloat(1-n/float64(p.frames), 0, 1)
	size = p.size * life * (1 - float64(trail)/6)
	alpha = (254.0 / 255) * life * (1 - float64(trail)/3)
	// ShiftSegment stores two earlier positions, but fades/scales all three
	// using the current tick. Initial history is at the actor's feet.
	step := math.Max(0, n-float64(trail))
	if step > 0 {
		distance := p.radius + step*p.speed - p.speed/(2*float64(p.frames))*step*(step+1)/2
		position = scene.Add3(scene.Point3{Y: height}, scene.Mul3(p.axis, distance))
	}
	return position, size, alpha
}

func (m *System) drawHitParticlesEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, componentIndex int, x, y, z float64, now time.Time) {
	elapsed := now.Sub(effect.Starts) - component.Delay
	if elapsed < 0 || elapsed >= component.Duration || component.FrameDelay <= 0 || component.hitParticles == nil {
		return
	}
	view := m.EffectSpriteView(ctx.Resources, component.SpriteFile)
	if view == nil || len(view.Act.Actions) == 0 || len(view.Act.Actions[0].Animations) == 0 {
		return
	}
	frame := int(elapsed / component.FrameDelay)
	// AnimMotion advances immediately, then every four primitive ticks.
	motion := (frame/4 + 1) % len(view.Act.Actions[0].Animations)
	key := scene.SpriteBillboardKey{Motion: motion}
	billboard, ok := view.Billboards[key]
	if !ok {
		billboard, ok = scene.ComposeSingleSpriteBillboard(view, view.Act.Actions[0].Animations[motion])
		if !ok {
			return
		}
		view.Billboards[key] = billboard
	}
	options := scene.TriangleDrawOptions(render.FilterLinear, render.AddressClampToZero)
	// Hit3/Hit4 keep their CRagEffect origin at the impact point. The primitive
	// trails record earlier particle positions, independent of later actor movement.
	if effect.HasActorTransform {
		x, y, z = effect.ActorOrigin.X, effect.ActorOrigin.Z, effect.ActorOrigin.Y
	}
	for i := 0; i < component.Duplicate; i++ {
		particle := newHitParticle(effect, component, componentIndex*101+i)
		for trail := 0; trail < 3; trail++ {
			position, size, alpha := particle.segment(frame, trail, component.PosZ)
			if alpha <= 0 || size <= 0 {
				continue
			}
			scene.DrawSpriteBillboardTintAlphaWorld3DWithOptions(screen, projection, billboard,
				x+position.X, y+position.Z, z+position.Y, size*scene.PixelRatio, 0, alpha, 1,
				effectComponentTint(component, 1), options)
		}
	}
}

func initImpactParticles(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	origin := destination
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		origin = EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, effect.Starts)
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ActorOrigin = destination
	effect.ActorOrigin.Y += 1.6
	effect.ProjectileDirection = scene.Sub3(destination, origin)
	effect.HasActorTransform = true
}

func impactParticle(effect Instance, index, frame int) (position scene.Point3, size, angle float64, alpha int) {
	salt := index * 5
	age := frame + 1 - 40 + HitEffectRandom(effect, salt, 10)
	alpha = max(0, 15*scene.ClampInt(age-1, 0, 9)-3*max(0, age-10))
	if alpha == 0 {
		return
	}
	direction := math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X)
	direction += scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 60) - 30))
	speed := (.75 + .01*float64(HitEffectRandom(effect, salt+2, 50))) / 5
	position = scene.Add3(effect.ActorOrigin, scene.Point3{
		X: math.Cos(direction) * speed * float64(age),
		Z: math.Sin(direction) * speed * float64(age),
		Y: (1 - .01*float64(HitEffectRandom(effect, salt+3, 200))) * float64(age) / 5,
	})
	size = (.1 + .05*float64(HitEffectRandom(effect, salt+4, 16))) / 5 * math.Sqrt2
	angle = scene.DegreesToRadians(float64(-5*age + 45))
	return
}

func (m *System) drawImpactParticles(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	for i := 0; i < component.Duplicate; i++ {
		position, size, angle, alpha := impactParticle(effect, i, frame)
		if alpha > 0 {
			drawTexturedEffectBillboardRotatedXY(screen, projection, texture,
				position.X, position.Z, position.Y, size, size, angle, color.RGBA{130, 130, 250, uint8(alpha)}, true)
		}
	}
}

// BASH3D variants share this ray renderer across Paladin, Lord Knight, Champion, Ninja, and Gunslinger.
func (m *System) drawBashRays(screen *render.Frame, ctx client.Context, _ scene.Projection, component Component, effect Instance, now time.Time) {
	process := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1 - 24
	alpha := min(process, 10)*20 - max(0, process-12)*15
	if effect.EffectID == Bash3D2 || effect.EffectID == Kouenka {
		process += 24
		alpha = min(process, 10)*10 - max(0, process-11)*3
	}
	if process <= 0 || alpha <= 0 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	origin.Y += 2.4
	distance := .4 * math.Pow(1.15, float64(process))
	vertices, indices := []render.Vertex3D{}, []uint16{}
	count := 20
	widths := []float64{2, 5}
	if effect.EffectID == Bash3D2 || effect.EffectID == Kouenka {
		count, distance = 32, float64(2+3*process)/5
		widths = []float64{1, 2}
	}
	if effect.EffectID == Bash3D3 || effect.EffectID == Bash3D4 || effect.EffectID == Bash3D5 {
		count = 24
	}
	if effect.EffectID == Kouenka {
		count = 8
	}
	for i := 0; i < count; i++ {
		yaw := scene.DegreesToRadians(float64(90*(i%4) + 22*(i/4)))
		roll := deterministicAngle(effect, i)
		point := func(angle float64) scene.Point3 {
			r := distance * math.Cos(angle)
			return scene.Add3(origin, scene.Point3{X: r * math.Cos(roll), Y: -r * math.Sin(roll), Z: distance * math.Sin(angle)})
		}
		for layer, width := range widths {
			tint := color.RGBA{0, 250, 250, uint8(alpha)}
			if layer == 1 {
				tint = color.RGBA{250, 0, 0, uint8(alpha)}
			}
			if effect.EffectID == Bash3D2 || effect.EffectID == Kouenka || effect.EffectID == Bash3D3 {
				tint = color.RGBA{0, 0, 250, uint8(alpha)}
				if layer == 1 {
					tint = color.RGBA{250, 250, 0, uint8(alpha)}
				}
			} else if effect.EffectID == Bash3D4 || effect.EffectID == Bash3D5 {
				tint = color.RGBA{50, 50, 50, uint8(alpha)}
				if layer == 1 {
					tint = color.RGBA{250, 250, 250, uint8(alpha)}
				}
			}
			a, b := point(yaw-scene.DegreesToRadians(width)), point(yaw+scene.DegreesToRadians(width))
			appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{origin, origin, b, a}, tint)
			if count == 24 {
				// Head Crush and Joint Beat use alpha blending for the wider ray.
				screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(layer == 0, false))
				vertices, indices = nil, nil
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}

func (m *System) DrawHitLineBurst(screen *render.Frame, ctx client.Context, component Component, effect Instance, x, y, z float64, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame >= 200 {
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	var vertices []render.Vertex3D
	var indices []uint16
	// HitLine4(3) for Estun; HitLine3(3) for Esma.
	count, start, randomDelay, fixedDelay, baseRadius, heightBase := 16, 6, 31, 0, .05/5, .8
	if effect.EffectID == HitLine6 {
		count, start, randomDelay, fixedDelay, baseRadius, heightBase = 20, 0, 41, 30, .07/5, .6
	}
	if effect.EffectID == HitLine5 {
		count, start, randomDelay, fixedDelay, baseRadius = 12, 0, 21, 25, .06/5
	}
	for i := 0; i < count; i++ {
		salt := i * 5
		age := frame - (start + i/4) - fixedDelay - HitEffectRandom(effect, salt, randomDelay) + 1
		if age <= 0 || age >= 34 {
			continue
		}
		angle := deterministicAngle(effect, salt+1)
		if effect.EffectID == HitLine5 {
			angle = math.Atan2(effect.ProjectileDirection.Z, effect.ProjectileDirection.X) + scene.DegreesToRadians(float64(HitEffectRandom(effect, salt+1, 121)-60))
		}
		sin, cos := math.Sincos(angle)
		distance := float64(6+HitEffectRandom(effect, salt+2, 6)) / 5
		if effect.EffectID == HitLine5 {
			distance = float64(6+HitEffectRandom(effect, salt+2, 7)) / 5
		}
		height := heightBase + .1*float64(HitEffectRandom(effect, salt+3, 7))
		alpha := float64(125 + HitEffectRandom(effect, salt+4, 121))
		point := func(tick int) scene.Point3 {
			phase := float64(scene.ClampInt(tick, 0, 30) * 6)
			travel := phase / 90 * distance
			return scene.Point3{X: x + cos*travel, Y: z + 1 + distance*math.Sin(scene.DegreesToRadians(phase))*height, Z: y + sin*travel}
		}
		for trail := 0; trail < 4; trail++ {
			// PrimHitLine3 updates history after advancing its head, so the
			// first history slot has the same position as the head.
			a, b := point(age-max(0, trail-1)), point(age-trail)
			if a == b {
				continue
			}
			for layer := 0; layer < 2; layer++ {
				radius := baseRadius
				brightness := uint8(255 - (5-trail)*30)
				tint := color.RGBA{brightness, brightness, brightness, uint8(alpha - float64(trail*10))}
				if layer == 1 {
					radius *= 4
					tint = color.RGBA{0, 0, 255, tint.A / 2}
				}
				offset := func(side int) scene.Point3 {
					s, c := math.Sincos(scene.DegreesToRadians(float64(side * 120)))
					return scene.Point3{X: -sin * radius * c, Y: -radius * s, Z: cos * radius * c}
				}
				for side := 0; side < 3; side++ {
					p, q := offset(side), offset(side+1)
					appendEffectQuad(&vertices, &indices, texture, [4]scene.Point3{scene.Add3(a, q), scene.Add3(b, q), scene.Add3(b, p), scene.Add3(a, p)}, tint)
				}
			}
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
