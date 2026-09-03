package ui

import (
	"image"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

// NPCCutinOverlay owns the single illustration displayed by an NPC script. The
// position byte selects its placement; receiving another image replaces it.
// Static cut-ins draw below the UI, while movable variants reuse Window.
type NPCCutinOverlay struct {
	texture  *render.Image
	position uint8
	window   Window
}

// Apply first removes the previous illustration, matching the original client.
// A nil texture leaves it cleared, preventing stale artwork after a load error.
func (c *NPCCutinOverlay) Apply(cutin network.NPCCutin, texture *render.Image) {
	if c == nil {
		return
	}
	c.Clear()
	if cutin.Position == network.NPCCutinClear || cutin.Position > network.NPCCutinWindowless || cutin.Image == "" || texture == nil {
		return
	}
	c.texture = texture
	c.position = cutin.Position
}

func (c *NPCCutinOverlay) Clear() {
	if c == nil {
		return
	}
	if c.window.IsOpen() || c.window.published != nil {
		c.window.Close()
	}
	c.window = Window{}
	c.texture = nil
	c.position = network.NPCCutinLeft
}

func (c *NPCCutinOverlay) Visible() bool {
	return c != nil && c.texture != nil && !c.texture.Bounds().Empty()
}

func (c *NPCCutinOverlay) Draw(screen *render.Frame) {
	if c == nil || screen == nil || npcCutinIsMovable(c.position) {
		return
	}
	bounds, ok := npcCutinBounds(screen.Bounds(), c.position, c.texture)
	if !ok {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	// The image remains at its native logical size. Linear sampling avoids
	// uneven pixels when the final framebuffer uses fractional scaling.
	opts.Filter = render.FilterLinear
	screen.DrawImage(c.texture, &opts)
}

// Update owns the two windowed cut-in variants. The ordinary left, center and
// right illustrations remain lightweight render overlays and need no update.
func (c *NPCCutinOverlay) Update(ctx Context) bool {
	if c == nil || !c.Visible() || !npcCutinIsMovable(c.position) {
		return false
	}
	if !c.window.IsOpen() {
		screenWidth, screenHeight := ctx.ScreenSize()
		bounds, ok := npcCutinBounds(image.Rect(0, 0, screenWidth, screenHeight), c.position, c.texture)
		if !ok {
			return false
		}
		c.openWindow(ctx, bounds)
	}
	return c.window.Update(ctx)
}

func (c *NPCCutinOverlay) openWindow(ctx Context, bounds image.Rectangle) {
	c.window = NewWindow(bounds.Dx(), bounds.Dy())
	c.window.CloseOnEsc = false
	texture := c.texture.RGBA()
	var content widget.Widget
	if c.position == network.NPCCutinWindow {
		content = Win(
			Title(""),
			Content(newStaticImageWidget(texture, texture.Bounds().Dx(), texture.Bounds().Dy())),
			Size(float32(bounds.Dx()), float32(bounds.Dy())),
		)
	} else {
		content = newNPCWindowlessCutinWidget(texture, c.Clear)
	}
	c.window.OpenAt(bounds.Min.X, bounds.Min.Y, content)
	c.window.Publish(ctx)
}

func (c *NPCCutinOverlay) PointerBlocked(screenWidth, screenHeight, x, y int) bool {
	if c == nil || screenWidth <= 0 || screenHeight <= 0 {
		return false
	}
	if npcCutinIsMovable(c.position) && c.window.IsOpen() {
		return image.Pt(x, y).In(image.Rect(c.window.x, c.window.y, c.window.x+c.window.width, c.window.y+c.window.height))
	}
	bounds, ok := npcCutinBounds(image.Rect(0, 0, screenWidth, screenHeight), c.position, c.texture)
	return ok && image.Pt(x, y).In(bounds)
}

func npcCutinBounds(screen image.Rectangle, position uint8, texture *render.Image) (image.Rectangle, bool) {
	if texture == nil || texture.Bounds().Empty() || position > network.NPCCutinWindowless || screen.Empty() {
		return image.Rectangle{}, false
	}
	width := texture.Bounds().Dx()
	height := texture.Bounds().Dy()
	x := screen.Min.X
	y := screen.Max.Y - height
	switch position {
	case network.NPCCutinCenter:
		x += (screen.Dx() - width) / 2
	case network.NPCCutinRight:
		x = screen.Max.X - width
	case network.NPCCutinWindow:
		height += ROWindowTitleHeight
		x += (screen.Dx() - width) / 2
		y = screen.Min.Y + (screen.Dy()-height)/2
	case network.NPCCutinWindowless:
		x += (screen.Dx() - width) / 2
		y = screen.Min.Y + (screen.Dy()-height)/2
	}
	if x < screen.Min.X {
		x = screen.Min.X
	}
	if y < screen.Min.Y {
		y = screen.Min.Y
	}
	return image.Rect(x, y, x+width, y+height), true
}

func npcCutinIsMovable(position uint8) bool {
	return position == network.NPCCutinWindow || position == network.NPCCutinWindowless
}

type npcWindowlessCutinWidget struct {
	widget.WidgetBase
	image        image.Image
	onClose      func()
	closeHovered bool
}

func newNPCWindowlessCutinWidget(img image.Image, onClose func()) *npcWindowlessCutinWidget {
	w := &npcWindowlessCutinWidget{image: img, onClose: onClose}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *npcWindowlessCutinWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := geometry.Size{}
	if w.image != nil {
		bounds := w.image.Bounds()
		size = geometry.Sz(float32(bounds.Dx()), float32(bounds.Dy()))
	}
	size = constraints.Constrain(size)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *npcWindowlessCutinWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.image == nil {
		return
	}
	canvas.DrawImage(w.image, w.Bounds().Min)
	rotheme.DrawIconButton(canvas, npcWindowlessCutinCloseBounds(w.Bounds()), rotheme.IconButtonClose, w.closeHovered, false)
}

func (w *npcWindowlessCutinWidget) Event(ctx widget.Context, raw event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := raw.(*event.MouseEvent)
	if !ok {
		return false
	}
	hovered := npcWindowlessCutinCloseBounds(w.Bounds()).Contains(mouse.Position)
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		w.setCloseHovered(ctx, hovered)
		return hovered
	case event.MouseLeave:
		w.setCloseHovered(ctx, false)
	case event.MousePress:
		if mouse.Button == event.ButtonLeft && hovered {
			if w.onClose != nil {
				w.onClose()
			}
			return true
		}
	}
	return false
}

func (w *npcWindowlessCutinWidget) setCloseHovered(ctx widget.Context, hovered bool) {
	if hovered {
		ctx.SetCursor(widget.CursorPointer)
	} else {
		ctx.SetCursor(widget.CursorDefault)
	}
	if w.closeHovered == hovered {
		return
	}
	w.closeHovered = hovered
	w.SetNeedsRedraw(true)
}

func npcWindowlessCutinCloseBounds(bounds geometry.Rect) geometry.Rect {
	size := float32(windowTitleButtonSize)
	x := bounds.Max.X - float32(windowTitleButtonPadR+windowTitleButtonSize)
	y := bounds.Min.Y + float32((ROWindowTitleHeight-windowTitleButtonSize)/2)
	return geometry.NewRect(x, y, size, size)
}
