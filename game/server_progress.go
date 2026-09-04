package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

type serverProgressState struct {
	started  time.Time
	duration time.Duration
	color    color.RGBA
}

func (m *WorldMode) startServerProgress(ctx client.Context, progress network.ProgressBar, now time.Time) {
	if now.IsZero() {
		now = time.Now()
	}
	m.serverProgress = serverProgressState{
		started:  now,
		duration: progress.Duration,
		color:    progressBarColor(progress.Color),
	}
	// A progress bar is a server-owned WIP_DISABLE_ALL interval. Drop queued
	// character work so it cannot fire when that interval ends.
	m.cancelAttackIntent()
	m.pendingPickup = pickupIntent{}
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	m.pendingPetCapture = petCaptureState{}
	m.clearLocalActorAction(ctx)
}

func (m *WorldMode) clearServerProgress() {
	m.serverProgress = serverProgressState{}
}

// updateServerProgress owns both completion and input blocking. An attempted
// map move cancels the server work, but its click is consumed rather than
// becoming a movement request; other input remains blocked until completion.
func (m *WorldMode) updateServerProgress(ctx client.Context, now time.Time) bool {
	if m.serverProgress.started.IsZero() {
		return false
	}
	if m.serverProgress.duration <= 0 || !now.Before(m.serverProgress.started.Add(m.serverProgress.duration)) {
		m.finishServerProgress(ctx, "complete")
		return false
	}
	if ctx.Input != nil {
		if ctx.Input.MouseJustPressed(input.MouseButtonLeft) && !m.mapPointerBlocked(ctx) {
			m.finishServerProgress(ctx, "input")
		}
	}
	return true
}

func (m *WorldMode) finishServerProgress(ctx client.Context, reason string) {
	if m.serverProgress.started.IsZero() {
		return
	}
	m.serverProgress = serverProgressState{}
	if ctx.Network == nil {
		return
	}
	if err := ctx.Network.SendProgressBarDone(); err != nil {
		glog.Warnf("progress bar %s acknowledgement failed: %v", reason, err)
	}
}

func (m *WorldMode) serverProgressBar(localPlayer bool, now time.Time) (actorCastBar, bool) {
	if m.serverProgress.started.IsZero() || !localPlayer {
		return actorCastBar{}, false
	}
	bar := actorCastBar{
		started:  m.serverProgress.started,
		duration: m.serverProgress.duration,
		color:    m.serverProgress.color,
	}
	_, active := actorCastBarProgress(bar, now)
	return bar, active
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
