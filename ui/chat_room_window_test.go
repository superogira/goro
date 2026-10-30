package ui

import (
	"fmt"
	"testing"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestChatRoomOwnerControlsAndSettingsFocus(t *testing.T) {
	ctx, _, app := newWindowInstanceTest()
	ctx.Session = &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Alice"}}
	var room ChatRoomWindow
	room.Open(ctx, "Room", 5, true, []string{"Alice", "Bob"})
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	buttons := collectEscapeMenuButtons(room.content)
	if len(buttons) != 6 {
		t.Fatalf("owner buttons = %d, want Close, Make Owner, Kick, Send, Settings, Leave", len(buttons))
	}
	for _, button := range buttons {
		bounds := button.ScreenBounds()
		if bounds.Min.X < float32(room.x) || bounds.Max.X > float32(room.x+room.width) || bounds.Max.Y > float32(room.y+room.height) {
			t.Fatalf("button outside the room: %+v", bounds)
		}
	}
	x, y := float32(room.x+chatRoomWindowW-chatRoomMembersW+20), float32(room.y+ROWindowTitleHeight+24+22+10)
	app.Window().HandleEvent(uitest.Click(x, y))
	app.Window().HandleEvent(uitest.Release(x, y))
	if room.selectedMember != "Bob" {
		t.Fatalf("selected member = %q", room.selectedMember)
	}
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	for i, want := range []ChatRoomWindowAction{{MakeOwner: "Bob"}, {Kick: "Bob"}, {Settings: true}} {
		button := buttons[[]int{1, 2, 4}[i]]
		point := button.ScreenBounds().Center()
		app.Window().HandleEvent(uitest.Click(point.X, point.Y))
		app.Window().HandleEvent(uitest.Release(point.X, point.Y))
		if got := room.PopAction(); got != want {
			t.Fatalf("action = %+v, want %+v", got, want)
		}
	}
	var settings ChatRoomCreateWindow
	room.OpenSettings(ctx, &settings)
	app.Frame()
	if app.Window().Context().FocusedWidget() != settings.titleField {
		t.Fatal("settings title did not receive focus")
	}
	room.AddMessage(ctx, "Bob : hello")
	app.Frame()
	if app.Window().Context().FocusedWidget() != settings.titleField || room.inputField.IsFocused() {
		t.Fatal("incoming message stole focus from settings")
	}
	settings.Close()
	room.SetOwner(ctx, "Bob")
	if room.IsOwner() || len(collectEscapeMenuButtons(room.content)) != 3 {
		t.Fatal("ownership loss retained management buttons")
	}
}

func TestChatRoomSettingsFormDoesNotCreateAnotherRoom(t *testing.T) {
	ctx := Context{ScreenW: 800, ScreenH: 600}
	var form ChatRoomCreateWindow
	form.OpenEdit(ctx, "Room", 5, false)
	if form.title != "Room" || form.limit != "5" || form.public {
		t.Fatal("settings were not prefilled")
	}
	form.password = " pass "
	form.submit(ctx)
	if action := form.PopAction(); !action.Edit || action.Password != " pass " || action.Title != "Room" {
		t.Fatalf("edit action = %+v", action)
	}
	form.Open(ctx)
	form.title = "New room"
	form.submit(ctx)
	if action := form.PopAction(); action.Edit || action.Password != "" || !action.Public || action.Limit != 20 {
		t.Fatalf("create retained edit state: %+v", action)
	}
}

func TestChatRoomEscapeRequestsLeave(t *testing.T) {
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	var window ChatRoomWindow
	window.Open(ctx, "Room", 20, true, []string{"Kivutar"})
	inputState.SetKey(input.KeyEscape, true)

	if !window.Update(ctx) {
		t.Fatal("Escape was not consumed")
	}
	if window.IsOpen() {
		t.Fatal("chat room window remained open after Escape")
	}
	if action := window.PopAction(); !action.Leave {
		t.Fatalf("action = %+v, want leave request", action)
	}
}

func TestChatRoomMessagesScrollToMeasuredBottom(t *testing.T) {
	var window ChatRoomWindow
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	window.Open(ctx, "Room", 20, true, []string{"Kivutar"})
	for i := 0; i < 20; i++ {
		window.AddMessage(ctx, fmt.Sprintf("line %d", i))
	}

	root := window.Window.Widget()
	root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(
		chatRoomWindowW,
		ROWindowTitleHeight+chatRoomWindowContentH+ROWindowFooterHeight,
	)))
	view := findChatRoomScrollView(root)
	if view == nil {
		t.Fatal("chat room scroll view was not found")
	}
	_, scrollY := view.ScrollOffset()
	want := view.ContentSize().Height - view.ViewportSize().Height
	if scrollY != want {
		t.Fatalf("scroll Y = %.1f, want measured bottom %.1f", scrollY, want)
	}
}

func findChatRoomScrollView(root widget.Widget) *scrollview.Widget {
	if view, ok := root.(*scrollview.Widget); ok {
		return view
	}
	parent, ok := root.(interface{ Children() []widget.Widget })
	if !ok {
		return nil
	}
	for _, child := range parent.Children() {
		if view := findChatRoomScrollView(child); view != nil {
			return view
		}
	}
	return nil
}
