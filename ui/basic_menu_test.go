package ui

import (
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	"testing"
)

// inertOverlay is a minimal widget for manager tests.
type inertOverlay struct {
	widget.WidgetBase
}

func newInertOverlay() *inertOverlay {
	w := &inertOverlay{}
	w.SetVisible(true)
	return w
}

func (w *inertOverlay) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(100, 50))
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

// basicMenuTestApp adapts a ui app for tests.
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

// InvalidateLayout mirrors render.uiAppBridge: newly published in-place
// overlays rely on it to schedule the layout pass that sizes their content.
func (a basicMenuTestApp) InvalidateLayout() {
	if a.app.Window() != nil && a.app.Window().Context() != nil {
		a.app.Window().Context().Invalidate()
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
	if !character.dragging {
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
	if character.y > wantCharacterY {
		t.Fatalf("character drag y = %d, want clamped at or above %d", character.y, wantCharacterY)
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
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
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

	// Click the Status button through the uiapp (window-based menu).
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	point := geometry.Pt(
		float32(basicMenuX+basicMenuPad+basicMenuButtonW/2),
		float32(basicMenuY+basicMenuPad+basicMenuButtonH/2),
	)
	app.Window().HandleEvent(uitest.Click(point.X, point.Y))
	app.Window().HandleEvent(uitest.Release(point.X, point.Y))
	app.Frame()

	if carriedCalls != 1 {
		t.Fatalf("carried calls = %d, want 1", carriedCalls)
	}
	if originalCalls != 0 {
		t.Fatalf("original calls = %d, want 0", originalCalls)
	}
}
