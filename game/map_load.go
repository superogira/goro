package game

import (
	maps "github.com/kivutar/goro/game/map"
	"sync/atomic"

	"github.com/kivutar/goro/client"
)

// mapLoadState tracks the background map load started by WorldMode.Enter.
// The heavy half of entering a map (sprite views, GND/RSW/RSM, texture
// prefetch, BGM) used to run on the game loop, blocking every frame — the
// player stared at a black screen with no feedback for seconds. The load
// now runs on a goroutine while the loop keeps rendering the loading screen
// with a progress bar; Update/Draw skip world logic until it finishes.
type mapLoadState struct {
	total    int
	done     atomic.Int32
	err      error
	finished atomic.Bool
}

func (l *mapLoadState) loading() bool {
	return l != nil && !l.finished.Load()
}

// progress reports completed steps over total steps, 0..1.
func (l *mapLoadState) progress() float64 {
	if l == nil || l.total == 0 {
		return 1
	}
	fraction := float64(l.done.Load()) / float64(l.total)
	if fraction < 0 {
		return 0
	}
	if fraction > 1 {
		return 1
	}
	return fraction
}

// startMapLoad moves the heavy tail of Enter onto a background goroutine.
// The captured client.Context holds pointers (resources, world, network)
// that outlive the per-frame context copies, so the loader keeps a valid
// view of them; the game loop owns none of the written state while the
// load runs because Update and Draw take the loading early-out.
func (m *WorldMode) startMapLoad(ctx client.Context) {
	// Upstream's maps.Loading owns the load stages; the fork contributes
	// only the cover mascot, composed up front.
	m.showcase = loadLoadingShowcase(ctx.Resources)
	m.loadingBG = nil
	m.startMapPrewarm()
}

func (m *WorldMode) startMapPrewarm() {
	if m.mapFade.Phase == maps.FadeNone {
		m.mapFade.Phase = maps.FadePrewarm
	}
}
