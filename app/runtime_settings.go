package app

import (
	"math"
	"sync/atomic"

	"github.com/gogpu/gogpu"
	"github.com/kivutar/goro/config"
)

type runtimeSettings struct {
	fullscreen       atomic.Bool
	vsync            atomic.Bool
	fps              atomic.Bool
	resScale         atomic.Uint32 // IEEE-754 bits; 0 = 1.0
	anisotropy       atomic.Int32
	textureUpscaling atomic.Bool
	smoothSprites    atomic.Bool
	msaa             atomic.Bool
	bloom            atomic.Bool
	ssao             atomic.Bool
}

func newRuntimeSettings(cfg config.Config) *runtimeSettings {
	settings := &runtimeSettings{}
	settings.fullscreen.Store(cfg.Window.Fullscreen)
	settings.vsync.Store(cfg.Render.VSync)
	settings.fps.Store(cfg.Render.FPS)
	settings.SetResolutionScale(cfg.Render.ResolutionScale)
	settings.SetAnisotropy(cfg.Render.Anisotropy)
	settings.SetTextureUpscaling(cfg.Render.TextureUpscaling)
	settings.SetSmoothSprites(cfg.Render.SmoothSprites)
	settings.SetMSAA(cfg.Render.MSAA)
	settings.SetBloom(cfg.Render.Bloom)
	settings.SetSSAO(cfg.Render.SSAO)
	return settings
}

func (s *runtimeSettings) TextureUpscaling() bool {
	return s != nil && s.textureUpscaling.Load()
}

func (s *runtimeSettings) SetTextureUpscaling(value bool) {
	if s != nil {
		s.textureUpscaling.Store(value)
	}
}

func (s *runtimeSettings) SmoothSprites() bool {
	return s == nil || s.smoothSprites.Load()
}

func (s *runtimeSettings) SetSmoothSprites(value bool) {
	if s != nil {
		s.smoothSprites.Store(value)
	}
}

func (s *runtimeSettings) Anisotropy() int {
	if s == nil {
		return 0
	}
	return int(s.anisotropy.Load())
}

func (s *runtimeSettings) SetAnisotropy(value int) {
	if s != nil {
		s.anisotropy.Store(int32(value))
	}
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

func (s *runtimeSettings) MSAA() bool {
	return s != nil && s.msaa.Load()
}

func (s *runtimeSettings) SetMSAA(value bool) {
	if s != nil {
		s.msaa.Store(value)
	}
}

func (s *runtimeSettings) Bloom() bool {
	return s != nil && s.bloom.Load()
}

func (s *runtimeSettings) SetBloom(value bool) {
	if s != nil {
		s.bloom.Store(value)
	}
}

func (s *runtimeSettings) SSAO() bool {
	return s != nil && s.ssao.Load()
}

func (s *runtimeSettings) SetSSAO(value bool) {
	if s != nil {
		s.ssao.Store(value)
	}
}
