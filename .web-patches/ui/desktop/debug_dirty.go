package desktop

import (
	"os"
	"sync"
	"time"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/geometry"
)

var (
	debugDirtyOnce    sync.Once
	debugDirtyEnabled bool
)

func isDebugDirtyEnabled() bool {
	debugDirtyOnce.Do(func() {
		debugDirtyEnabled = os.Getenv("GOGPU_DEBUG_DIRTY") == "1"
	})
	return debugDirtyEnabled
}

const dirtyFlashDuration = 400 * time.Millisecond

type dirtyFlash struct {
	rect geometry.Rect
	time time.Time
}

// dirtyOverlay tracks dirty regions with flash-and-fade effect.
// Android SurfaceFlinger pattern: flash on dirty, fade over duration.
// In debug mode, extra frames are requested for the fade animation.
type dirtyOverlay struct {
	flashes []dirtyFlash
}

func (o *dirtyOverlay) update(regions []geometry.Rect) {
	now := time.Now()

	// Prune expired.
	alive := o.flashes[:0]
	for _, f := range o.flashes {
		if now.Sub(f.time) < dirtyFlashDuration {
			alive = append(alive, f)
		}
	}
	o.flashes = alive

	// Add new.
	for _, r := range regions {
		if r.Width() <= 0 || r.Height() <= 0 {
			continue
		}
		o.flashes = append(o.flashes, dirtyFlash{rect: r, time: now})
	}
}

func (o *dirtyOverlay) draw(cc *gg.Context, scale float64) {
	now := time.Now()
	for _, f := range o.flashes {
		age := now.Sub(f.time)
		if age >= dirtyFlashDuration {
			continue
		}
		fade := 1.0 - float64(age)/float64(dirtyFlashDuration)

		x := float64(f.rect.Min.X) * scale
		y := float64(f.rect.Min.Y) * scale
		w := float64(f.rect.Max.X-f.rect.Min.X) * scale
		h := float64(f.rect.Max.Y-f.rect.Min.Y) * scale
		if w <= 0 || h <= 0 {
			continue
		}

		cc.SetRGBA(0, 0.7, 0.9, 0.12*fade)
		cc.DrawRectangle(x, y, w, h)
		_ = cc.Fill()

		cc.SetRGBA(0, 0.7, 0.9, 0.7*fade)
		cc.SetLineWidth(2)
		cc.DrawRectangle(x+1, y+1, w-2, h-2)
		_ = cc.Stroke()
	}
}

func (o *dirtyOverlay) needsAnimationFrame() bool {
	if len(o.flashes) == 0 {
		return false
	}
	now := time.Now()
	for _, f := range o.flashes {
		if now.Sub(f.time) < dirtyFlashDuration {
			return true
		}
	}
	return false
}
