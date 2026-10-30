package maps

import (
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
)

func SoundLoadTasks(ctx client.Context) []func() {
	if ctx.Audio == nil || ctx.World == nil || ctx.World.RSW == nil {
		return nil
	}
	var tasks []func()
	for _, sound := range ctx.World.RSW.Sounds {
		if sound.Volume <= 0 {
			continue
		}
		tasks = append(tasks, func() {
			if err := ctx.Audio.PreloadSFX(sound.File); err != nil {
				glog.Debugf("preload map sound %s: %v", sound.File, err)
			}
		})
	}
	return tasks
}

func (m *System) ProcessMapSounds(ctx client.Context, now time.Time, schedule func(time.Time, float64, float64, float64, ...string)) {
	if ctx.World == nil || ctx.World.RSW == nil || ctx.World.GND == nil || len(ctx.World.RSW.Sounds) == 0 {
		return
	}
	playerX, playerY := scene.ActorRenderPosition(ctx.World.Player, now)
	width := float64(ctx.World.GND.Width)
	height := float64(ctx.World.GND.Height)
	if m.MapSoundNext == nil {
		m.MapSoundNext = make(map[int]time.Time)
	}
	for index, sound := range ctx.World.RSW.Sounds {
		if strings.TrimSpace(sound.File) == "" {
			continue
		}
		if sound.Volume <= 0 {
			continue
		}
		if next := m.MapSoundNext[index]; !next.IsZero() && now.Before(next) {
			continue
		}
		soundX := float64(sound.Position.X) + width
		soundY := float64(sound.Position.Z) + height
		maxDistance := float64(sound.Range)*0.2 + float64(sound.Height)
		if math.Hypot(soundX-playerX, soundY-playerY) > maxDistance {
			continue
		}
		schedule(now, float64(sound.Volume), soundX, soundY, sound.File)
		delay := time.Duration(float64(time.Second) * float64(sound.Cycle))
		if delay < 100*time.Millisecond {
			delay = 100 * time.Millisecond
		}
		m.MapSoundNext[index] = now.Add(delay)
	}
}
