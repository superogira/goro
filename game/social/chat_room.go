package social

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *System) UpdateChatRoomWindows(ctx client.Context) bool {
	password := &m.bindings.Windows.ChatRoomPassword
	consumed := password.Update(ctx)
	if action := password.PopAction(); action.Submitted {
		m.enterChatRoom(ctx, action.Text)
		return true
	} else if action.Cancelled {
		m.PendingChatRoom = network.ChatRoomCreate{}
		m.pendingChatRoomID = 0
		return true
	}
	if consumed || password.IsOpen() {
		return true
	}
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		m.closeChatRoomSettings()
	}
	createConsumed := m.bindings.Windows.ChatRoomCreate.Update(ctx)
	if action := m.bindings.Windows.ChatRoomCreate.PopAction(); action.Title != "" {
		m.createChatRoomFromWindow(ctx, action)
		return true
	}
	if createConsumed || m.bindings.Windows.ChatRoomCreate.IsOpen() {
		return true
	}
	roomConsumed := m.bindings.Windows.ChatRoom.Update(ctx)
	if action := m.bindings.Windows.ChatRoom.PopAction(); action != (gameui.ChatRoomWindowAction{}) {
		m.HandleChatRoomAction(ctx, action)
		return true
	}
	return roomConsumed
}

func (m *System) createChatRoomFromWindow(ctx client.Context, action gameui.ChatRoomCreateWindowAction) {
	room := network.ChatRoomCreate{
		Title:    strings.TrimSpace(action.Title),
		Password: action.Password,
		Limit:    action.Limit,
		Public:   action.Public,
	}
	if room.Title == "" {
		return
	}
	if action.Edit {
		if !m.bindings.Windows.ChatRoom.IsOwner() {
			return
		}
		if ctx.Network == nil {
			m.bindings.Console.AddErrorMessage("Change chat room failed: not connected.")
			return
		}
		if err := ctx.Network.SendChangeChatRoom(room); err != nil {
			m.bindings.Console.AddErrorMessage("Change chat room failed.")
			glog.Warnf("chat room change failed: %v", err)
		}
		return
	}
	if m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Create chat room failed: not connected.")
		return
	}
	if err := ctx.Network.SendCreateChatRoom(room); err != nil {
		m.bindings.Console.AddErrorMessage("Create chat room failed.")
		glog.Warnf("chat room create failed title=%q limit=%d public=%t: %v", room.Title, room.Limit, room.Public, err)
		return
	}
	m.PendingChatRoom = room
}

func (m *System) HandleChatRoomCreateAck(ctx client.Context, ack network.ChatRoomCreateAck) {
	switch ack.Result {
	case 0:
		room := m.PendingChatRoom
		if room.Title == "" {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: 20, Public: true}
		}
		m.PendingChatRoom = network.ChatRoomCreate{}
		member := scene.SelectedCharacterName(ctx.Session)
		m.bindings.Windows.ChatRoom.Open(ctx, room.Title, room.Limit, room.Public, []string{member})
		m.bindings.Console.AddBlueMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	case 1, 2:
		m.PendingChatRoom = network.ChatRoomCreate{}
		m.bindings.Console.AddErrorMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	default:
		m.PendingChatRoom = network.ChatRoomCreate{}
		m.bindings.Console.AddErrorMessage("Create chat room failed.")
	}
}

func (m *System) ApplyChatRoomBoard(ctx client.Context, board network.ChatRoomBoard) {
	ApplyChatRoomBoardToWorld(ctx, board)
	glog.Debugf("chat room board actor=%d room=%d title=%q count=%d limit=%d public=%t", board.OwnerID, board.RoomID, board.Title, board.Count, board.Limit, board.Public)
}

func ApplyChatRoomBoardToWorld(ctx client.Context, board network.ChatRoomBoard) {
	if ctx.World == nil {
		return
	}
	if scene.IsLocalActor(ctx, board.OwnerID) {
		ctx.World.Player.ChatRoom = true
		ctx.World.Player.ChatRoomID = board.RoomID
		ctx.World.Player.ChatRoomTitle = board.Title
		ctx.World.Player.ChatRoomCount = board.Count
		ctx.World.Player.ChatRoomLimit = board.Limit
		ctx.World.Player.ChatRoomPublic = board.Public
		return
	}
	actor, ok := ctx.World.Actors[board.OwnerID]
	if !ok {
		actor = worldstate.Actor{ID: board.OwnerID}
	}
	actor.ChatRoom = true
	actor.ChatRoomID = board.RoomID
	actor.ChatRoomTitle = board.Title
	actor.ChatRoomCount = board.Count
	actor.ChatRoomLimit = board.Limit
	actor.ChatRoomPublic = board.Public
	ctx.World.Actors[board.OwnerID] = actor
}

func (m *System) ApplyChatRoomDestroy(ctx client.Context, destroy network.ChatRoomDestroy) {
	ApplyChatRoomDestroyToWorld(ctx, destroy)
	glog.Debugf("chat room board removed room=%d", destroy.RoomID)
}

func ApplyChatRoomDestroyToWorld(ctx client.Context, destroy network.ChatRoomDestroy) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.ChatRoomID == destroy.RoomID {
		clearActorChatRoom(&ctx.World.Player)
	}
	for id, actor := range ctx.World.Actors {
		if actor.ChatRoomID != destroy.RoomID {
			continue
		}
		clearActorChatRoom(&actor)
		ctx.World.Actors[id] = actor
		return
	}
}

func clearActorChatRoom(actor *worldstate.Actor) {
	actor.ChatRoom = false
	actor.ChatRoomID = 0
	actor.ChatRoomTitle = ""
	actor.ChatRoomCount = 0
	actor.ChatRoomLimit = 0
	actor.ChatRoomPublic = false
}

func ActorHasChatRoom(actor worldstate.Actor) bool {
	return actor.ChatRoom && actor.ChatRoomID != 0 && actor.ChatRoomTitle != ""
}

func (m *System) HandleChatRoomAction(ctx client.Context, action gameui.ChatRoomWindowAction) {
	room := &m.bindings.Windows.ChatRoom
	if action.Settings {
		room.OpenSettings(ctx, &m.bindings.Windows.ChatRoomCreate)
		return
	}
	if action.MakeOwner != "" || action.Kick != "" {
		name := action.MakeOwner
		if action.Kick != "" {
			name = action.Kick
		}
		if !room.CanManageMember(name) {
			return
		}
		if ctx.Network == nil {
			room.AddError(ctx, "Member update failed: not connected.")
			return
		}
		var err error
		if action.Kick != "" {
			err = ctx.Network.SendKickChatRoomMember(name)
		} else {
			err = ctx.Network.SendChangeChatRoomOwner(name)
		}
		if err != nil {
			room.AddError(ctx, "Member update failed.")
			glog.Warnf("chat room member update failed: %v", err)
		}
		return
	}
	if action.Leave {
		m.closeChatRoomSettings()
		if ctx.Network == nil {
			m.bindings.Console.AddErrorMessage("Leave chat room failed: not connected.")
			return
		}
		if err := ctx.Network.SendExitChatRoom(); err != nil {
			m.bindings.Console.AddErrorMessage("Leave chat room failed.")
			glog.Warnf("chat room leave failed: %v", err)
		}
		return
	}
	message := strings.TrimSpace(action.Message)
	if message == "" {
		return
	}
	name := scene.SelectedCharacterName(ctx.Session)
	if name == "" {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: missing character name")
		m.bindings.Console.AddErrorMessage("Chat room send failed: missing character name.")
		return
	}
	if ctx.Network == nil {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: not connected")
		m.bindings.Console.AddErrorMessage("Chat room send failed: not connected.")
		return
	}
	if err := client.SendGlobalChat(ctx, message); err != nil {
		m.bindings.Windows.ChatRoom.AddError(ctx, "send failed: "+err.Error())
		m.bindings.Console.AddErrorMessage("Chat room send failed.")
		glog.Warnf("chat room send failed: %v", err)
	}
}

func (m *System) RequestChatRoomEnter(ctx client.Context, actor worldstate.Actor) {
	if actor.ChatRoomID == 0 || m.pendingChatRoomID != 0 || m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	m.bindings.Windows.ChatRoomPassword.Close()
	m.pendingChatRoomID = actor.ChatRoomID
	m.PendingChatRoom = network.ChatRoomCreate{
		Title: actor.ChatRoomTitle, Limit: actor.ChatRoomLimit, Public: actor.ChatRoomPublic,
	}
	if !actor.ChatRoomPublic {
		m.bindings.Windows.ChatRoomPassword.OpenPassword(ctx, "Enter Chat Room", actor.ChatRoomTitle, 8)
		return
	}
	m.enterChatRoom(ctx, "")
}

func (m *System) enterChatRoom(ctx client.Context, password string) {
	if m.pendingChatRoomID == 0 || m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Enter chat room failed: not connected.")
	} else if err := ctx.Network.SendEnterChatRoom(m.pendingChatRoomID, password); err != nil {
		m.bindings.Console.AddErrorMessage("Enter chat room failed.")
		glog.Warnf("chat room enter failed room=%d: %v", m.pendingChatRoomID, err)
	} else {
		return
	}
	m.pendingChatRoomID = 0
	m.PendingChatRoom = network.ChatRoomCreate{}
}

func (m *System) HandleChatRoomEnter(ctx client.Context, enter network.ChatRoomEnter) {
	members := make([]string, 0, len(enter.Members))
	owner := ""
	for _, member := range enter.Members {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		members = append(members, name)
		if member.Owner {
			owner = name
		}
	}
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		room := m.PendingChatRoom
		if room.Title == "" || enter.RoomID != m.pendingChatRoomID {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: uint16(len(members)), Public: true}
		}
		m.bindings.Windows.ChatRoom.Open(ctx, room.Title, room.Limit, room.Public, members)
	}
	m.bindings.Windows.ChatRoom.RoomID = enter.RoomID
	m.bindings.Windows.ChatRoom.SetMembers(ctx, members, owner)
	m.bindings.Windows.ChatRoomPassword.Close()
	m.pendingChatRoomID = 0
	m.PendingChatRoom = network.ChatRoomCreate{}
}

func (m *System) HandleChatRoomMemberJoin(ctx client.Context, join network.ChatRoomMemberJoin) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(join.Name)
	m.bindings.Windows.ChatRoom.AddMember(ctx, name, join.Count)
	m.bindings.Windows.ChatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, 179, "%s entered the chat room.", name))
}

func (m *System) HandleChatRoomMemberLeave(ctx client.Context, leave network.ChatRoomMemberLeave) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(leave.Name)
	messageID := 180
	fallback := "%s left the chat room."
	if leave.Kicked {
		messageID = 181
		fallback = "%s has been kicked out of the chat room."
	}
	if strings.EqualFold(name, scene.SelectedCharacterName(ctx.Session)) {
		m.bindings.Windows.ChatRoom.Close()
		m.closeChatRoomSettings()
		if leave.Kicked {
			m.bindings.Console.AddErrorMessage("%s", chatRoomMemberMessage(ctx, messageID, fallback, name))
		}
		return
	}
	m.bindings.Windows.ChatRoom.RemoveMember(ctx, name, leave.Count)
	m.bindings.Windows.ChatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, messageID, fallback, name))
}

func (m *System) HandleChatRoomChange(ctx client.Context, change network.ChatRoomChange) {
	room := &m.bindings.Windows.ChatRoom
	if !room.IsOpen() || (room.RoomID != 0 && room.RoomID != change.RoomID) {
		return
	}
	room.RoomID = change.RoomID
	m.bindings.Windows.ChatRoom.UpdateRoom(ctx, change.Title, change.Limit, change.Count, change.Public)
}

func (m *System) HandleChatRoomRoleChange(ctx client.Context, role network.ChatRoomRoleChange) {
	if !m.bindings.Windows.ChatRoom.IsOpen() || !role.Owner {
		return
	}
	m.bindings.Windows.ChatRoom.SetOwner(ctx, role.Name)
	if !m.bindings.Windows.ChatRoom.IsOwner() {
		m.closeChatRoomSettings()
	}
}

func (m *System) closeChatRoomSettings() {
	if m.bindings.Windows.ChatRoomCreate.Editing() {
		m.bindings.Windows.ChatRoomCreate.Close()
		m.bindings.Windows.ChatRoomCreate.PopAction()
	}
}

func (m *System) CancelChatRoomRequests() {
	m.bindings.Windows.ChatRoomPassword.Close()
	m.bindings.Windows.ChatRoomPassword.PopAction()
	m.closeChatRoomSettings()
	m.pendingChatRoomID = 0
	m.PendingChatRoom = network.ChatRoomCreate{}
}

func (m *System) HandleChatRoomEnterRefused(ctx client.Context, refusal network.ChatRoomEnterRefused) {
	if refusal.Result == 3 {
		return
	}
	m.pendingChatRoomID = 0
	m.PendingChatRoom = network.ChatRoomCreate{}
	m.bindings.Windows.ChatRoomPassword.Close()
	messageID := 0
	message := "Unable to enter the chat room."
	switch refusal.Result {
	case 0:
		messageID, message = 67, "The chat room is full."
	case 1:
		messageID, message = 7, "Incorrect password."
	case 2:
		messageID, message = 68, "You have been kicked out of this chat room."
	case 4:
		messageID, message = 55, "Not enough Zeny to enter this chat room."
	case 5:
		messageID, message = 432, "Your level is too low to enter this chat room."
	case 6:
		messageID, message = 433, "Your level is too high to enter this chat room."
	case 7:
		messageID, message = 434, "Your job cannot enter this chat room."
	}
	if ctx.Resources != nil && messageID != 0 {
		if text, ok := ctx.Resources.MsgString(messageID); ok && strings.TrimSpace(text) != "" {
			message = text
		}
	}
	m.bindings.Console.AddErrorMessage("%s", message)
}

func (m *System) AddChatRoomMessage(ctx client.Context, chat network.ChatMessage) {
	if !m.bindings.Windows.ChatRoom.IsOpen() {
		return
	}
	text := FormatConsoleMessage(ctx.Resources, chat)
	if text == "" {
		return
	}
	m.bindings.Windows.ChatRoom.AddMessage(ctx, text)
}

func chatRoomMemberMessage(ctx client.Context, messageID int, fallback string, name string) string {
	name = strings.TrimSpace(name)
	message := ""
	if ctx.Resources != nil {
		message, _ = ctx.Resources.MsgString(messageID)
	}
	if message == "" {
		message = fallback
	}
	if strings.Contains(message, "%s") {
		return strings.Replace(message, "%s", name, 1)
	}
	if name != "" {
		return strings.TrimSpace(message + " " + name)
	}
	return message
}

func chatRoomCreateAckMessage(ctx client.Context, ack network.ChatRoomCreateAck) string {
	if ctx.Resources != nil {
		if text, ok := ctx.Resources.MsgString(64 + int(ack.Result)); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	switch ack.Result {
	case 0:
		return "Chat room has been created."
	case 1:
		return "Chat room limit exceeded."
	case 2:
		return "Same chat room already exists."
	default:
		return fmt.Sprintf("Create chat room failed (%d).", ack.Result)
	}
}
