package maps

import (
	"image/color"
	"math"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

type FadePhase int

const (
	FadeNone FadePhase = iota
	FadeOut
	FadeHold
	FadePrewarm
	FadeIn
)

type Fade struct {
	Phase           FadePhase
	Started         time.Time
	CoveredFrames   int
	Change          network.MapChange
	HasChange       bool
	CharacterSelect bool
}

const (
	FadeOutDuration = 220 * time.Millisecond
	FadeInDuration  = 340 * time.Millisecond
	// Present the transition cover before blocking on synchronous map loading.
	FadeHandoffFrames = 1
	// Warm the base map, then allow a frame for the initial post-ack actors.
	FadePrewarmFrames = 2
)

func (f *Fade) StartOut(change network.MapChange, now time.Time) {
	if f.Phase == FadeHold || f.Phase == FadePrewarm {
		*f = Fade{
			Phase:     FadeHold,
			Change:    change,
			HasChange: true,
		}
		return
	}
	*f = Fade{
		Phase:     FadeOut,
		Started:   now,
		Change:    change,
		HasChange: true,
	}
}

func (f *Fade) StartCharacterSelectOut(now time.Time) {
	*f = Fade{
		Phase:           FadeOut,
		Started:         now,
		CharacterSelect: true,
	}
}

func (f *Fade) StartIn(now time.Time) {
	*f = Fade{Phase: FadeIn, Started: now}
}

func (f *Fade) StartPrewarm() {
	*f = Fade{Phase: FadePrewarm}
}

func (f *Fade) RecordCoveredFrame() {
	switch f.Phase {
	case FadeHold:
		if f.CoveredFrames < FadeHandoffFrames {
			f.CoveredFrames++
		}
	case FadePrewarm:
		if f.CoveredFrames < FadePrewarmFrames {
			f.CoveredFrames++
		}
	}
}

func (f *Fade) AdvancePrewarm(now time.Time, uploadsPending bool) {
	if f.Phase == FadePrewarm && !uploadsPending && f.CoveredFrames >= FadePrewarmFrames {
		f.StartIn(now)
	}
}

func (f *Fade) Elapsed(now time.Time) bool {
	switch f.Phase {
	case FadeOut:
		return now.Sub(f.Started) >= FadeOutDuration
	case FadeIn:
		return now.Sub(f.Started) >= FadeInDuration
	default:
		return false
	}
}

func (f *Fade) Alpha(now time.Time) uint8 {
	switch f.Phase {
	case FadeHold, FadePrewarm:
		return 255
	case FadeOut:
		if f.Started.IsZero() {
			return 0
		}
		return scene.ClampColor(255 * clampUnit(float64(now.Sub(f.Started))/float64(FadeOutDuration)))
	case FadeIn:
		if f.Started.IsZero() {
			return 0
		}
		return scene.ClampColor(255 * (1 - clampUnit(float64(now.Sub(f.Started))/float64(FadeInDuration))))
	default:
		return 0
	}
}

func (f *Fade) Draw(screen *render.Frame, now time.Time) {
	alpha := f.Alpha(now)
	if alpha == 0 {
		return
	}
	bounds := screen.Bounds()
	render.DrawRect(screen, 0, 0, float64(bounds.Dx()), float64(bounds.Dy()), color.RGBA{A: alpha})
}

func clampUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
