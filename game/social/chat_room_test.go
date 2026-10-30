package social

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestPrivateChatRoomJoinAndCancel(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Input: input.NewState(), ScreenW: 800, ScreenH: 600}
	m := newTestSystem()
	actor := worldstate.Actor{ChatRoomID: 123, ChatRoomTitle: "Private", ChatRoomLimit: 5}
	m.RequestChatRoomEnter(ctx, actor)
	if !m.bindings.Windows.ChatRoomPassword.IsOpen() {
		t.Fatal("private room did not prompt for a password")
	}
	ctx.Input.SetKey(input.KeyEscape, true)
	m.UpdateChatRoomWindows(ctx)
	if m.pendingChatRoomID != 0 || m.PendingChatRoom.Title != "" {
		t.Fatal("cancel retained the pending join")
	}
	ctx.Input.SetKey(input.KeyEscape, false)
	ctx.Input.EndFrame()
	m.RequestChatRoomEnter(ctx, actor)
	m.bindings.Windows.ChatRoomPassword.Control(ctx, "cancel")
	m.UpdateChatRoomWindows(ctx)
	if m.pendingChatRoomID != 0 {
		t.Fatal("cancel before Update retained the pending join")
	}
	m.RequestChatRoomEnter(ctx, actor)
	// Empty passwords are valid for private rooms too; no request on Escape.
	m.bindings.Windows.ChatRoomPassword.Control(ctx, "confirm")
	m.UpdateChatRoomWindows(ctx)
	testutil.ReadPackets(t, server, []byte{0xd9, 0, 123, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	m.HandleChatRoomEnter(ctx, network.ChatRoomEnter{RoomID: 123, Members: []network.ChatRoomMember{{Name: "Alice", Owner: true}}})
	if !m.bindings.Windows.ChatRoom.IsOpen() || m.bindings.Windows.ChatRoom.RoomID != 123 || m.pendingChatRoomID != 0 {
		t.Fatal("join did not open the room and clear the request")
	}
}

func TestCancelChatRoomRequestsDiscardsQueuedPassword(t *testing.T) {
	m := newTestSystem()
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	m.RequestChatRoomEnter(ctx, worldstate.Actor{ChatRoomID: 123, ChatRoomTitle: "Private"})
	m.bindings.Windows.ChatRoomPassword.Control(ctx, "confirm")
	m.CancelChatRoomRequests()
	if m.pendingChatRoomID != 0 || m.PendingChatRoom.Title != "" || m.bindings.Windows.ChatRoomPassword.PopAction().Submitted {
		t.Fatal("a map transition retained a pending password submission")
	}
}

func TestChatRoomJoinWaitsForReply(t *testing.T) {
	for name, public := range map[string]bool{"private": false, "public": true} {
		t.Run(name, func(t *testing.T) {
			netClient, server := testutil.NewConnection(t, 20080910)
			ctx := client.Context{Network: netClient, ScreenW: 800, ScreenH: 600}
			m := newTestSystem()
			first := worldstate.Actor{ChatRoomID: 123, ChatRoomTitle: "First room", ChatRoomLimit: 5, ChatRoomPublic: public}
			second := worldstate.Actor{ChatRoomID: 456, ChatRoomTitle: "Second room", ChatRoomLimit: 10}
			m.RequestChatRoomEnter(ctx, first)
			if !public {
				m.bindings.Windows.ChatRoomPassword.Control(ctx, "confirm")
				m.UpdateChatRoomWindows(ctx)
			}
			testutil.ReadPackets(t, server, []byte{0xd9, 0, 123, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
			m.RequestChatRoomEnter(ctx, first)
			m.RequestChatRoomEnter(ctx, second)
			if m.pendingChatRoomID != first.ChatRoomID || m.PendingChatRoom.Title != first.ChatRoomTitle || m.bindings.Windows.ChatRoomPassword.IsOpen() {
				t.Fatal("another click replaced the pending join or opened a password prompt")
			}
			// The marker must arrive next: repeated clicks must not send another join.
			if err := netClient.Send([]byte{0x81, 0, 0}); err != nil {
				t.Fatal(err)
			}
			testutil.ReadPackets(t, server, []byte{0x81, 0, 0})
			m.HandleChatRoomEnterRefused(ctx, network.ChatRoomEnterRefused{Result: 0})
			m.RequestChatRoomEnter(ctx, second)
			if m.pendingChatRoomID != second.ChatRoomID || !m.bindings.Windows.ChatRoomPassword.IsOpen() {
				t.Fatal("a refused join blocked the next attempt")
			}
			m.bindings.Windows.ChatRoomPassword.Control(ctx, "cancel")
			m.UpdateChatRoomWindows(ctx)
			if m.pendingChatRoomID != 0 {
				t.Fatal("cancellation retained the pending join")
			}
		})
	}
}

func TestLeavingRoomClosesSettingsBeforeProcessingActions(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Input: input.NewState(), Session: &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Alice"}}}
	m := newTestSystem()
	m.bindings.Windows.ChatRoom.Open(ctx, "Room", 5, true, []string{"Alice"})
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Settings: true})
	ctx.Input.SetKey(input.KeyEscape, true)
	m.bindings.Windows.ChatRoom.Update(ctx)
	ctx.Input.EndFrame()
	m.UpdateChatRoomWindows(ctx)
	if m.bindings.Windows.ChatRoomCreate.IsOpen() {
		t.Fatal("leaving retained the settings form")
	}
	testutil.ReadPackets(t, server, []byte{0xe3, 0})
}

func TestChatRoomOwnerActionsWaitForServerAndRecheckPermissions(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Session: &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Alice"}}}
	m := newTestSystem()
	room := &m.bindings.Windows.ChatRoom
	room.Open(ctx, "Room", 5, true, []string{"Alice", "Bob"})
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{MakeOwner: "Bob"})
	want := make([]byte, 30)
	want[0] = 0xe0
	copy(want[6:], "Bob")
	testutil.ReadPackets(t, server, want)
	if !room.IsOwner() {
		t.Fatal("transfer changed ownership before server confirmation")
	}
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Kick: "Bob"})
	want = make([]byte, 26)
	want[0] = 0xe2
	copy(want[2:], "Bob")
	testutil.ReadPackets(t, server, want)
	if !room.CanManageMember("Bob") {
		t.Fatal("kick removed the member before server confirmation")
	}
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Settings: true})
	if !m.bindings.Windows.ChatRoomCreate.Editing() {
		t.Fatal("settings did not open in edit mode")
	}
	m.createChatRoomFromWindow(ctx, gameui.ChatRoomCreateWindowAction{Edit: true, Title: "New", Limit: 3, Public: true})
	testutil.ReadPackets(t, server, []byte{0xde, 0, 18, 0, 3, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 'N', 'e', 'w'})
	m.HandleChatRoomRoleChange(ctx, network.ChatRoomRoleChange{Owner: true, Name: "Bob"})
	if room.IsOwner() || m.bindings.Windows.ChatRoomCreate.IsOpen() {
		t.Fatal("ownership loss retained owner controls or the settings form")
	}
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Kick: "Bob"})
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{MakeOwner: "Alice"})
	m.createChatRoomFromWindow(ctx, gameui.ChatRoomCreateWindowAction{Edit: true, Title: "Stale", Limit: 3})
	// A marker verifies that stale actions sent nothing.
	if err := netClient.Send([]byte{0x81, 0, 0}); err != nil {
		t.Fatal(err)
	}
	testutil.ReadPackets(t, server, []byte{0x81, 0, 0})
}

func TestChatRoomKickClosesLocalRoomOnly(t *testing.T) {
	m := newTestSystem()
	ctx := client.Context{Session: &session.Session{CharID: 1, Selected: session.Character{ID: 1, Name: "Alice"}}}
	room := &m.bindings.Windows.ChatRoom
	room.Open(ctx, "Room", 5, true, []string{"Alice", "Bob"})
	m.HandleChatRoomMemberLeave(ctx, network.ChatRoomMemberLeave{Name: "Bob", Count: 1, Kicked: true})
	if !room.IsOpen() || room.CanManageMember("Bob") {
		t.Fatal("remote kick did not remove only the other member")
	}
	m.HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Settings: true})
	m.HandleChatRoomMemberLeave(ctx, network.ChatRoomMemberLeave{Name: "Alice", Kicked: true})
	if room.IsOpen() || m.bindings.Windows.ChatRoomCreate.IsOpen() || len(m.bindings.Console.Messages()) != 1 {
		t.Fatal("local kick did not close the room and settings, and report the kick")
	}
}

func TestChatRoomJoinRefusals(t *testing.T) {
	for result, want := range []string{
		"The chat room is full.", "Incorrect password.", "You have been kicked out of this chat room.", "",
		"Not enough Zeny to enter this chat room.", "Your level is too low to enter this chat room.",
		"Your level is too high to enter this chat room.", "Your job cannot enter this chat room.",
	} {
		m := newTestSystem()
		m.pendingChatRoomID = 123
		m.HandleChatRoomEnterRefused(client.Context{}, network.ChatRoomEnterRefused{Result: uint8(result)})
		messages := m.bindings.Console.Messages()
		if result == 3 {
			if len(messages) != 0 || m.pendingChatRoomID != 123 {
				t.Fatal("success marker was treated as a refusal")
			}
		} else if len(messages) != 1 || messages[0].Text != want || m.pendingChatRoomID != 0 {
			t.Fatalf("result %d: messages = %+v, pending = %d", result, messages, m.pendingChatRoomID)
		}
	}
}
