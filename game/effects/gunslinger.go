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

func (m *System) initGunslingerShot(ctx client.Context, effect *Instance) {
	destination := EffectWorldAnchor(ctx, *effect, effect.Starts)
	if _, ok, _ := scene.ActorForCombatID(ctx, effect.TargetID); effect.TargetID != 0 && ok {
		effect.ActorID, effect.TargetID = effect.TargetID, effect.ActorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
	}
	effect.ActorOrigin = EffectWorldAnchor(ctx, *effect, effect.Starts)
	effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
	effect.HasActorTransform = true
	if effect.EffectID == Chemical4 {
		return
	}
	yaw := m.cameraYaw(ctx)
	effect.SpriteFrameOverride = directedSpriteAction(scene.Mul3(effect.ProjectileDirection, -1), yaw)
	if effect.EffectID == SpreadAttack {
		effect.SpriteFrameOverride *= 2
	}
	effect.HasSpriteFrame = true
}

type gunslingerCoins struct {
	tick   int
	angles [5][7]int
}

func NewGunslingerCoins(effect Instance) *gunslingerCoins {
	p := &gunslingerCoins{}
	for i := range p.angles {
		for j := range p.angles[i] {
			p.angles[i][j] = HitEffectRandom(effect, i*7+j, 360)
		}
	}
	return p
}

func (p *gunslingerCoins) Advance(effect Instance, now time.Time) {
	end := max(0, int(now.Sub(effect.Starts)/db.EffectFrameDuration)+1)
	// Catch up cosmetic wandering for at most two seconds after a long pause.
	p.tick = max(p.tick, end-120)
	for p.tick < end {
		for i := range p.angles {
			a := &p.angles[i]
			for j := 0; j < 6; j += 2 {
				seed := (p.tick*5+i)*7 + j
				if a[j+1] > a[j] {
					a[j] = min(a[j]+1, 359)
					if a[j] > a[j+1] {
						a[j+1] = HitEffectRandom(effect, seed, max(a[j], 1))
					}
				} else {
					a[j] = max(a[j]-1, 0)
					if a[j] <= a[j+1] {
						a[j+1] = 359 - HitEffectRandom(effect, seed, max(a[j], 1))
					}
				}
			}
			a[6] = (a[6] + 5 + HitEffectRandom(effect, (p.tick*5+i)*7+6, 5)) % 360
		}
		p.tick++
	}
}

func (m *System) drawGunslingerCoins(screen *render.Frame, ctx client.Context, projection scene.Projection, c Component, effect Instance, x, y, z float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, c.TextureFile)
	p := effect.GunslingerCoins
	if texture == nil || p == nil || now.Before(effect.Starts) {
		return
	}
	p.Advance(effect, now)
	count := min(effect.HitCount, 5)
	for i := 0; i < count; i++ {
		a := p.angles[i]
		angle := scene.DegreesToRadians(float64(i*360/count + p.tick/2))
		cx := x + 1.4*math.Sin(angle) + .1*math.Sin(scene.DegreesToRadians(float64(a[0])))
		cy := y + 1.4*math.Cos(angle) + .1*math.Sin(scene.DegreesToRadians(float64(a[2])))
		cz := z + 3 - .1*math.Sin(scene.DegreesToRadians(float64(a[4])))
		radius := .2
		if i < effect.HitCount-5 {
			radius = .4
		}
		pulse := radius * .05 * math.Sin(scene.DegreesToRadians(float64(a[6])))
		alpha := uint8(min(p.tick, 200))
		outer, inner := math.Sqrt2*(radius+pulse), math.Sqrt2*(radius*.5+pulse)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, outer, outer, math.Pi/4, color.RGBA{68, 42, 30, alpha}, true)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, cx, cy, cz, inner, inner, math.Pi/4, color.RGBA{255, 255, 255, alpha}, true)
	}
}
