package ui

import (
	"image/color"
	"strings"
	"time"

	"github.com/kivutar/goro/render"
)

const (
	poptipMaxItems = 2
	poptipSolid    = 3 * time.Second
	poptipFade     = time.Second
	poptipBaseY    = 90
	poptipPadding  = 4
)

type poptipItem struct {
	text    string
	shownAt time.Time
}

// Poptips is the short, server-driven message stack shown near the top of the
// game view. It is deliberately input-transparent.
type Poptips struct {
	items []poptipItem
}

func (p *Poptips) Clear() {
	if p != nil {
		p.items = nil
	}
}

func (p *Poptips) Show(text string, now time.Time) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	items := p.items[:0]
	for _, item := range p.items {
		if item.text != text && now.Sub(item.shownAt) < poptipSolid+poptipFade {
			items = append(items, item)
		}
	}
	p.items = append([]poptipItem{{text: text, shownAt: now}}, items...)
	if len(p.items) > poptipMaxItems {
		p.items = p.items[:poptipMaxItems]
	}
}

func (p *Poptips) Draw(screen *render.Frame, now time.Time) {
	if screen == nil || p == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	lineHeight := 20
	for index, item := range p.items {
		alpha := poptipAlpha(now.Sub(item.shownAt))
		if alpha == 0 {
			continue
		}
		textW, textH := render.BitmapTextSize(item.text)
		boxW := textW + 2*poptipPadding
		boxH := textH + 2*poptipPadding
		x := (screen.Bounds().Dx() - boxW) / 2
		y := poptipBaseY - index*lineHeight - boxH/2
		backgroundAlpha := uint8(float64(204) * alpha)
		textAlpha := uint8(float64(255) * alpha)
		render.DrawRect(screen, float64(x), float64(y), float64(boxW), float64(boxH), color.RGBA{A: backgroundAlpha})
		render.DrawCenteredUITextAt(screen, item.text, float64(screen.Bounds().Dx())/2, float64(y+poptipPadding-2), color.RGBA{R: 255, G: 255, B: 255, A: textAlpha})
	}
}

func poptipAlpha(age time.Duration) float64 {
	if age < 0 || age >= poptipSolid+poptipFade {
		return 0
	}
	if age <= poptipSolid {
		return 1
	}
	return 1 - float64(age-poptipSolid)/float64(poptipFade)
}
