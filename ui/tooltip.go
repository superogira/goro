package ui

import (
	"strings"

	"github.com/kivutar/goro/render"
)

type tooltipState struct {
	text       string
	centerX    int
	belowY     int
	aboveY     int
	maxWidth   float64
	maxLines   int
	colorCodes bool
	open       bool
}

// TooltipsSuppressed reports whether transient hover UI should stay hidden.
// Window dragging uses a renderer-owned fast path, so that drag state is the
// authoritative signal shared by every window and tooltip.
func TooltipsSuppressed(ctx Context) bool {
	return windowDragActive(ctx)
}

func (t *tooltipState) Show(ctx Context, text string, centerX, belowY, aboveY int) {
	t.ShowBox(ctx, text, centerX, belowY, aboveY, 0, 1)
}

func (t *tooltipState) ShowBox(ctx Context, text string, centerX, belowY, aboveY int, maxWidth float64, maxLines int) {
	if TooltipsSuppressed(ctx) {
		t.Hide()
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		t.Hide()
		return
	}
	t.text = text
	t.centerX = centerX
	t.belowY = belowY
	t.aboveY = aboveY
	t.maxWidth = maxWidth
	t.maxLines = maxLines
	t.colorCodes = false
	t.open = true
}

func (t *tooltipState) ShowColoredBox(ctx Context, text string, centerX, belowY int, maxWidth float64, maxLines int) {
	t.ShowBox(ctx, text, centerX, belowY, 0, maxWidth, maxLines)
	t.colorCodes = true
}

func (t *tooltipState) Hide() {
	if t == nil {
		return
	}
	t.text = ""
	t.open = false
}

func (t *tooltipState) Draw(ctx Context, screen *render.Frame) {
	if t == nil || !t.open || TooltipsSuppressed(ctx) {
		return
	}
	render.DrawUITooltipBox(screen, t.text, float64(t.centerX), float64(t.belowY), float64(t.aboveY), t.maxWidth, t.maxLines, t.colorCodes)
}

func (t *tooltipState) Open() bool {
	return t != nil && t.open
}

func (t *tooltipState) Text() string {
	if t == nil {
		return ""
	}
	return t.text
}
