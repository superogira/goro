package ui

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

const (
	itemPickupNotificationTopY     = 72
	itemPickupNotificationH        = 36
	itemPickupNotificationIconSize = 24
	itemPickupNotificationIconGap  = 5
	itemPickupNotificationMargin   = 8
	itemPickupNotificationOuter    = 3
	itemPickupNotificationBorderW  = 1
	itemPickupNotificationPadX     = 5
	itemPickupNotificationPadRight = 14
	itemPickupNotificationTextH    = 20
	itemPickupNotificationRadius   = 4
	itemPickupNotificationLife     = 5 * time.Second
)

var (
	itemPickupNotificationOutline = color.RGBA{R: 255, G: 255, B: 255, A: 245}
	itemPickupNotificationBorder  = color.RGBA{R: 74, G: 138, B: 202, A: 245}
	itemPickupNotificationFill    = color.RGBA{R: 255, G: 255, B: 255, A: 245}
	itemPickupNotificationText    = color.RGBA{R: 30, G: 34, B: 40, A: 255}
)

type ItemPickupNotification struct {
	item    session.InventoryItem
	text    string
	shownAt time.Time
}

func (n *ItemPickupNotification) Show(ctx Context, item session.InventoryItem, count int, now time.Time) {
	if count <= 0 {
		count = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	n.item = item
	n.text = itemPickupNotificationTextFor(ctx.Resources, item, count)
	n.shownAt = now
	if pickupWebEnabled() {
		pickupWebShow(n.text, pickupWebIcon(ctx.Resources, item))
	}
}

func (n *ItemPickupNotification) Draw(screen *render.Frame, ctx Context, assets AssetProvider, now time.Time) {
	if screen == nil || n == nil || assets == nil || !n.visible(now) {
		return
	}
	if pickupWebEnabled() {
		return
	}
	screenW := screen.Bounds().Dx()
	if screenW <= itemPickupNotificationMargin*2 {
		return
	}
	text := fitItemPickupNotificationText(n.text, screenW)
	if text == "" {
		return
	}
	x, y, w, h := itemPickupNotificationBounds(screenW, text)
	drawItemPickupNotificationSurface(screen, x, y, w, h)
	contentInset := itemPickupNotificationOuter + itemPickupNotificationBorderW
	iconX := x + contentInset + itemPickupNotificationPadX
	iconY := y + (h-itemPickupNotificationIconSize)/2
	assets.DrawInventoryItemIcon(screen, ctx.Resources, n.item, iconX, iconY)
	textX := iconX + itemPickupNotificationIconSize + itemPickupNotificationIconGap
	textY := y + (h-itemPickupNotificationTextH)/2 - 2
	render.DrawUITextAt(screen, text, float64(textX), float64(textY), itemPickupNotificationText)
}

func (n *ItemPickupNotification) visible(now time.Time) bool {
	if n == nil || n.shownAt.IsZero() || n.text == "" {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	return now.Sub(n.shownAt) <= itemPickupNotificationLife
}

func itemPickupNotificationTextFor(manager *res.Manager, item session.InventoryItem, count int) string {
	if count <= 0 {
		count = 1
	}
	name := fmt.Sprintf("item %d", item.ItemID)
	if manager != nil {
		if resolved, ok := manager.ItemDisplayName(int(item.ItemID), item.Identified); ok && strings.TrimSpace(resolved) != "" {
			name = strings.TrimSpace(resolved)
		}
	}
	suffix := "- %d obtained."
	if manager != nil {
		if msg, ok := manager.MsgString(696); ok && strings.TrimSpace(msg) != "" {
			suffix = strings.TrimSpace(msg)
		}
	}
	suffix = strings.Replace(suffix, "%d", strconv.Itoa(count), 1)
	return strings.TrimSpace(name + " " + suffix)
}

func fitItemPickupNotificationText(text string, screenW int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	maxTextW := itemPickupNotificationMaxTextWidth(screenW)
	if maxTextW <= 0 {
		return ""
	}
	if w, _ := render.BitmapTextSize(text); w <= maxTextW {
		return text
	}
	runes := []rune(text)
	for len(runes) > 0 {
		candidate := string(runes) + "..."
		if w, _ := render.BitmapTextSize(candidate); w <= maxTextW {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return ""
}

func itemPickupNotificationBounds(screenW int, text string) (int, int, int, int) {
	textW, _ := render.BitmapTextSize(text)
	contentInset := itemPickupNotificationOuter + itemPickupNotificationBorderW
	w := contentInset*2 + itemPickupNotificationPadX*2 + itemPickupNotificationIconSize + itemPickupNotificationIconGap + textW
	w += itemPickupNotificationPadRight - itemPickupNotificationPadX
	maxW := screenW - itemPickupNotificationMargin*2
	if w > maxW {
		w = maxW
	}
	h := itemPickupNotificationH
	x := (screenW - w) / 2
	if x < itemPickupNotificationMargin {
		x = itemPickupNotificationMargin
	}
	return x, itemPickupNotificationTopY, w, h
}

func itemPickupNotificationMaxTextWidth(screenW int) int {
	contentInset := itemPickupNotificationOuter + itemPickupNotificationBorderW
	return screenW - itemPickupNotificationMargin*2 -
		contentInset*2 -
		itemPickupNotificationPadX -
		itemPickupNotificationPadRight -
		itemPickupNotificationIconSize -
		itemPickupNotificationIconGap
}

var itemPickupNotificationSurfaceCache = map[int]*render.Image{}

func drawItemPickupNotificationSurface(screen *render.Frame, x, y, w, h int) {
	if screen == nil || w <= 0 || h <= 0 {
		return
	}
	surface := cachedItemPickupNotificationSurface(w, h)
	if surface == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = render.FilterLinear
	screen.DrawImage(surface, &opts)
}

func cachedItemPickupNotificationSurface(w, h int) *render.Image {
	if w <= 0 || h <= 0 {
		return nil
	}
	if img := itemPickupNotificationSurfaceCache[w]; img != nil && img.Bounds().Dy() == h {
		return img
	}
	img := render.NewImage(w, h)
	DrawRoundedSurface(img, 0, 0, w, h, itemPickupNotificationOutline, color.RGBA{}, itemPickupNotificationRadius)
	blueInset := itemPickupNotificationOuter
	DrawRoundedSurface(
		img,
		blueInset,
		blueInset,
		w-blueInset*2,
		h-blueInset*2,
		itemPickupNotificationBorder,
		color.RGBA{},
		float32(maxInt(0, itemPickupNotificationRadius-blueInset)),
	)
	fillInset := itemPickupNotificationOuter + itemPickupNotificationBorderW
	DrawRoundedSurface(
		img,
		fillInset,
		fillInset,
		w-fillInset*2,
		h-fillInset*2,
		itemPickupNotificationFill,
		color.RGBA{},
		float32(maxInt(0, itemPickupNotificationRadius-fillInset)),
	)
	if len(itemPickupNotificationSurfaceCache) > 128 {
		itemPickupNotificationSurfaceCache = map[int]*render.Image{}
	}
	itemPickupNotificationSurfaceCache[w] = img
	return img
}
