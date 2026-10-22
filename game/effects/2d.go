package effects

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

func (m *System) draw2DEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, componentIndex int, worldX, worldY, worldZ, progress float64, now time.Time) {
	texture := m.EffectFileTexture(ctx.Resources, component.TextureFile)
	if texture == nil {
		return
	}
	salt := componentIndex * 1009
	if component.durationRandMax > 0 {
		duration := deterministicDurationRange(effect, salt+19, component.durationRandMin, component.durationRandMax)
		progress = WorldEffectComponentProgress(effect.Starts.Add(component.Delay), duration, now)
		if progress >= 1 {
			return
		}
	}
	alpha := effectBillboardAlpha(progress, component)
	if alpha <= 0 {
		return
	}
	offsetX, offsetY, offsetZ := m.Effect3DOffset(ctx, component, effect, salt, 0, progress, worldX, worldY, worldZ)
	sizeX, sizeY := Effect3DSize(component, effect, salt, progress, 0)
	if sizeX <= 0 || sizeY <= 0 {
		return
	}
	angle := WorldEffectBillboardAngleForEffect(component, projection, effect, salt, progress)
	DrawTexturedEffectBillboardRotatedXYWithOptions(screen, projection, texture, worldX+offsetX, worldY+offsetY, worldZ+offsetZ, sizeX, sizeY, angle, effectComponentTint(component, alpha), TexturedEffectBillboardDrawOptions(true, component.Overlay))
}

// These primitives use the 2008 client's 60 Hz updates, including the first
// update before drawing. Fade-out starts from the opacity reached at that tick.
func effectParticleAlpha(frame, life, rise, fade int, peak float64) uint8 {
	if frame < 0 || frame >= life {
		return 0
	}
	alpha := peak * math.Min(1, float64(frame+1)/float64(rise))
	if frame >= fade {
		alpha = peak * math.Min(1, float64(fade)/float64(rise)) * float64(life-frame-1) / float64(life-fade)
	}
	return uint8(alpha)
}

type effectScreenParticle struct {
	x, Y, halfWidth, halfHeight, angle float64
	Alpha                              uint8
}

func drawEffectScreenParticle(screen *render.Frame, texture *render.Image, origin scene.Point2, particle effectScreenParticle) {
	if texture == nil || particle.Alpha == 0 || particle.halfWidth <= 0 || particle.halfHeight <= 0 {
		return
	}
	w, h := float64(texture.Bounds().Dx()), float64(texture.Bounds().Dy())
	options := &render.DrawImageOptions{Filter: render.FilterLinear}
	options.GeoM.Translate(-w/2, -h/2)
	options.GeoM.Scale(particle.halfWidth*2/w, particle.halfHeight*2/h)
	options.GeoM.Rotate(particle.angle)
	options.GeoM.Translate(float64(origin.X)+particle.x, float64(origin.Y)+particle.Y)
	options.ColorScale.ScaleAlpha(float32(particle.Alpha) / 255)
	screen.DrawImage(texture, options)
}
