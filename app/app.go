package app

import (
	"fmt"
	"time"

	"github.com/gogpu/gpucontext"
	gameaudio "github.com/kivutar/goro/audio"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/game"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/world"
)

type Game struct {
	cfg               config.Config
	input             *input.State
	resource          *res.Manager
	session           *session.Session
	world             *world.World
	network           *network.Client
	audio             *gameaudio.BGM
	modes             *game.Manager
	runtime           *runtimeSettings
	uiApp             client.UIApp
	ui                *gameui.Manager
	started           time.Time
	screenW           int
	screenH           int
	quit              func()
	quitting          bool
	pendingScreenshot string
	// Handheld audio state: side-key volume steps and the power-key mute.
	muted      bool
	mutedBGM   float64
	mutedSFX   float64
	enterPulse bool
}

func New(cfg config.Config) (*Game, error) {
	resource, err := res.NewManager(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("resource manager: %w", err)
	}
	if !cfg.Headless {
		loadClientUIFont(resource)
	}

	g := &Game{
		cfg:      cfg,
		input:    input.NewState(),
		resource: resource,
		session:  session.New(),
		world:    world.New(),
		network:  network.NewClient(cfg.Packet.ClientDate, cfg.Network.Trace),
		audio:    gameaudio.NewBGM(resource, cfg.Audio.BGM, cfg.Audio.BGMVolume, cfg.Audio.SFXVolume, cfg.Audio.Disabled, cfg.Audio.Backend),
		runtime:  newRuntimeSettings(cfg.Window.Fullscreen, cfg.Render.VSync, cfg.Render.FPS, cfg.Render.ResolutionScale),
		ui:       gameui.NewManager(),
		started:  time.Now(),
		screenW:  cfg.Window.Width,
		screenH:  cfg.Window.Height,
	}
	// Preferred resolver (e.g. 8.8.8.8 on the rg35xx, whose local DNS
	// NXDOMAINs DDNS hostnames); no-op when unset.
	network.SetPreferredDNS(cfg.Network.DNS)
	g.session.KeepLoginID = cfg.Login.KeepID
	g.session.SavedUsername = cfg.Login.SavedUsername
	g.session.NoShift = cfg.Gameplay.NoShift
	g.session.NoCtrl = cfg.Gameplay.NoCtrl
	g.session.LessEffects = cfg.Gameplay.LessEffects
	g.session.SnapTargets = cfg.Gameplay.SnapTargets
	g.session.SnapItems = cfg.Gameplay.SnapItems
	g.session.SnapRadius = cfg.Gameplay.SnapRadius
	if cfg.Gameplay.ForceUserAI {
		g.session.HomunculusCustomAI = true
		g.session.MercenaryCustomAI = true
	}

	ctx := g.modeContext()
	g.modes = game.NewManager(ctx, game.NewLoginMode())
	return g, nil
}

func (g *Game) Update() error {
	// Release the previous frame's A-as-Enter pulse (login mode): the press
	// edge must be visible for exactly one frame.
	if g.enterPulse {
		g.input.SetKey(input.KeyEnter, false)
		g.enterPulse = false
	}
	defer g.input.EndFrame()
	g.network.Pump()
	g.modes.UpdateContext(g.modeContext())
	// Drive resource prefetching: the web build warms sprite files between
	// frames so a first-seen actor never stalls the frame on network reads.
	res.PrefetchTick()
	return g.modes.Update()
}

func (g *Game) Draw(screen *render.Frame) {
	if g.cfg.Headless {
		return
	}
	g.modes.Draw(screen)
}

func (g *Game) DrawOverlay(screen *render.Frame) {
	if g.cfg.Headless {
		return
	}
	g.modes.DrawOverlay(screen)
}

func (g *Game) DrawUIOverlay(screen *render.Frame) {
	if g.cfg.Headless {
		return
	}
	g.modes.DrawUIOverlay(screen)
}

func (g *Game) FrameSubmitted() {
	g.modes.FrameSubmitted()
}

func (g *Game) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		g.screenW = g.cfg.Window.Width
		g.screenH = g.cfg.Window.Height
		return
	}
	g.screenW = width
	g.screenH = height
}

func (g *Game) InputState() *input.State {
	return g.input
}

// SuppressUI reports whether the widget UI layer should be skipped this
// frame (?noui=1). Only world mode suppresses — login and character select
// are unplayable without their windows, and the point of the switch is to
// measure the map's smoothness with zero UI raster cost.
func (g *Game) SuppressUI() bool {
	return g.cfg.Login.DebugNoUI && g.modes.ModeName() == "world"
}

func (g *Game) HandleKeyPress(code input.KeyCode) {
	if g.modes != nil {
		g.modes.HandleKeyPress(g.modeContext(), code)
	}
}

func (g *Game) PrepareTextInput(code input.KeyCode) bool {
	return g.modes != nil && g.modes.PrepareTextInput(g.modeContext(), code)
}

func (g *Game) PrepareKeyInput(code input.KeyCode, mods gpucontext.Modifiers) {
	// PrintScreen captures the current frame from any screen (title, login,
	// world). The rg35xx launcher binds a short MENU tap to this key.
	if code == gpucontext.KeyPrintScreen {
		if path, err := g.RequestScreenshot(); err == nil {
			glog.Infof("screenshot requested path=%s", path)
		} else {
			glog.Warnf("screenshot request failed: %v", err)
		}
	}
	// Handheld side keys: volume steps (both BGM and SFX together) and the
	// power-key screen cycle, whose F14/F16 edges mute/unmute the game.
	switch code {
	case gpucontext.KeyAudioVolumeUp:
		g.stepVolume(0.05)
	case gpucontext.KeyAudioVolumeDown:
		g.stepVolume(-0.05)
	case gpucontext.KeyF14:
		g.muteForScreenOff()
	case gpucontext.KeyF16:
		g.unmuteFromScreenOn()
	case gpucontext.KeyF13:
		// A doubles as Enter on the login screens (account/password/service/
		// character select all advance on Enter); in the world the gamepad
		// layer routes it instead.
		if g.modes != nil && g.modes.ModeName() == "login" {
			g.input.SetKey(input.KeyEnter, true)
			g.enterPulse = true
		}
	}
	if g.modes != nil {
		g.modes.PrepareKeyInput(g.modeContext(), code, mods)
	}
}

// stepVolume nudges both volume channels and clears any mute so the change
// is audible immediately; the on-screen volume HUD follows.
func (g *Game) stepVolume(delta float64) {
	if g.audio == nil {
		return
	}
	g.muted = false
	g.audio.SetBGMVolume(g.audio.BGMVolume() + delta)
	g.audio.SetSFXVolume(g.audio.SFXVolume() + delta)
	glog.Infof("volume changed bgm=%.2f sfx=%.2f", g.audio.BGMVolume(), g.audio.SFXVolume())
	render.ShowVolumeHUD(g.audio.BGMVolume(), false)
}

// muteForScreenOff silences the game when the power cycle blanks the panel.
func (g *Game) muteForScreenOff() {
	if g.audio == nil || g.muted {
		return
	}
	g.mutedBGM = g.audio.BGMVolume()
	g.mutedSFX = g.audio.SFXVolume()
	g.muted = true
	g.audio.SetVolume(0)
	glog.Infof("audio muted for screen off (was bgm=%.2f sfx=%.2f)", g.mutedBGM, g.mutedSFX)
}

// unmuteFromScreenOn restores the levels captured at mute time when the
// power cycle returns to the on state.
func (g *Game) unmuteFromScreenOn() {
	if g.audio == nil || !g.muted {
		return
	}
	g.muted = false
	g.audio.SetBGMVolume(g.mutedBGM)
	g.audio.SetSFXVolume(g.mutedSFX)
	glog.Infof("audio unmuted bgm=%.2f sfx=%.2f", g.mutedBGM, g.mutedSFX)
}

func (g *Game) SetQuitFunc(quit func()) {
	g.quit = quit
}

func (g *Game) SetUIApp(uiApp client.UIApp) {
	g.uiApp = uiApp
	if g.ui != nil {
		g.ui.SetUIApp(uiApp)
	}
}

func (g *Game) RequestQuit() {
	if g.quitting {
		return
	}
	g.quitting = true
	if g.network != nil {
		g.network.Close()
	}
	if g.audio != nil {
		g.audio.Stop()
	}
	if g.quit != nil {
		g.quit()
	}
}

func (g *Game) RequestScreenshot() (string, error) {
	path, err := config.NextScreenshotPath(time.Now())
	if err != nil {
		return "", err
	}
	g.pendingScreenshot = path
	return path, nil
}

func (g *Game) ConsumeScreenshotRequest() (string, bool) {
	if g.pendingScreenshot == "" {
		return "", false
	}
	path := g.pendingScreenshot
	g.pendingScreenshot = ""
	return path, true
}

func (g *Game) CompleteScreenshot(path string, err error) {
	if err != nil {
		glog.Errorf("screenshot failed path=%s error=%v", path, err)
		return
	}
	glog.Infof("screenshot saved path=%s", path)
}

func (g *Game) RuntimeFullscreen() bool {
	return g.runtime.Fullscreen()
}

func (g *Game) RuntimeVSync() bool {
	return g.runtime.VSync()
}

func (g *Game) RuntimeFPS() bool {
	return g.runtime.FPS()
}

func loadClientUIFont(resource *res.Manager) {
	regular, err := resource.ReadFileExact("System/Font/SCDream4.otf")
	if err != nil {
		return
	}
	bold, err := resource.ReadFileExact("System/Font/SCDream6.otf")
	if err != nil {
		glog.Warnf("ui font regular loaded but bold missing: %v", err)
	}
	if err := render.SetUIFont(regular, bold); err != nil {
		glog.Errorf("ui font load failed: %v", err)
		return
	}
	if len(bold) > 0 {
		glog.Infof("ui font loaded path=System/Font/SCDream4.otf bold=System/Font/SCDream6.otf")
	} else {
		glog.Infof("ui font loaded path=System/Font/SCDream4.otf")
	}
}

func (g *Game) modeContext() client.Context {
	return client.Context{
		Config:            g.cfg,
		Input:             g.input,
		Resources:         g.resource,
		Session:           g.session,
		World:             g.world,
		Network:           g.network,
		Audio:             g.audio,
		Started:           g.started,
		ScreenW:           g.screenW,
		ScreenH:           g.screenH,
		Runtime:           g.runtime,
		RequestQuit:       g.RequestQuit,
		RequestScreenshot: g.RequestScreenshot,
		UIApp:             g.uiApp,
		UIManager:         g.ui,
	}
}
