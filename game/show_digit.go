package game

import (
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

const (
	timeFontSPR        = "data\\sprite\\이팩트\\timefont.spr"
	timeFontACT        = "data\\sprite\\이팩트\\timefont.act"
	showDigitTop       = 40.0
	showDigitStep      = 40.0
	showDigitStatic    = 5 * time.Second
	showDigitFastTick  = 500 * time.Millisecond
	showDigitSeparator = 10
)

type showDigitState struct {
	mode    network.ShowDigitMode
	start   uint32
	shownAt time.Time
}

func newShowDigitState(show network.ShowDigit, now time.Time) showDigitState {
	if now.IsZero() {
		now = time.Now()
	}
	value := int64(show.Value)
	if value < 0 {
		value = -value
	}
	if show.Mode == network.ShowDigitFastCountDown && value > 99 {
		value = 99
	}
	return showDigitState{mode: show.Mode, start: uint32(value), shownAt: now}
}

func (s showDigitState) value(now time.Time) (uint32, bool) {
	if s.shownAt.IsZero() {
		return 0, false
	}
	elapsed := now.Sub(s.shownAt)
	if elapsed < 0 {
		elapsed = 0
	}
	switch s.mode {
	case network.ShowDigitStatic:
		return s.start, elapsed < showDigitStatic
	case network.ShowDigitCountUp:
		return s.start + uint32(elapsed/time.Second), true
	case network.ShowDigitCountDown:
		return s.start - uint32(elapsed/time.Second), true
	case network.ShowDigitFastCountDown:
		steps := uint32(elapsed / showDigitFastTick)
		if steps >= s.start {
			return 0, false
		}
		return s.start - steps, true
	default:
		return 0, false
	}
}

func (s showDigitState) actions(now time.Time) ([]int, bool) {
	value, visible := s.value(now)
	if !visible {
		return nil, false
	}
	if s.mode == network.ShowDigitFastCountDown {
		return []int{int(value / 10), int(value % 10)}, true
	}
	groups := showDigitTimeGroups(value)
	actions := make([]int, 0, len(groups)*3-1)
	for index := len(groups) - 1; index >= 0; index-- {
		if len(actions) > 0 {
			actions = append(actions, showDigitSeparator)
		}
		actions = append(actions, int(groups[index]/10), int(groups[index]%10))
	}
	return actions, true
}

func showDigitTimeGroups(seconds uint32) []uint32 {
	groups := []uint32{seconds % 60}
	minutes := seconds / 60
	if minutes == 0 {
		return groups
	}
	groups = append(groups, minutes%60)
	hours := minutes / 60
	if hours == 0 {
		return groups
	}
	groups = append(groups, hours%24)
	days := hours / 24
	if days > 0 {
		groups = append(groups, days%100)
	}
	return groups
}

func (m *WorldMode) drawShowDigit(screen *render.Frame, ctx client.Context, now time.Time) {
	actions, visible := m.showDigit.actions(now)
	if !visible {
		m.showDigit = showDigitState{}
		return
	}
	view := m.timeFontSprite(ctx)
	if view == nil {
		render.DrawCenteredUITextAtSize(screen, showDigitFallbackText(actions), float64(screen.Bounds().Dx())/2, showDigitTop, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 28, false)
		return
	}
	startX := float64(screen.Bounds().Dx())/2 - float64(len(actions))*showDigitStep/2
	for index, actionIndex := range actions {
		billboard, ok := spriteBillboardForAction(view, actionIndex)
		if !ok {
			continue
		}
		var opts render.DrawImageOptions
		opts.GeoM.Translate(startX+float64(index)*showDigitStep-billboard.anchorX, showDigitTop-billboard.anchorY)
		opts.Filter = render.FilterNearest
		screen.DrawImage(billboard.image, &opts)
	}
}

func (m *WorldMode) timeFontSprite(ctx client.Context) *spriteView {
	if m.timeFontView != nil || m.timeFontMiss || ctx.Resources == nil {
		return m.timeFontView
	}
	view, status := loadSpriteView(ctx.Resources,
		[]string{timeFontACT, strings.ReplaceAll(timeFontACT, "\\", "/")},
		[]string{timeFontSPR, strings.ReplaceAll(timeFontSPR, "\\", "/")},
		nil,
		"time font",
	)
	if view == nil {
		m.timeFontMiss = true
		glog.Warnf("time font sprite unavailable: %s", status)
		return nil
	}
	m.timeFontView = view
	glog.Debugf("time font sprite resources %s", status)
	return view
}

// serverDigitSpritePrefetchGroups lists the timefont candidates for
// background warming at map enter: server-driven digit displays (WoE
// countdowns and friends) can start at any moment, and the first frame
// would otherwise fetch the sprite inline.
func serverDigitSpritePrefetchGroups() [][]string {
	return [][]string{
		{timeFontACT, strings.ReplaceAll(timeFontACT, "\\", "/")},
		{timeFontSPR, strings.ReplaceAll(timeFontSPR, "\\", "/")},
	}
}

func spriteBillboardForAction(view *spriteView, actionIndex int) (*spriteBillboard, bool) {
	if view == nil || view.act == nil || actionIndex < 0 || actionIndex >= len(view.act.Actions) {
		return nil, false
	}
	action := view.act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return nil, false
	}
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: 0}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeSingleSpriteBillboard(view, action.Animations[0])
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
	return billboard, true
}

func showDigitFallbackText(actions []int) string {
	var text strings.Builder
	for _, action := range actions {
		if action == showDigitSeparator {
			text.WriteByte(':')
		} else {
			text.WriteByte(byte('0' + action))
		}
	}
	return text.String()
}
