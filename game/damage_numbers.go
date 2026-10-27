package game

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
)

const (
	damageNumberSPR = "data\\sprite\\이팩트\\숫자.spr"
	damageNumberACT = "data\\sprite\\이팩트\\숫자.act"
	damageMsgSPR    = "data\\sprite\\이팩트\\msg.spr"
	damageMsgACT    = "data\\sprite\\이팩트\\msg.act"
)

var (
	damageFloaterWhite  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterYellow = color.RGBA{R: 230, G: 230, B: 38, A: 255}
	damageFloaterRed    = color.RGBA{R: 255, G: 64, B: 64, A: 255}
)

func (m *WorldMode) damageNumberSprite(ctx client.Context) *scene.SpriteView {
	if m.damageNumberView != nil || m.damageNumberMiss || ctx.Resources == nil {
		return m.damageNumberView
	}
	view, status := scene.LoadSpriteView(ctx.Resources,
		[]string{damageNumberACT, strings.ReplaceAll(damageNumberACT, "\\", "/")},
		[]string{damageNumberSPR, strings.ReplaceAll(damageNumberSPR, "\\", "/")},
		nil,
		"damage number",
	)
	if view == nil {
		m.damageNumberMiss = true
		glog.Warnf("damage number sprite unavailable: %s", status)
		return nil
	}
	m.damageNumberView = view
	glog.Debugf("damage number sprite resources %s", status)
	return view
}

func (m *WorldMode) damageNumberBillboard(ctx client.Context, text string) (*scene.Billboard, bool) {
	if text == "" || !allASCIIDigits(text) {
		return nil, false
	}
	if m.damageNumbers == nil {
		m.damageNumbers = make(map[string]*scene.Billboard)
	}
	if billboard, ok := m.damageNumbers[text]; ok {
		return billboard, true
	}
	view := m.damageNumberSprite(ctx)
	if view == nil {
		return nil, false
	}
	billboard, ok := composeDamageNumberBillboard(view, text)
	if !ok {
		return nil, false
	}
	m.damageNumbers[text] = billboard
	return billboard, true
}

func (m *WorldMode) damageMessageSprite(ctx client.Context) *scene.SpriteView {
	if m.damageMsgView != nil || m.damageMsgMiss || ctx.Resources == nil {
		return m.damageMsgView
	}
	view, status := scene.LoadSpriteView(ctx.Resources,
		[]string{damageMsgACT, strings.ReplaceAll(damageMsgACT, "\\", "/")},
		[]string{damageMsgSPR, strings.ReplaceAll(damageMsgSPR, "\\", "/")},
		nil,
		"damage message",
	)
	if view == nil {
		m.damageMsgMiss = true
		glog.Warnf("damage message sprite unavailable: %s", status)
		return nil
	}
	m.damageMsgView = view
	glog.Debugf("damage message sprite resources %s", status)
	return view
}

func (m *WorldMode) damageMessageBillboard(ctx client.Context, actionIndex, motion int) (*scene.Billboard, bool) {
	view := m.damageMessageSprite(ctx)
	if view == nil || view.Act == nil {
		return nil, false
	}
	if actionIndex < 0 || actionIndex >= len(view.Act.Actions) {
		return nil, false
	}
	action := view.Act.Actions[actionIndex]
	if motion < 0 || motion >= len(action.Animations) {
		return nil, false
	}
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func composeDamageNumberBillboard(view *scene.SpriteView, text string) (*scene.Billboard, bool) {
	const digitGap = 2
	digits := make([]*scene.Billboard, 0, len(text))
	totalWidth := 0
	maxHeight := 0
	for _, ch := range text {
		digit := int(ch - '0')
		billboard, ok := damageDigitBillboard(view, digit)
		if !ok {
			return nil, false
		}
		bounds := billboard.Image.Bounds()
		if len(digits) > 0 {
			totalWidth += digitGap
		}
		totalWidth += bounds.Dx()
		maxHeight = scene.MaxInt(maxHeight, bounds.Dy())
		digits = append(digits, billboard)
	}
	if totalWidth <= 0 || maxHeight <= 0 {
		return nil, false
	}
	target := render.NewImage(totalWidth, maxHeight)
	x := 0
	for index, digit := range digits {
		if index > 0 {
			x += digitGap
		}
		bounds := digit.Image.Bounds()
		var opts render.DrawImageOptions
		opts.GeoM.Translate(float64(x), float64((maxHeight-bounds.Dy())/2))
		opts.Filter = scene.SpriteCompositionFilter()
		target.DrawImage(digit.Image, &opts)
		x += bounds.Dx()
	}
	return &scene.Billboard{
		Image:   target,
		AnchorX: float64(totalWidth) * 0.5,
		AnchorY: float64(maxHeight) * 0.5,
	}, true
}

func damageDigitBillboard(view *scene.SpriteView, digit int) (*scene.Billboard, bool) {
	if view == nil || view.Act == nil || len(view.Act.Actions) == 0 || digit < 0 || digit > 9 {
		return nil, false
	}
	action := view.Act.Actions[0]
	if digit >= len(action.Animations) {
		return nil, false
	}
	key := scene.SpriteBillboardKey{ActionIndex: 0, Motion: digit}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, action.Animations[digit])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func allASCIIDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func damageFloaterPlacement(kind damageFloaterKind, progress float64) (dx, dy, zLift, scale, alpha float64) {
	progress = scene.ClampFloat(progress, 0, 1)
	alpha = 1 - progress
	switch kind {
	case damageFloaterCombo:
		scale = math.Min(progress, 0.05) * 70
		zLift = 7.0 + progress
	case damageFloaterRecoveryHP, damageFloaterRecoverySP:
		scale = math.Max((1-progress*2)*3, 0.8)
		zLift = 2.0
		if progress >= 0.4 {
			zLift += (progress - 0.4) * 5.0
		}
	case damageFloaterMiss:
		scale = 0.5
		zLift = 3.5 + progress*7.0
	default:
		scale = math.Max((1-progress)*4, 0)
		arc := math.Sin(-math.Pi/2+math.Pi*(0.5+progress*1.5)) * 5.0
		dx = progress * 4.0
		zLift = 2.0 + arc
	}
	return dx, dy, zLift, scale, alpha
}

func damageFloaterColor(kind damageFloaterKind, fallback color.RGBA) color.RGBA {
	switch kind {
	case damageFloaterCritical:
		return damageFloaterYellow
	case damageFloaterCombo:
		return damageFloaterYellow
	case damageFloaterIncoming:
		return damageFloaterRed
	case damageFloaterRecoveryHP:
		return color.RGBA{R: 0, G: 255, B: 0, A: 255}
	case damageFloaterRecoverySP:
		return color.RGBA{R: 0, G: 0, B: 255, A: 255}
	default:
		if fallback.A != 0 {
			return fallback
		}
		return damageFloaterWhite
	}
}

func damageFloaterProgress(floater damageFloater, now time.Time) float64 {
	duration := floater.duration
	if duration <= 0 {
		duration = floater.expires.Sub(floater.starts)
	}
	if duration <= 0 {
		return 1
	}
	return float64(now.Sub(floater.starts)) / float64(duration)
}
