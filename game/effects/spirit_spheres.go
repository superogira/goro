package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func (m *System) drawSpiritSphereEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, component Component, effect Instance, x, y, z float64, now time.Time) {
	if effect.EffectID == Chookgi3 {
		m.drawGunslingerCoins(screen, ctx, projection, component, effect, x, y, z, now)
		return
	}
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	count := scene.MaxInt(component.Duplicate, 1)
	if effect.Persistent {
		actor, ok, _ := scene.ActorForCombatID(ctx, effect.ActorID)
		if !ok {
			return
		}
		// Monks have up to five; Gunslingers have ten. Bound malformed packets
		// without changing the authoritative count kept on the actor.
		count = min(int(actor.SpiritSpheres), 10)
	}
	elapsedFrames := math.Max(0, now.Sub(effect.Starts).Seconds()*60)
	alpha := scene.ClampFloat(elapsedFrames*0.005, 0, 1)
	for i := 0; i < count; i++ {
		angle := elapsedFrames*math.Pi/180 + float64(i)*2*math.Pi/float64(count)
		radius := 0.55
		worldX := x + math.Cos(angle)*radius
		worldY := y + math.Sin(angle)*radius
		worldZ := z + 1.2 + math.Sin(angle*0.5)*0.12
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, worldX, worldY, worldZ, 0.35, 0.35, angle, color.RGBA{R: 0, G: 0, B: 255, A: uint8(153 * alpha)}, true)
		drawTexturedEffectBillboardRotatedXY(screen, projection, texture, worldX, worldY, worldZ+0.02, 0.21, 0.21, -angle, color.RGBA{R: 204, G: 204, B: 255, A: uint8(255 * alpha)}, true)
	}
}
