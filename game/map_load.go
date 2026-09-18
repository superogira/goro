package game

import (
	"image/color"
	"sync/atomic"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
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
	load := &mapLoadState{}
	m.showcase = nil

	character := ctx.Session.SelectedCharacter()
	visualCharacter := localPlayerVisualCharacter(ctx)
	hasMap := ctx.World.MapName != ""

	var rswLoaded bool
	steps := []struct {
		name string
		run  func() error
	}{
		{"showcase", func() error {
			// The loading-cover mascot: a random NPC or monster, picked
			// fresh for every map load. First on purpose — the cover shows
			// for the whole load, so the sooner it lands the better.
			m.showcase = loadLoadingShowcase(ctx.Resources)
			return nil
		}},
		{"sprites", func() error {
			playerStatus := ""
			if view, status := loadPlayerHumanoidSpriteView(ctx.Resources, visualCharacter, ctx.Session.Sex, localPlayerIsAdmin(ctx)); view != nil {
				m.playerView = view
				playerStatus = status
			} else {
				playerStatus = status
			}
			if view, status := loadActorShadowSpriteView(ctx.Resources); view != nil {
				m.shadowView = view
				if status != "" {
					playerStatus += " " + status
				}
			} else {
				m.shadowViewMiss = true
				glog.Warnf("actor shadow resources unavailable: %s", status)
			}
			if view, status := loadCursorSpriteView(ctx.Resources); view != nil {
				m.cursorView = view
				if status != "" {
					playerStatus += " " + status
				}
			} else {
				m.cursorViewMiss = true
				glog.Warnf("cursor resources unavailable: %s", status)
			}
			glog.Debugf("player sprite resources char_id=%d name=%s admin=%t job=%d visual_job=%d hair=%d weapon=%d shield=%d head_top=%d head_mid=%d head_low=%d body_pal=%d head_pal=%d hair_color=%d account_sex=%d %s", character.ID, character.Name, localPlayerIsAdmin(ctx), character.Job, visualCharacter.Job, character.Hair, character.Weapon, character.Shield, character.HeadTop, character.HeadMid, character.HeadLow, character.BodyPal, character.HeadPal, character.HairColor, ctx.Session.Sex, playerStatus)
			return nil
		}},
		{"terrain", func() error {
			if !hasMap {
				return nil
			}
			if gnd, _, err := loadGND(ctx.Resources, ctx.World.MapName); err == nil {
				ctx.World.GND = gnd
			} else {
				ctx.World.GND = nil
			}
			return nil
		}},
		{"scenery", func() error {
			if !hasMap {
				return nil
			}
			rsw, rswSource, err := loadRSW(ctx.Resources, ctx.World.MapName)
			if err != nil {
				ctx.World.RSW = nil
				ctx.World.RSM = nil
				ctx.World.RSMFail = 0
				m.prefetchMapTextures(ctx.Resources, ctx.World.GND, nil, nil)
				m.playMapBGM(ctx, ctx.World.MapName)
				return nil
			}
			ctx.World.RSW = rsw
			m.mapLoadRSWSource = rswSource
			rswLoaded = true
			return nil
		}},
		{"models", func() error {
			if !hasMap || !rswLoaded {
				return nil
			}
			ctx.World.RSM, ctx.World.RSMFail = loadRSMModels(ctx.Resources, ctx.World.RSW, defaultRSMLoadLimit)
			return nil
		}},
		{"textures", func() error {
			if !hasMap {
				return nil
			}
			if rswLoaded {
				m.prefetchMapTextures(ctx.Resources, ctx.World.GND, ctx.World.RSW, ctx.World.RSM)
			} else {
				m.prefetchMapTextures(ctx.Resources, ctx.World.GND, nil, nil)
			}
			return nil
		}},
		{"audio", func() error {
			if !hasMap {
				return nil
			}
			if rswLoaded {
				m.playMapBGM(ctx, m.mapLoadRSWSource)
				prefetchMapSoundFiles(ctx.Resources, ctx.World.RSW)
			} else {
				m.playMapBGM(ctx, ctx.World.MapName)
			}
			return nil
		}},
		{"ready", func() error {
			// Server-driven digit displays load their sprite on first use;
			// warm them with the rest of the map.
			ctx.Resources.Prefetch(serverDigitSpritePrefetchGroups()...)
			// The item tables (names, descriptions, slot counts, card
			// prefixes) parse lazily on the first item window open — a
			// ~100ms frame hitch on the handheld. Warm them here while the
			// loading cover is still up.
			ctx.Resources.WarmItemTables()
			if hasMap {
				_ = ctx.Network.SendLoadEndAck()
			}
			return nil
		}},
	}
	load.total = len(steps)
	m.mapLoad = load
	glog.Infof("map load started map=%s steps=%d", ctx.World.MapName, load.total)
	go func() {
		for _, step := range steps {
			if err := step.run(); err != nil {
				load.err = err
				break
			}
			load.done.Add(1)
		}
		load.finished.Store(true)
		glog.Infof("map load finished map=%s done=%d/%d err=%v", ctx.World.MapName, load.done.Load(), load.total, load.err)
	}()
}

// drawMapLoadProgress draws the progress bar on the loading cover. The cover
// itself (background image/black + showcase sprite) is drawn by
// drawLoadingScreen/drawLoadingShowcase first — an opaque fill here would
// erase them, which is exactly what hid the showcase sprite on the device.
func drawMapLoadProgress(screen *render.Frame, load *mapLoadState) {
	bounds := screen.Bounds()
	width, height := float64(bounds.Dx()), float64(bounds.Dy())

	fraction := load.progress()
	const barWidthFactor = 0.5
	const barHeight = 10
	const barBottomMargin = 48
	barWidth := width * barWidthFactor
	barX := (width - barWidth) / 2
	barY := height - barBottomMargin
	render.DrawRect(screen, barX, barY, barWidth, barHeight, color.RGBA{R: 255, G: 255, B: 255, A: 70})
	if fraction > 0 {
		render.DrawRect(screen, barX, barY, barWidth*fraction, barHeight, color.RGBA{R: 255, G: 255, B: 255, A: 230})
	}
}
