package ui

import (
	"image"
	"image/color"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
)

const (
	levelUpNotificationDefaultSize = 43
	levelUpNotificationBottom      = 0
)

type LevelUpNotificationAction uint8

const LevelUpNotificationNone LevelUpNotificationAction = 0

const (
	LevelUpNotificationBase LevelUpNotificationAction = 1 << iota
	LevelUpNotificationJob
)

type LevelUpNotifications struct {
	showBase bool
	showJob  bool
	pending  LevelUpNotificationAction
	manager  *res.Manager
	loaded   bool
	offImage image.Image
	onImage  image.Image
	base     levelUpNotificationIcon
	job      levelUpNotificationIcon
}

type levelUpNotificationIcon struct {
	widget    *levelUpNotificationWidget
	root      widget.Widget
	published bool
	x         int
	y         int
	width     int
	height    int
}

type foregroundOverlayManager interface {
	AddForegroundOverlay(widget.Widget)
}

func (n *LevelUpNotifications) NotifyBase() {
	if n != nil {
		n.showBase = true
	}
}

func (n *LevelUpNotifications) NotifyJob() {
	if n != nil {
		n.showJob = true
	}
}

func (n *LevelUpNotifications) BaseVisible() bool {
	return n != nil && n.showBase
}

func (n *LevelUpNotifications) JobVisible() bool {
	return n != nil && n.showJob
}

// Rebind detaches callbacks captured by a previous WorldMode while preserving
// pending notifications across a map change.
func (n *LevelUpNotifications) Rebind(ctx Context) {
	if n == nil {
		return
	}
	n.base.unpublish(ctx)
	n.job.unpublish(ctx)
	n.base.widget = nil
	n.job.widget = nil
}

func (n *LevelUpNotifications) Update(ctx Context) LevelUpNotificationAction {
	if n == nil {
		return LevelUpNotificationNone
	}
	if n.showBase || n.showJob {
		n.loadImages(ctx.Resources)
	}
	width, height := n.imageSize()
	screenW, screenH := ctx.ScreenSize()
	y := maxInt(0, screenH-height-levelUpNotificationBottom)
	n.syncIcon(ctx, &n.job, n.showJob, 0, y, width, height, LevelUpNotificationJob)
	n.syncIcon(ctx, &n.base, n.showBase, maxInt(0, screenW-width), y, width, height, LevelUpNotificationBase)
	action := n.pending
	n.pending = LevelUpNotificationNone
	return action
}

func (n *LevelUpNotifications) syncIcon(ctx Context, icon *levelUpNotificationIcon, visible bool, x, y, width, height int, action LevelUpNotificationAction) {
	if ctx.UIManager == nil {
		return
	}
	if !visible {
		icon.unpublish(ctx)
		return
	}
	if icon.widget == nil {
		icon.widget = newLevelUpNotificationWidget(func() { n.activate(action) })
	}
	imageChanged := icon.widget.offImage != n.offImage || icon.widget.onImage != n.onImage
	icon.widget.offImage = n.offImage
	icon.widget.onImage = n.onImage
	if icon.root == nil || icon.x != x || icon.y != y || icon.width != width || icon.height != height {
		icon.unpublish(ctx)
		icon.x, icon.y = x, y
		icon.width, icon.height = width, height
		icon.root = positionedWidget(icon.widget, x, y, width, height)
	}
	if imageChanged {
		icon.widget.SetNeedsRedraw(true)
		markNeedsRedraw(icon.root)
	}
	if icon.published {
		return
	}
	if foreground, ok := ctx.UIManager.(foregroundOverlayManager); ok {
		foreground.AddForegroundOverlay(icon.root)
	} else {
		ctx.UIManager.AddOverlay(icon.root)
	}
	icon.published = true
}

func (n *LevelUpNotifications) activate(action LevelUpNotificationAction) {
	switch action {
	case LevelUpNotificationBase:
		n.showBase = false
	case LevelUpNotificationJob:
		n.showJob = false
	}
	n.pending |= action
}

func (n *LevelUpNotifications) loadImages(manager *res.Manager) {
	if manager == nil {
		return
	}
	if n.manager != manager {
		n.manager = manager
		n.loaded = false
		n.offImage = nil
		n.onImage = nil
	}
	if n.loaded {
		return
	}
	n.loaded = true
	n.offImage, _, _ = res.LoadImage(manager, levelUpNotificationImageCandidates("lv_up_off.bmp"))
	n.onImage, _, _ = res.LoadImage(manager, levelUpNotificationImageCandidates("LV_UP_ON.BMP"))
}

func levelUpNotificationImageCandidates(name string) []string {
	stem := name
	if len(name) >= 4 && strings.EqualFold(name[len(name)-4:], ".bmp") {
		stem = name[:len(name)-4]
	}
	candidates := []string{
		"skin\\default\\basic_interface\\" + name,
		"skin/default/basic_interface/" + name,
	}
	return append(candidates, res.InterfaceTextureCandidates("basic_interface\\"+stem)...)
}

func (n *LevelUpNotifications) imageSize() (int, int) {
	img := n.offImage
	if img == nil {
		img = n.onImage
	}
	if img == nil || img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		return levelUpNotificationDefaultSize, levelUpNotificationDefaultSize
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func (i *levelUpNotificationIcon) unpublish(ctx Context) {
	if i == nil || !i.published || i.root == nil || ctx.UIManager == nil {
		return
	}
	ctx.UIManager.RemoveOverlay(i.root)
	i.published = false
	i.root = nil
}

type levelUpNotificationWidget struct {
	widget.WidgetBase
	offImage   image.Image
	onImage    image.Image
	hovered    bool
	pressed    bool
	onActivate func()
}

func newLevelUpNotificationWidget(onActivate func()) *levelUpNotificationWidget {
	w := &levelUpNotificationWidget{onActivate: onActivate}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *levelUpNotificationWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(levelUpNotificationDefaultSize, levelUpNotificationDefaultSize)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *levelUpNotificationWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	img := w.offImage
	if w.pressed && w.onImage != nil {
		img = w.onImage
	}
	if img != nil {
		canvas.DrawImage(img, w.Bounds().Min)
		return
	}
	bounds := w.Bounds()
	canvas.DrawRect(bounds, Color(color.RGBA{R: 70, G: 126, B: 189, A: 255}))
	canvas.StrokeRect(bounds, Color(color.RGBA{R: 35, G: 72, B: 120, A: 255}), 1)
	canvas.DrawText("UP", bounds, 11, Color(color.RGBA{R: 255, G: 255, B: 255, A: 255}), true, widget.TextAlignCenter)
}

func (w *levelUpNotificationWidget) Event(ctx widget.Context, raw event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := raw.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		inside := w.Bounds().Contains(mouse.Position)
		w.setHovered(ctx, inside)
		if !inside {
			w.setPressed(false)
		}
	case event.MouseLeave:
		w.setHovered(ctx, false)
		w.setPressed(false)
	case event.MousePress:
		if mouse.Button == event.ButtonLeft {
			w.setPressed(true)
			return true
		}
	case event.MouseRelease:
		if mouse.Button == event.ButtonLeft {
			activate := w.pressed && w.Bounds().Contains(mouse.Position)
			w.setPressed(false)
			if activate && w.onActivate != nil {
				w.onActivate()
			}
			return true
		}
	}
	return true
}

func (w *levelUpNotificationWidget) setHovered(ctx widget.Context, hovered bool) {
	if hovered {
		ctx.SetCursor(widget.CursorPointer)
	} else {
		ctx.SetCursor(widget.CursorDefault)
	}
	if w.hovered == hovered {
		return
	}
	w.hovered = hovered
	w.SetNeedsRedraw(true)
}

func (w *levelUpNotificationWidget) setPressed(pressed bool) {
	if w.pressed == pressed {
		return
	}
	w.pressed = pressed
	w.SetNeedsRedraw(true)
}
