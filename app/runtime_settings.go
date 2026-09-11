package app

import (
	"math"
	"sync/atomic"

	"github.com/gogpu/gogpu"
)

type runtimeSettings struct {
	fullscreen atomic.Bool
	vsync      atomic.Bool
	fps        atomic.Bool
	resScale   atomic.Uint32 // IEEE-754 bits; 0 = 1.0
}

func newRuntimeSettings(fullscreen, vsync, fps bool, resolutionScale float64) *runtimeSettings {
	settings := &runtimeSettings{}
	settings.fullscreen.Store(fullscreen)
	settings.vsync.Store(vsync)
	settings.fps.Store(fps)
	settings.SetResolutionScale(resolutionScale)
	return settings
}

func (s *runtimeSettings) Fullscreen() bool {
	if s == nil {
		return false
	}
	return s.fullscreen.Load()
}

func (s *runtimeSettings) SetFullscreen(value bool) {
	if s != nil {
		s.fullscreen.Store(value)
	}
}

func (s *runtimeSettings) VSync() bool {
	if s == nil {
		return false
	}
	return s.vsync.Load()
}

func (s *runtimeSettings) SetVSync(value bool) {
	if s != nil {
		s.vsync.Store(value)
	}
	// On the web this also retargets the frame pacing immediately, so the
	// settings checkbox takes effect without a restart; native builds keep
	// needing one until the surface is recreated.
	gogpu.SetBrowserVSync(value)
}

func (s *runtimeSettings) FPS() bool {
	if s == nil {
		return false
	}
	return s.fps.Load()
}

func (s *runtimeSettings) SetFPS(value bool) {
	if s != nil {
		s.fps.Store(value)
	}
}

// ResolutionScale returns the canvas backing-store scale (web-only effect).
func (s *runtimeSettings) ResolutionScale() float64 {
	if s == nil {
		return 1
	}
	scale := float64(math.Float32frombits(s.resScale.Load()))
	if !(scale > 0) || scale > 1 {
		return 1
	}
	return scale
}

// SetResolutionScale retargets the browser canvas backing store live: the
// next PrepareFrame resizes it and the surface follows, so the settings
// dropdown lands without a restart. Native builds keep the OS window size.
func (s *runtimeSettings) SetResolutionScale(value float64) {
	if !(value > 0) || value > 1 {
		value = 1
	}
	if s != nil {
		s.resScale.Store(math.Float32bits(float32(value)))
	}
	gogpu.SetBrowserResolutionScale(value)
}
