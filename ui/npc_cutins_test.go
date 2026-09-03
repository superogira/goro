package ui

import (
	"image"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func TestNPCCutinUsesBottomAlignedPositions(t *testing.T) {
	screen := image.Rect(0, 0, 100, 80)
	tests := []struct {
		position uint8
		texture  *render.Image
		want     image.Rectangle
	}{
		{network.NPCCutinLeft, render.NewImage(20, 30), image.Rect(0, 50, 20, 80)},
		{network.NPCCutinCenter, render.NewImage(10, 15), image.Rect(45, 65, 55, 80)},
		{network.NPCCutinRight, render.NewImage(25, 5), image.Rect(75, 75, 100, 80)},
	}
	for _, test := range tests {
		got, ok := npcCutinBounds(screen, test.position, test.texture)
		if !ok || got != test.want {
			t.Fatalf("position %d bounds = %v, %v, want %v, true", test.position, got, ok, test.want)
		}
	}
}

func TestNPCCutinSupportsCenteredWindowModes(t *testing.T) {
	screen := image.Rect(0, 0, 200, 160)
	texture := render.NewImage(80, 100)
	tests := []struct {
		position uint8
		want     image.Rectangle
	}{
		{network.NPCCutinWindow, image.Rect(60, 16, 140, 144)},
		{network.NPCCutinWindowless, image.Rect(60, 30, 140, 130)},
	}
	for _, test := range tests {
		got, ok := npcCutinBounds(screen, test.position, texture)
		if !ok || got != test.want {
			t.Fatalf("position %d bounds = %v, %v, want %v, true", test.position, got, ok, test.want)
		}
	}
}

func TestNPCCutinNewImageReplacesPreviousIllustration(t *testing.T) {
	var cutin NPCCutinOverlay
	cutin.Apply(network.NPCCutin{Image: "old", Position: network.NPCCutinLeft}, render.NewImage(40, 20))
	cutin.Apply(network.NPCCutin{Image: "new", Position: network.NPCCutinRight}, render.NewImage(15, 20))

	if cutin.PointerBlocked(100, 80, 5, 70) {
		t.Fatal("replaced illustration still blocks its old bounds")
	}
	if !cutin.PointerBlocked(100, 80, 95, 70) {
		t.Fatal("replacement illustration does not block its bounds")
	}
}

func TestNPCCutinEmptyImageAndClearAllRemoveIllustration(t *testing.T) {
	var cutin NPCCutinOverlay
	cutin.Apply(network.NPCCutin{Image: "old", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	cutin.Apply(network.NPCCutin{Position: network.NPCCutinRight}, nil)
	if cutin.Visible() {
		t.Fatal("empty image did not clear the illustration")
	}

	cutin.Apply(network.NPCCutin{Image: "old", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	cutin.Apply(network.NPCCutin{Position: network.NPCCutinClear}, nil)
	if cutin.Visible() {
		t.Fatal("clear-all packet left an illustration visible")
	}
}

func TestNPCCutinUnknownPositionRemovesPreviousIllustration(t *testing.T) {
	var cutin NPCCutinOverlay
	cutin.Apply(network.NPCCutin{Image: "old", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	cutin.Apply(network.NPCCutin{Image: "unsupported", Position: 5}, render.NewImage(80, 80))
	if cutin.Visible() {
		t.Fatal("unsupported replacement left the previous illustration visible")
	}
}

func TestNPCWindowCutinCanBeDraggedByItsTitleBar(t *testing.T) {
	var cutin NPCCutinOverlay
	cutin.Apply(network.NPCCutin{Image: "window", Position: network.NPCCutinWindow}, render.NewImage(80, 100))
	inputState := input.NewState()
	ctx := Context{Input: inputState, ScreenW: 200, ScreenH: 240}
	cutin.Update(ctx)
	startX, startY := cutin.window.x, cutin.window.y

	inputState.SetMousePosition(startX+10, startY+10)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !cutin.Update(ctx) || !cutin.window.dragging {
		t.Fatal("windowed cut-in did not begin dragging from its title bar")
	}
	inputState.SetMousePosition(startX+30, startY+25)
	cutin.Update(ctx)
	if cutin.window.x != startX+20 || cutin.window.y != startY+15 {
		t.Fatalf("dragged position = %d,%d, want %d,%d", cutin.window.x, cutin.window.y, startX+20, startY+15)
	}
}

func TestNPCWindowlessCutinCloseControlClearsIllustration(t *testing.T) {
	var cutin NPCCutinOverlay
	cutin.Apply(network.NPCCutin{Image: "windowless", Position: network.NPCCutinWindowless}, render.NewImage(80, 100))
	w := newNPCWindowlessCutinWidget(cutin.texture.RGBA(), cutin.Clear)
	w.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(80, 100)))
	closeBounds := npcWindowlessCutinCloseBounds(w.Bounds())
	press := event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		0,
		closeBounds.Center(),
		closeBounds.Center(),
		0,
	)
	if !w.Event(widget.NewContext(), press) {
		t.Fatal("windowless cut-in close control did not consume click")
	}
	if cutin.Visible() {
		t.Fatal("windowless cut-in close control left illustration visible")
	}
}

func TestNPCCutinOversizeImageAnchorsAtScreenOrigin(t *testing.T) {
	texture := render.NewImage(120, 90)
	for _, position := range []uint8{network.NPCCutinCenter, network.NPCCutinRight} {
		got, ok := npcCutinBounds(image.Rect(0, 0, 100, 80), position, texture)
		want := image.Rect(0, 0, 120, 90)
		if !ok || got != want {
			t.Fatalf("position %d bounds = %v, %v, want %v, true", position, got, ok, want)
		}
	}
}
