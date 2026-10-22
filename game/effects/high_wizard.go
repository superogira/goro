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

func MagicCrasherFlashActive(elapsed time.Duration) bool {
	frame := elapsed / db.EffectFrameDuration
	return frame >= 30 && frame <= 60
}

func initSoulDrain(ctx client.Context, effect *Instance) {
	caster := EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ActorOrigin = caster
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorOrigin = EffectWorldAnchor(ctx, Instance{ActorID: effect.TargetID}, effect.Starts)
		effect.ActorOrigin.Y = caster.Y
	}
	effect.ProjectileDirection = scene.Sub3(caster, effect.ActorOrigin)
	effect.HasActorTransform = true
}

func SoulDrainPosition(effect Instance, tick int, roll float64) scene.Point3 {
	if tick == 0 {
		return scene.Add3(effect.ActorOrigin, effect.ProjectileDirection)
	}
	n := float64(tick)
	direction := scene.Normalize3(effect.ProjectileDirection)
	forward := scene.Sub3(scene.Mul3(effect.ProjectileDirection, n/60), scene.Mul3(direction, (2*n-n*(n+1)/30)/5))
	arc := (2*n - n*(n-1)/30) / 5
	s, c := math.Sincos(scene.DegreesToRadians(roll))
	p := scene.Add3(effect.ActorOrigin, forward)
	p.Y += 2 + arc*c
	p.X -= direction.Z * arc * s
	p.Z += direction.X * arc * s
	return p
}

func (m *System) drawSoulDrain(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	if now.Before(effect.Starts) || frame >= 60 {
		return
	}
	for i := 0; i < 3; i++ {
		tint := color.RGBA{uint8(160 + HitEffectRandom(effect, i*3, 81)), uint8(160 + HitEffectRandom(effect, i*3+1, 81)), 255, 254}
		size := 1.5 + .1*float64(HitEffectRandom(effect, i*3+2, 6))
		for trail := 0; trail < 7; trail++ {
			tint.A = uint8(254 * (1 - float64(trail)/7))
			p := SoulDrainPosition(effect, max(0, frame+1-trail), float64(-90+i*90))
			m.drawEffectParticleSprite(screen, ctx, projection, component.SpriteFile, p, frame/4+1, size*(1-float64(trail)/14), 0, tint)
		}
	}
}

// Prim3DQuadHorn2 alternates three descending and three ascending ticks.
// Each six-tick cycle has zero net travel; after tick 550 it retracts.
func gravitationTravel(tick int) float64 {
	if tick <= 0 {
		return 0
	}
	if tick == 1 {
		return 1.01
	}
	remainder := (min(tick, 550) - 2) % 6
	return 2.03 + [...]float64{0, -1.2, -2.4, -3.6, -2.41, -1.21}[remainder] - 1.2*float64(max(0, tick-550))
}

func (m *System) drawGravitation(screen *render.Frame, ctx client.Context, component Component, effect Instance, origin scene.Point3, now time.Time) {
	n := int(now.Sub(effect.Starts)/db.EffectFrameDuration) + 1
	if n <= 0 || n >= 2000 {
		return
	}
	origin.Y -= 2
	alpha := uint8(min(20, 2*(2000-n)))
	for group, file := range [...]string{component.TextureFile, component.TextureFiles[0]} {
		texture := m.EffectFileTexture(ctx.Resources, file)
		if texture == nil {
			continue
		}
		var vertices []render.Vertex3D
		var indices []uint16
		for i := group * 2; i < group*2+2; i++ {
			size := float64(300+HitEffectRandom(effect, i*3, 50)) / 100
			yaw := scene.DegreesToRadians(float64(HitEffectRandom(effect, i*3+1, 360)))
			pitch := scene.DegreesToRadians(float64(60 + HitEffectRandom(effect, i*3+2, 40)))
			appendEffectSpike(&vertices, &indices, texture, origin, size, 18, pitch, yaw, gravitationTravel(n), color.RGBA{255, 255, 255, alpha})
		}
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(false, false))
	}
}
