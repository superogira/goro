package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) startMapFadeOut(change network.MapChange, now time.Time) {
	m.ui.npcCutin.Clear()
	if m.mapFade.phase == mapFadeHold || m.mapFade.phase == mapFadePrewarm {
		m.mapFade = mapFadeState{
			phase:     mapFadeHold,
			change:    change,
			hasChange: true,
		}
		return
	}
	m.mapFade = mapFadeState{
		phase:     mapFadeOut,
		started:   now,
		change:    change,
		hasChange: true,
	}
}

func (m *WorldMode) startCharacterSelectFadeOut(now time.Time) {
	m.ui.npcCutin.Clear()
	m.mapFade = mapFadeState{
		phase:           mapFadeOut,
		started:         now,
		characterSelect: true,
	}
}

func (m *WorldMode) startMapFadeIn(now time.Time) {
	m.mapFade = mapFadeState{phase: mapFadeIn, started: now}
}

func (m *WorldMode) startMapPrewarm() {
	m.mapFade = mapFadeState{phase: mapFadePrewarm}
}

func (m *WorldMode) recordCoveredMapFrame() {
	switch m.mapFade.phase {
	case mapFadeHold:
		if m.mapFade.coveredFrames < mapFadeHandoffFrames {
			m.mapFade.coveredFrames++
		}
	case mapFadePrewarm:
		if m.mapFade.coveredFrames < mapFadePrewarmFrames {
			m.mapFade.coveredFrames++
		}
	}
}

func (m *WorldMode) advanceMapPrewarm(now time.Time) {
	if m.mapFade.phase == mapFadePrewarm && m.mapFade.coveredFrames >= mapFadePrewarmFrames {
		m.startMapFadeIn(now)
	}
}

func (m *WorldMode) mapFadeElapsed(now time.Time) bool {
	switch m.mapFade.phase {
	case mapFadeOut:
		return now.Sub(m.mapFade.started) >= mapFadeOutDuration
	case mapFadeIn:
		return now.Sub(m.mapFade.started) >= mapFadeInDuration
	default:
		return false
	}
}

func (m *WorldMode) mapFadeAlpha(now time.Time) uint8 {
	switch m.mapFade.phase {
	case mapFadeHold, mapFadePrewarm:
		return 255
	case mapFadeOut:
		if m.mapFade.started.IsZero() {
			return 0
		}
		return clampColor(255 * clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeOutDuration)))
	case mapFadeIn:
		if m.mapFade.started.IsZero() {
			return 0
		}
		return clampColor(255 * (1 - clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeInDuration))))
	default:
		return 0
	}
}

func (m *WorldMode) drawMapFade(ctx client.Context, screen *render.Frame, now time.Time) {
	alpha := m.mapFadeAlpha(now)
	if alpha == 0 {
		return
	}
	// Hold and prewarm are the phases where the screen sits fully covered
	// while map data loads — show the loading screen (random background
	// image when configured, plus the Now Loading label).
	if m.mapFade.phase == mapFadeHold || m.mapFade.phase == mapFadePrewarm {
		drawLoadingScreen(screen, m.loadingBackground(ctx))
		return
	}
	bounds := screen.Bounds()
	render.DrawRect(screen, 0, 0, float64(bounds.Dx()), float64(bounds.Dy()), color.RGBA{A: alpha})
}

// loadingBackground lazily picks a random loading image for this cover.
// Each WorldMode instance re-rolls (map changes create a new mode).
func (m *WorldMode) loadingBackground(ctx client.Context) *render.Image {
	if m.loadingBG != nil {
		return m.loadingBG
	}
	m.loadingBG = loadRandomBackground(ctx.Resources, ctx.Config.Background.LoadingPool)
	return m.loadingBG
}
