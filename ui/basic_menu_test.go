package ui

import (
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

type inertOverlay struct {
	widget.WidgetBase
}

func newInertOverlay() *inertOverlay {
	w := &inertOverlay{}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *inertOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.BiggestFinite(1, 1)
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *inertOverlay) Draw(widget.Context, widget.Canvas) {}

func (w *inertOverlay) Event(widget.Context, event.Event) bool {
	return false
}

func (w *inertOverlay) Children() []widget.Widget {
	return nil
}

type basicMenuTestApp struct {
	app *uiapp.App
}

func (a basicMenuTestApp) SetUIRoot(root widget.Widget) {
	a.app.SetRoot(root)
}

func (a basicMenuTestApp) Frame() {
	a.app.Frame()
}

func (a basicMenuTestApp) Invalidate() {
	if a.app.Window() != nil && a.app.Window().Context() != nil {
		a.app.Window().Context().Invalidate()
	}
}

func (a basicMenuTestApp) InvalidateRect(rect geometry.Rect) {
	if a.app.Window() != nil && a.app.Window().Context() != nil && !rect.IsEmpty() {
		a.app.Window().Context().InvalidateRect(rect)
	}
}

func (a basicMenuTestApp) RequestFullRepaint() {
	if a.app.Window() != nil {
		a.app.Window().RequestFullRepaint()
	}
}

func (a basicMenuTestApp) WidgetContext() widget.Context {
	if a.app.Window() == nil {
		return nil
	}
	return a.app.Window().Context()
}

func (a basicMenuTestApp) Cursor() widget.CursorType {
	return a.app.Window().Context().Cursor()
}

func (a basicMenuTestApp) HoveredWidget() widget.Widget {
	return a.app.Window().HoveredWidget()
}

func TestBasicMenuRowsUsePointerCursor(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx := client.Context{
		Input:     input.NewState(),
		Session:   &session.Session{Selected: session.Character{Name: "Kivutar"}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	var character CharacterWindow
	var menu BasicMenu
	character.Update(ctx)
	menu.Update(ctx, BasicMenuCallbacks{})
	manager.AddOverlay(positionedWidget(newInertOverlay(), 600, 16, 188, 206))

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})

	firstRow := geometry.Pt(
		float32(basicMenuX+basicMenuPad+basicMenuButtonW/2),
		float32(basicMenuY+basicMenuPad+basicMenuButtonH/2),
	)
	secondRow := geometry.Pt(
		firstRow.X,
		float32(basicMenuY+basicMenuPad+basicMenuButtonH+basicMenuGapY+basicMenuButtonH/2),
	)

	app.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, firstRow, firstRow, event.ModNone))
	if got := app.Window().Context().Cursor(); got != widget.CursorPointer {
		t.Fatalf("first row cursor = %v, want pointer, hovered=%T", got, app.Window().HoveredWidget())
	}

	app.Window().HandleEvent(event.NewMouseEvent(event.MouseMove, event.ButtonNone, 0, secondRow, secondRow, event.ModNone))
	if got := app.Window().Context().Cursor(); got != widget.CursorPointer {
		t.Fatalf("second row cursor = %v, want pointer, hovered=%T", got, app.Window().HoveredWidget())
	}
}

func TestBasicMenuFollowsCharacterWindow(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{Selected: session.Character{Name: "Kivutar"}},
		ScreenW: 800,
		ScreenH: 600,
	}
	var character CharacterWindow
	var menu BasicMenu
	character.Update(ctx)
	menu.FollowCharacterWindow(ctx, &character)
	menu.Update(ctx, BasicMenuCallbacks{})

	if menu.x != character.x || menu.y != character.y+character.height+basicMenuFollowGap {
		t.Fatalf("basic menu position = %d,%d, want attached below character at %d,%d", menu.x, menu.y, character.x, character.y+character.height+basicMenuFollowGap)
	}
	character.setPosition(ctx, 120, 90)
	menu.FollowCharacterWindow(ctx, &character)
	if menu.x != 120 || menu.y != 90+character.height+basicMenuFollowGap {
		t.Fatalf("moved basic menu position = %d,%d, want 120,%d", menu.x, menu.y, 90+character.height+basicMenuFollowGap)
	}
}

func TestCharacterDragKeepsAttachedBasicMenuOnScreen(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		Session: &session.Session{Selected: session.Character{Name: "Kivutar"}},
		ScreenW: 800,
		ScreenH: 300,
	}
	var character CharacterWindow
	var menu BasicMenu
	character.Update(ctx)
	menu.FollowCharacterWindow(ctx, &character)
	menu.Update(ctx, BasicMenuCallbacks{})

	inputState.SetMousePosition(character.x+10, character.y+5)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !character.Update(ctx) {
		t.Fatal("character window drag start was not consumed")
	}
	if !character.dragLayer {
		t.Fatal("character window did not enter drag state")
	}
	menu.FollowCharacterWindow(ctx, &character)
	inputState.EndFrame()
	inputState.SetMousePosition(700, 1000)
	if !character.Update(ctx) {
		t.Fatal("character window drag move was not consumed")
	}
	menu.FollowCharacterWindow(ctx, &character)

	_, menuHeight := basicMenuSize()
	wantCharacterY := ctx.ScreenH - character.height - basicMenuFollowGap - menuHeight
	if character.y != wantCharacterY {
		t.Fatalf("character drag y = %d, want clamped at %d", character.y, wantCharacterY)
	}
	if menu.y != character.y+character.height+basicMenuFollowGap {
		t.Fatalf("basic menu y = %d, want attached y %d", menu.y, character.y+character.height+basicMenuFollowGap)
	}
	if menu.y+menu.height > ctx.ScreenH {
		t.Fatalf("attached basic menu bottom = %d, beyond screen limit %d", menu.y+menu.height, ctx.ScreenH)
	}

	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	character.Update(ctx)
	if character.dragLayer {
		t.Fatal("character window drag state did not end on release")
	}
	menu.FollowCharacterWindow(ctx, &character)
	if menu.x != character.x || menu.y != character.y+character.height+basicMenuFollowGap {
		t.Fatalf("basic menu detached after drag: %d,%d want %d,%d", menu.x, menu.y, character.x, character.y+character.height+basicMenuFollowGap)
	}
}


func TestBasicMenuRebindRefreshesButtonCallbacks(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	var original BasicMenu
	original.Update(ctx, BasicMenuCallbacks{})
	carried := original
	originalCalls := 0
	carriedCalls := 0
	original.Rebind(ctx, BasicMenuCallbacks{
		OnStatus: func() { originalCalls++ },
	})
	carried.Rebind(ctx, BasicMenuCallbacks{
		OnStatus: func() { carriedCalls++ },
	})

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	point := geometry.Pt(
		float32(basicMenuX+basicMenuPad+basicMenuButtonW/2),
		float32(basicMenuY+basicMenuPad+basicMenuButtonH/2),
	)
	app.Window().HandleEvent(uitest.Click(point.X, point.Y))
	app.Window().HandleEvent(uitest.Release(point.X, point.Y))

	if carriedCalls != 1 {
		t.Fatalf("carried calls = %d, want 1", carriedCalls)
	}
	if originalCalls != 0 {
		t.Fatalf("original calls = %d, want 0", originalCalls)
	}
}

func TestBasicMenuButtonDoesNotReinvokeFromEnterKey(t *testing.T) {
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	itemCalls := 0
	var menu BasicMenu
	menu.Update(ctx, BasicMenuCallbacks{
		OnItems: func() { itemCalls++ },
	})

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	point := geometry.Pt(
		float32(basicMenuX+basicMenuPad+2*(basicMenuButtonW+basicMenuGapX)+basicMenuButtonW/2),
		float32(basicMenuY+basicMenuPad+basicMenuButtonH/2),
	)
	app.Window().HandleEvent(uitest.Click(point.X, point.Y))
	app.Window().HandleEvent(uitest.Release(point.X, point.Y))
	if itemCalls != 1 {
		t.Fatalf("item calls after click = %d, want 1", itemCalls)
	}
	if focused := app.Window().Context().FocusedWidget(); focused != nil {
		t.Fatalf("basic menu button kept keyboard focus: %T", focused)
	}

	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyRelease, event.KeyEnter, 0, event.ModNone))
	if itemCalls != 1 {
		t.Fatalf("item calls after enter = %d, want 1", itemCalls)
	}
}
