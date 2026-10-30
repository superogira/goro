package npc

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

type ProgressState struct {
	Started  time.Time
	Duration time.Duration
	Color    color.RGBA
}

func (m *System) StartProgress(progress network.ProgressBar, now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	m.Progress = ProgressState{
		Started:  now,
		Duration: progress.Duration,
		Color:    progressBarColor(progress.Color),
	}
}

func (m *System) ClearProgress() {
	m.Progress = ProgressState{}
}

// UpdateProgress owns both completion and input blocking. An attempted
// map move cancels the server work, but its click is consumed rather than
// becoming a movement request; other input remains blocked until completion.
func (m *System) UpdateProgress(ctx client.Context, now time.Time) bool {
	if m.Progress.Started.IsZero() {
		return false
	}
	if m.Progress.Duration <= 0 || !now.Before(m.Progress.Started.Add(m.Progress.Duration)) {
		m.FinishProgress(ctx, "complete")
		return false
	}
	if ctx.Input != nil {
		if ctx.Input.MouseJustPressed(input.MouseButtonLeft) && !m.bindings.PointerBlocked(ctx) {
			m.FinishProgress(ctx, "input")
		}
	}
	return true
}

func (m *System) FinishProgress(ctx client.Context, reason string) {
	if m.Progress.Started.IsZero() {
		return
	}
	m.Progress = ProgressState{}
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendProgressBarDone(); err != nil {
		glog.Warnf("progress bar %s acknowledgement failed: %v", reason, err)
	}
}

func progressBarColor(rgb uint32) color.RGBA {
	if rgb == 0 {
		return color.RGBA{G: 255, A: 255}
	}
	return color.RGBA{
		R: uint8(rgb >> 16),
		G: uint8(rgb >> 8),
		B: uint8(rgb),
		A: 255,
	}
}
