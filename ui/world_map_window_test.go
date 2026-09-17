package ui

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func worldMapTestContext(t *testing.T) Context {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mapPosTable.txt"), []byte("0#prontera.rsw#10#20#70#80#\n1#geffen.rsw#80#30#130#80#"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"worldmap.bmp", "prontera.bmp"} {
		f, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, image.NewRGBA(image.Rect(0, 0, 200, 160)))
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	return Context{Resources: &res.Manager{Root: root}, Session: &session.Session{AccountID: 1},
		Input: input.NewState(), UIManager: NewManager(), ScreenW: 1024, ScreenH: 768,
		World: &world.World{MapName: "prontera.rsw", Player: world.Actor{X: 50, Y: 60}, GAT: &res.GAT{Width: 200, Height: 200}}}
}

func TestWorldMapIdleAndPartySnapshots(t *testing.T) {
	ctx := worldMapTestContext(t)
	ctx.Session.Party.Members = []session.PartyMember{
		{AccountID: 1, MapName: "prontera.rsw"},
		{AccountID: 2, Name: "Here", MapName: "prontera.rsw", X: 80, Y: 90},
		{AccountID: 3, Name: "Away", MapName: "geffen.rsw", X: 80, Y: 90},
		{AccountID: 4, Name: "Offline", MapName: "prontera.rsw", State: 1},
	}
	w := &WorldMapWindow{}
	if err := w.OpenWindow(ctx); err != nil {
		t.Fatal(err)
	}
	board, content, scaled := w.board, w.content, w.scaled
	if board.currentMap != "prontera" || len(board.members) != 2 {
		t.Fatalf("map = %s, members = %+v", board.currentMap, board.members)
	}
	board.ClearRedraw()
	ctx.World.Player.X++
	ctx.Session.Party.Members[1].HP = 50
	w.UpdatePresentation(ctx)
	if board.NeedsRedraw() || w.content != content || w.scaled != scaled {
		t.Fatal("walking/HP rebuilt an unchanged atlas")
	}
	if allocs := testing.AllocsPerRun(10, func() { w.UpdatePresentation(ctx) }); allocs != 0 {
		t.Fatalf("idle atlas allocates: %g", allocs)
	}
	board.hovered = 0
	w.UpdatePresentation(ctx)
	if len(board.dots) != 2 || board.dots[0].X != 51 || board.dots[1].ID != 2 {
		t.Fatalf("current-map dots = %+v", board.dots)
	}
	board.ClearRedraw()
	ctx.World.Player.X++
	w.UpdatePresentation(ctx)
	if !board.NeedsRedraw() {
		t.Fatal("visible preview did not update")
	}
	board.hovered = 1
	w.UpdatePresentation(ctx)
	if len(board.dots) != 0 {
		t.Fatal("remote preview used stale same-map coordinates")
	}
	w.Close()
	ctx.World.MapName = "geffen.rsw"
	w.UpdatePresentation(ctx)
	if board.currentMap != "prontera" {
		t.Fatal("closed atlas did presentation work")
	}
	if err := w.OpenWindow(ctx); err != nil || w.board.currentMap != "geffen" || w.scaled != scaled {
		t.Fatalf("reopen: %v", err)
	}
}

func TestWorldMapPublishedHoverClicksAndEscape(t *testing.T) {
	ctx := worldMapTestContext(t)
	a := uiapp.New()
	bridge := mailWindowTestApp{basicMenuTestApp{app: a}}
	ctx.UIManager.(*Manager).SetUIApp(bridge)
	ctx.UIApp = bridge
	w := &WorldMapWindow{}
	if err := w.OpenWindow(ctx); err != nil {
		t.Fatal(err)
	}
	a.Frame()
	a.Window().DrawTo(&uitest.MockCanvas{})
	r := w.board.mapRect(w.board.entries[0])
	p := w.board.ScreenBounds().Min.Add(r.Min.Sub(w.board.Bounds().Min)).Add(geometry.Pt(10, 10))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, p, p, event.ModNone))
	if w.board.hovered != 0 || w.previews["prontera"] == nil {
		t.Fatalf("hover = %d, preview = %v, point = %v, screen bounds = %v", w.board.hovered, w.previews["prontera"], p, w.board.ScreenBounds())
	}
	if len(w.previews) != 1 {
		t.Fatal("direct enter and translated move disagreed about the hovered map")
	}
	for _, button := range []event.Button{event.ButtonLeft, event.ButtonRight} {
		a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, button, 0, p, p, event.ModNone))
		a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, button, 0, p, p, event.ModNone))
		ctx.Input.SetMousePosition(int(p.X), int(p.Y))
		if !w.Update(ctx) {
			t.Fatal("map click leaked through atlas")
		}
	}
	// The title's magnifier toggles labels using the ordinary title button.
	p = geometry.Pt(float32(w.x+w.width-windowTitleButtonPadR-2*windowTitleButtonSize-windowTitleButtonGap+8), float32(w.y+14))
	a.Window().HandleEvent(event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
	a.Window().HandleEvent(event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
	if !w.showNames || !w.board.showNames {
		t.Fatal("magnifier did not toggle map names")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	if !w.Update(ctx) || w.IsOpen() {
		t.Fatal("Escape did not close atlas")
	}
}

func TestWorldMapRebindAndMissingAssets(t *testing.T) {
	ctx := worldMapTestContext(t)
	w := &WorldMapWindow{}
	if err := w.OpenWindow(Context{}); err == nil || w.IsOpen() {
		t.Fatal("missing assets opened a broken atlas")
	}
	if err := w.OpenWindow(ctx); err != nil {
		t.Fatal(err)
	}
	next := *w
	ctx.World = &world.World{MapName: "unknown_dungeon.rsw"}
	next.Rebind(ctx)
	if next.board == w.board || next.board.currentMap != "unknown_dungeon" || w.board.currentMap != "prontera" {
		t.Fatal("map transition reused the previous mode's presentation")
	}
}

func TestWorldMapSizeAndScaledHitTesting(t *testing.T) {
	for _, screen := range []image.Point{image.Pt(640, 480), image.Pt(1280, 720), image.Pt(1920, 1080)} {
		size := worldMapSize(image.Pt(1280, 1024), screen.X, screen.Y)
		if size.X > screen.X-16 || size.Y+ROWindowTitleHeight > screen.Y-16 || size.X > 1280 {
			t.Fatalf("%v does not fit %v", size, screen)
		}
	}
	b := &worldMapWidget{sourceSize: image.Pt(1280, 1024), entries: []res.WorldMapEntry{{MapName: "prontera", Bounds: image.Rect(812, 587, 870, 643)}}}
	b.SetBounds(geometry.NewRect(17, 28, 640, 512))
	if b.mapAt(geometry.Pt(438, 334)) != 0 || b.mapAt(geometry.Pt(20, 30)) != -1 {
		t.Fatal("scaled hit testing disagrees with map rectangles")
	}
}
