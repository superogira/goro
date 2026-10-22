package effects

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/world"
)

func (m *System) LordKnightSeparateEquipment(ctx client.Context, actor world.Actor, now time.Time) bool {
	if actor.Opt3State&db.Opt3Aurablade != 0 {
		return true
	}
	for _, effect := range m.Active {
		if effect.ActorID != actor.ID && !(scene.IsLocalActor(ctx, actor.ID) && scene.IsLocalActor(ctx, effect.ActorID)) {
			continue
		}
		if now.Before(effect.Starts) || effect.Expired(now) {
			continue
		}
		switch effect.EffectID {
		case RedBody, PierceBody, Bash3D3, Bash3D4:
			return true
		}
	}
	return false
}

func (m *System) drawSpiralPierceEffect(screen *render.Frame, ctx client.Context, component Component, effect Instance, now time.Time) {
	frame := int(now.Sub(effect.Starts) / db.EffectFrameDuration)
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	origin := EffectWorldAnchor(ctx, effect, now)
	origin.Y += .6
	var vertices []render.Vertex3D
	var indices []uint16
	for birth := 27; birth <= 54; birth += 3 {
		n := frame - birth + 1
		if n <= 0 || n > 10 {
			continue
		}
		offset := float64(birth - 25)
		for layer := 0; layer < 2; layer++ {
			alpha := min(n, 8)*(10-5*layer) - max(0, n-8)*30
			if alpha <= 0 {
				continue
			}
			rise := scene.DegreesToRadians(70 - float64(layer*8) - offset*2)
			height := (20 - offset*.7) / 5
			appendEffectBand(&vertices, &indices, texture, origin, (5+offset*.2)/5,
				scene.DegreesToRadians(float64(layer*9)+offset*6), 2*math.Pi, 1/math.Tan(rise), 20,
				color.RGBA{255, 255, 255, uint8(alpha)}, func(int) float64 { return height * math.Sin(rise) })
		}
	}
	if len(indices) > 0 {
		screen.DrawTriangles3DOwned(vertices, indices, texture, TexturedEffectBillboardDrawOptions(true, false))
	}
}
