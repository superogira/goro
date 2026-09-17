package ui

import (
	"fmt"
	"image"
	"math"
	"slices"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui/rotheme"
	xdraw "golang.org/x/image/draw"
)

// WorldMapWindow is the legacy illustrated atlas, not a navigation service.
// Artwork and previews are cached; the closed window does no presentation work.
type WorldMapWindow struct {
	Window
	art       image.Image
	scaled    image.Image
	board     *worldMapWidget
	previews  map[string]image.Image
	showNames bool
	memberBuf []worldMapMember
	dotBuf    []worldMapDot
}

type worldMapMember struct {
	ID      uint32
	Name    string
	MapName string
}

type worldMapDot struct {
	ID   uint32
	X, Y int
}

func (w *WorldMapWindow) Toggle(ctx Context) error {
	if w.IsOpen() {
		w.Close()
		return nil
	}
	return w.OpenWindow(ctx)
}

func (w *WorldMapWindow) OpenWindow(ctx Context) error {
	if ctx.Resources == nil || len(ctx.Resources.WorldMapEntries()) == 0 {
		return fmt.Errorf("mapPosTable.txt is missing or empty")
	}
	if w.art == nil {
		var err error
		w.art, _, err = res.LoadImage(ctx.Resources, []string{
			"data/texture/유저인터페이스/worldmap.bmp", "data/texture/interface/worldmap.bmp", "worldmap.bmp",
		})
		if err != nil {
			return fmt.Errorf("worldmap.bmp could not be loaded")
		}
	}
	screenW, screenH := ctx.ScreenSize()
	size := worldMapSize(w.art.Bounds().Size(), screenW, screenH)
	w.EnsureWindow(size.X, size.Y+ROWindowTitleHeight)
	w.SetSize(size.X, size.Y+ROWindowTitleHeight)
	w.setTitleButtonCount(2)
	if w.scaled == nil || w.scaled.Bounds().Size() != size {
		w.scaled = scaleWorldMapImage(w.art, size)
	}
	if w.previews == nil {
		w.previews = make(map[string]image.Image)
	}
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.UpdatePresentation(ctx)
	w.Publish(ctx)
	return nil
}

func (w *WorldMapWindow) Rebind(ctx Context) {
	if w.IsOpen() {
		// Replace callbacks and the presentation snapshot copied across maps.
		w.RebindContent(ctx, w.widgetTree(ctx))
		w.UpdatePresentation(ctx)
	}
}

func (w *WorldMapWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *WorldMapWindow) widgetTree(ctx Context) widget.Widget {
	w.board = &worldMapWidget{
		resources: ctx.Resources, entries: ctx.Resources.WorldMapEntries(),
		image: w.scaled, sourceSize: w.art.Bounds().Size(), previews: w.previews,
		hovered: -1, showNames: w.showNames,
	}
	w.board.SetVisible(true)
	w.board.SetEnabled(true)
	w.board.SetRepaintBoundary(true)
	return Win(Title("World Map"), CloseButton(true), OnClose(w.Close),
		TitleButton(rotheme.IconButtonSearch, func() {
			w.showNames = !w.showNames
			w.board.showNames = w.showNames
			w.redraw(ctx)
		}),
		Size(float32(w.width), float32(w.height)), Content(w.board))
}

// Capture only data the atlas uses, never live session/world pointers in Draw.
// Walking only invalidates the atlas while its current-map preview is visible.
func (w *WorldMapWindow) UpdatePresentation(ctx Context) {
	if !w.IsOpen() || w.board == nil {
		return
	}
	b := w.board
	current := ""
	if ctx.World != nil {
		current = normalizeMinimapMapName(ctx.World.MapName)
	}
	w.memberBuf = w.memberBuf[:0]
	w.dotBuf = w.dotBuf[:0]
	mapW, mapH := 0, 0
	showPosition := b.hovered >= 0 && b.entries[b.hovered].MapName == current
	if showPosition && ctx.World != nil {
		mapW, mapH = minimapWorldSize(ctx.World)
		w.dotBuf = append(w.dotBuf, worldMapDot{X: ctx.World.Player.X, Y: ctx.World.Player.Y})
	}
	if ctx.Session != nil {
		for _, member := range ctx.Session.Party.Members {
			if member.AccountID == 0 || member.AccountID == ctx.Session.AccountID || !member.Online() {
				continue
			}
			name := normalizeMinimapMapName(member.MapName)
			w.memberBuf = append(w.memberBuf, worldMapMember{ID: member.AccountID, Name: member.Name, MapName: name})
			if showPosition && name == current && member.X >= 0 && member.Y >= 0 {
				w.dotBuf = append(w.dotBuf, worldMapDot{ID: member.AccountID, X: member.X, Y: member.Y})
			}
		}
	}
	if b.currentMap == current && b.mapW == mapW && b.mapH == mapH && slices.Equal(b.members, w.memberBuf) && slices.Equal(b.dots, w.dotBuf) {
		return
	}
	b.currentMap, b.mapW, b.mapH = current, mapW, mapH
	b.members = slices.Clone(w.memberBuf)
	b.dots = slices.Clone(w.dotBuf)
	w.redraw(ctx)
}

func (w *WorldMapWindow) redraw(ctx Context) {
	w.board.SetNeedsRedraw(true)
	invalidateWindowRect(ctx, windowFrameRect(w.x, w.y, w.width, w.height))
}

func worldMapSize(source image.Point, screenW, screenH int) image.Point {
	scale := min(1., float64(max(1, screenW-2*windowScreenMargin))/float64(source.X),
		float64(max(1, screenH-2*windowScreenMargin-ROWindowTitleHeight))/float64(source.Y))
	return image.Pt(max(1, int(float64(source.X)*scale)), max(1, int(float64(source.Y)*scale)))
}

func scaleWorldMapImage(img image.Image, size image.Point) image.Image {
	dst := image.NewRGBA(image.Rectangle{Max: size})
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)
	return dst
}

type worldMapWidget struct {
	widget.WidgetBase
	resources  *res.Manager
	entries    []res.WorldMapEntry
	image      image.Image
	sourceSize image.Point
	previews   map[string]image.Image
	hovered    int
	showNames  bool
	currentMap string
	members    []worldMapMember
	dots       []worldMapDot
	mapW, mapH int
}

func (w *worldMapWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := w.image.Bounds().Size()
	result := constraints.Constrain(geometry.Sz(float32(size.X), float32(size.Y)))
	w.SetBounds(geometry.FromPointSize(w.Position(), result))
	return result
}

func (w *worldMapWidget) mapRect(entry res.WorldMapEntry) geometry.Rect {
	b := w.Bounds()
	sx, sy := b.Width()/float32(w.sourceSize.X), b.Height()/float32(w.sourceSize.Y)
	r := entry.Bounds
	return geometry.NewRect(b.Min.X+float32(r.Min.X)*sx, b.Min.Y+float32(r.Min.Y)*sy, float32(r.Dx())*sx, float32(r.Dy())*sy)
}

func (w *worldMapWidget) mapAt(point geometry.Point) int {
	for i, entry := range w.entries {
		if w.mapRect(entry).Contains(point) {
			return i
		}
	}
	return -1
}

func (w *worldMapWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	index := w.hovered
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MousePress:
		// Hover-enter is delivered directly, whereas moves pass through parent
		// transforms. GlobalPosition gives both paths the same coordinates.
		point := mouse.GlobalPosition.Sub(w.ScreenBounds().Min).Add(w.Bounds().Min)
		index = w.mapAt(point)
	case event.MouseLeave:
		index = -1
	}
	if index != w.hovered {
		w.hovered = index
		if index >= 0 {
			w.loadPreview(w.entries[index].MapName)
		}
		w.SetNeedsRedraw(true)
		ctx.InvalidateRect(w.Bounds())
	}
	// This is a map browser: clicking must never issue a movement command.
	return true
}

func (w *worldMapWidget) loadPreview(name string) {
	if _, cached := w.previews[name]; cached {
		return
	}
	var preview image.Image
	if w.resources != nil {
		if img, _, err := res.LoadImage(w.resources, minimapImageCandidates(name)); err == nil {
			size := img.Bounds().Size()
			scale := 128. / float64(max(size.X, size.Y))
			preview = scaleWorldMapImage(img, image.Pt(max(1, int(float64(size.X)*scale)), max(1, int(float64(size.Y)*scale))))
		}
	}
	w.previews[name] = preview // Missing minimaps are normal, and cached too.
}

func (w *worldMapWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	canvas.PushClip(bounds)
	defer canvas.PopClip()
	canvas.DrawImage(w.image, bounds.Min)
	for i, entry := range w.entries {
		r := w.mapRect(entry)
		if i == w.hovered {
			canvas.DrawRect(r, widget.Color{R: 1, G: 1, B: 1, A: .2})
			canvas.StrokeRect(r, widget.ColorWhite, 1)
		}
		for _, member := range w.members {
			if member.MapName == entry.MapName {
				canvas.StrokeRect(r, Color(minimapMemberColor(member.ID)), 2)
				break
			}
		}
		if w.showNames {
			size := float32(9)
			width := rotheme.MeasureText(canvas, entry.MapName, size, false)
			if width > r.Width() && width > 0 {
				size *= r.Width() / width
			}
			label := geometry.NewRect(r.Min.X, r.Min.Y+(r.Height()-14)/2, r.Width(), 14)
			canvas.DrawRect(label, widget.Color{R: 1, G: 1, B: 1, A: .85})
			rotheme.DrawText(canvas, entry.MapName, label, size, rotheme.Default.Colors.Text, false, widget.TextAlignCenter)
		}
		if entry.MapName == w.currentMap {
			drawWorldMapStar(canvas, geometry.Pt(r.Min.X+r.Width()/2, r.Min.Y+r.Height()/2))
		}
	}
	if w.hovered >= 0 {
		w.drawPreview(canvas, w.entries[w.hovered])
	}
}

func drawWorldMapStar(canvas widget.Canvas, center geometry.Point) {
	// A small geometric star stays legible without depending on font coverage.
	var points [5]geometry.Point
	for i := range points {
		a := float64(i)*4*math.Pi/5 - math.Pi/2
		points[i] = geometry.Pt(center.X+float32(math.Cos(a))*7, center.Y+float32(math.Sin(a))*7)
	}
	for i, point := range points {
		canvas.DrawLine(point, points[(i+1)%len(points)], Color(minimapPlayerColor), 2)
	}
}

func (w *worldMapWidget) drawPreview(canvas widget.Canvas, entry res.WorldMapEntry) {
	b := w.Bounds()
	width := min(float32(190), b.Width()-8)
	rows := 0
	for _, member := range w.members {
		if member.MapName == entry.MapName {
			rows++
		}
	}
	rows = min(rows, max(0, int((b.Height()-164)/16)))
	height := min(float32(156+rows*16), b.Height()-8)
	panel := geometry.NewRect(b.Max.X-width-4, b.Min.Y+4, width, height)
	canvas.DrawRoundRect(panel, widget.Color{R: 1, G: 1, B: 1, A: .95}, 4)
	canvas.StrokeRoundRect(panel, rotheme.Default.Colors.WindowBorder, 4, 1)
	label := geometry.NewRect(panel.Min.X+6, panel.Min.Y+3, width-12, 18)
	rotheme.DrawLabel(canvas, entry.MapName, label, widget.TextAlignCenter)
	img := w.previews[entry.MapName]
	if img == nil {
		rotheme.DrawText(canvas, "No minimap available", geometry.NewRect(panel.Min.X+6, panel.Min.Y+22, width-12, 128),
			rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
	} else {
		size := img.Bounds().Size()
		at := geometry.Pt(panel.Min.X+(width-float32(size.X))/2, panel.Min.Y+22)
		canvas.DrawImage(img, at)
		if entry.MapName == w.currentMap {
			rect := minimapRect{x: int(at.X), y: int(at.Y), w: size.X, h: size.Y}
			for _, dot := range w.dots {
				if x, y, ok := minimapCellToScreen(rect, w.mapW, w.mapH, dot.X, dot.Y); ok {
					fill := minimapPlayerColor
					if dot.ID != 0 {
						fill = minimapMemberColor(dot.ID)
					}
					drawMinimapSquareCentered(canvas, x, y, 4, fill)
				}
			}
		}
	}
	y := panel.Min.Y + 153
	for _, member := range w.members {
		if member.MapName != entry.MapName || rows == 0 {
			continue
		}
		rotheme.DrawText(canvas, trimRunes(member.Name, 24), geometry.NewRect(panel.Min.X+6, y, width-12, 16),
			rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
		y += 16
		rows--
	}
}
