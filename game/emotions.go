package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func (m *WorldMode) applyEmotionNotify(ctx client.Context, notify network.EmotionNotify) {
	if ctx.Config.Headless {
		return
	}
	frame, ok := db.EmotionSpriteFrame(notify.Type)
	if !ok {
		return
	}
	now := time.Now()
	duration := m.emotionDuration(ctx, frame)
	m.effects().Remove(effects.Emotion, notify.GID)
	x, y, ok := effects.EffectAnchor(ctx, notify.GID)
	if !ok {
		return
	}
	m.effects().Active = append(m.effects().Active, effects.Instance{
		EffectID:            effects.Emotion,
		ActorID:             notify.GID,
		X:                   x,
		Y:                   y,
		Starts:              now,
		Expires:             now.Add(duration),
		Duration:            duration,
		SpriteFrameOverride: frame,
		HasSpriteFrame:      true,
	})
	glog.Debugf("emotion actor=%d type=%d frame=%d", notify.GID, notify.Type, frame)
}

func (m *WorldMode) emotionDuration(ctx client.Context, frame int) time.Duration {
	view := m.effects().EffectSpriteView(ctx.Resources, "emotion")
	if view == nil || frame < 0 || frame >= len(view.Act.Actions) {
		return time.Second
	}
	action := view.Act.Actions[frame]
	delayMS := action.DelayMS
	if delayMS <= 0 {
		delayMS = 100
	}
	frames := len(action.Animations)
	if frames <= 0 {
		frames = 1
	}
	return time.Duration(float64(delayMS)*float64(frames)) * time.Millisecond
}
