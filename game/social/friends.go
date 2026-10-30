package social

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func (m *System) OpenFriendRequest(ctx client.Context, request network.FriendRequest) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Someone"
	}
	glog.Debugf("friend request aid=%d gid=%d name=%q", request.AccountID, request.CharID, request.Name)
	m.bindings.Windows.FriendRequest.Open(ctx, "Friend Request", fmt.Sprintf("%s wants to be friends with you.", name), func() {
		if ctx.Network == nil {
			glog.Warnf("friend request accept failed: not connected")
			return
		}
		if err := ctx.Network.SendFriendRequestAck(request.AccountID, request.CharID, true); err != nil {
			glog.Warnf("friend request accept failed aid=%d gid=%d: %v", request.AccountID, request.CharID, err)
		}
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("friend request reject failed: not connected")
			return
		}
		if err := ctx.Network.SendFriendRequestAck(request.AccountID, request.CharID, false); err != nil {
			glog.Warnf("friend request reject failed aid=%d gid=%d: %v", request.AccountID, request.CharID, err)
		}
	})
}

func (m *System) AddFriendResultMessage(result network.FriendAddResult) {
	name := strings.TrimSpace(result.Name)
	if name == "" {
		name = "Player"
	}
	switch result.Result {
	case 0:
		m.bindings.Console.AddBlueMessage("You have become friends with %s.", name)
	case 1:
		m.bindings.Console.AddErrorMessage("%s does not want to be friends with you.", name)
	case 2:
		m.bindings.Console.AddErrorMessage("Your Friend List is full.")
	case 3:
		m.bindings.Console.AddErrorMessage("%s's Friend List is full.", name)
	default:
		m.bindings.Console.AddErrorMessage("Friend request failed.")
	}
}

func (m *System) SendAddFriend(ctx client.Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("add friend failed name=%q: not connected", name)
		return
	}
	if err := ctx.Network.SendAddFriend(name); err != nil {
		glog.Warnf("add friend failed name=%q: %v", name, err)
	}
}

func (m *System) openDeleteFriendConfirm(ctx client.Context, friend session.Friend) {
	name := strings.TrimSpace(friend.Name)
	if name == "" {
		name = "this friend"
	}
	m.bindings.Windows.FriendConfirm.Open(ctx, "Delete Friend", fmt.Sprintf("Delete %s from your friend list?", name), func() {
		if ctx.Network == nil {
			m.bindings.Console.AddErrorMessage("Delete friend failed: not connected.")
			return
		}
		if err := ctx.Network.SendDeleteFriend(friend.AccountID, friend.CharID); err != nil {
			m.bindings.Console.AddErrorMessage("Delete friend failed.")
			glog.Warnf("delete friend failed aid=%d gid=%d name=%q: %v", friend.AccountID, friend.CharID, friend.Name, err)
			return
		}
		m.bindings.Console.AddSystemMessage("Delete friend request sent for %s.", name)
	}, nil)
}

func PartyCanInvite(s *session.Session) bool {
	return partyCanManage(s)
}

func FriendNameInSession(s *session.Session, name string) bool {
	name = strings.TrimSpace(name)
	if s == nil || name == "" {
		return false
	}
	for _, friend := range s.Friends.List {
		if friend.Name == name {
			return true
		}
	}
	return false
}

func ApplyFriendsList(ctx client.Context, friends []network.Friend) {
	if ctx.Session == nil {
		return
	}
	known := make(map[[2]uint32]string, len(ctx.Session.Friends.List))
	for _, friend := range ctx.Session.Friends.List {
		if friend.Name != "" {
			known[[2]uint32{friend.AccountID, friend.CharID}] = friend.Name
		}
	}
	ctx.Session.Friends.List = ctx.Session.Friends.List[:0]
	for _, friend := range friends {
		next := sessionFriendFromNetwork(friend)
		if next.Name == "" {
			next.Name = known[[2]uint32{next.AccountID, next.CharID}]
		}
		ctx.Session.Friends.List = append(ctx.Session.Friends.List, next)
	}
	glog.Debugf("friend list received count=%d", len(ctx.Session.Friends.List))
}

func ApplyFriendState(ctx client.Context, state network.FriendState) {
	if ctx.Session == nil {
		return
	}
	for i := range ctx.Session.Friends.List {
		friend := &ctx.Session.Friends.List[i]
		if friend.AccountID != state.AccountID || friend.CharID != state.CharID {
			continue
		}
		friend.State = state.State
		if state.Name != "" {
			friend.Name = state.Name
		}
		glog.Debugf("friend state aid=%d gid=%d name=%q state=%d online=%t", friend.AccountID, friend.CharID, friend.Name, friend.State, friend.Online())
		return
	}
	ctx.Session.Friends.List = append(ctx.Session.Friends.List, session.Friend{
		AccountID: state.AccountID,
		CharID:    state.CharID,
		Name:      state.Name,
		State:     state.State,
	})
	glog.Debugf("friend state aid=%d gid=%d name=%q state=%d online=%t", state.AccountID, state.CharID, state.Name, state.State, state.State == 0)
}

func ApplyFriendAddResult(ctx client.Context, result network.FriendAddResult) {
	if ctx.Session == nil || result.Result != 0 {
		return
	}
	upsertSessionFriend(ctx.Session, session.Friend{
		AccountID: result.AccountID,
		CharID:    result.CharID,
		Name:      result.Name,
		State:     0,
	})
	glog.Debugf("friend added aid=%d gid=%d name=%q", result.AccountID, result.CharID, result.Name)
}

func ApplyFriendDelete(ctx client.Context, deleted network.FriendDelete) (session.Friend, bool) {
	if ctx.Session == nil {
		return session.Friend{}, false
	}
	for i := range ctx.Session.Friends.List {
		friend := ctx.Session.Friends.List[i]
		if friend.AccountID != deleted.AccountID || friend.CharID != deleted.CharID {
			continue
		}
		ctx.Session.Friends.List = append(ctx.Session.Friends.List[:i], ctx.Session.Friends.List[i+1:]...)
		glog.Debugf("friend deleted aid=%d gid=%d name=%q", friend.AccountID, friend.CharID, friend.Name)
		return friend, true
	}
	glog.Warnf("friend delete not found aid=%d gid=%d", deleted.AccountID, deleted.CharID)
	return session.Friend{}, false
}

func sessionFriendFromNetwork(friend network.Friend) session.Friend {
	return session.Friend{
		AccountID: friend.AccountID,
		CharID:    friend.CharID,
		Name:      friend.Name,
		State:     friend.State,
	}
}

func upsertSessionFriend(s *session.Session, friend session.Friend) {
	for i := range s.Friends.List {
		if s.Friends.List[i].AccountID != friend.AccountID || s.Friends.List[i].CharID != friend.CharID {
			continue
		}
		s.Friends.List[i] = friend
		return
	}
	s.Friends.List = append(s.Friends.List, friend)
}

func (m *System) UpdateFriendsWindow(ctx client.Context) bool {
	if m.bindings.Windows.FriendsWindow.Update(ctx) {
		switch action := m.bindings.Windows.FriendsWindow.PopAction(); action.Kind {
		case gameui.FriendsWindowActionPartySettings:
			m.bindings.Windows.PartySettings.Open(ctx)
		case gameui.FriendsWindowActionPartyCreate:
			m.bindings.Windows.PartyCreate.Open(ctx)
		case gameui.FriendsWindowActionPartyInvite:
			m.bindings.Windows.PartyInvite.Open(ctx)
		case gameui.FriendsWindowActionPartyLeave:
			if ctx.Network == nil {
				m.bindings.Console.AddErrorMessage("Leave party failed: not connected.")
			} else if err := ctx.Network.SendLeaveParty(); err != nil {
				m.bindings.Console.AddErrorMessage("Leave party failed.")
				glog.Warnf("leave party failed: %v", err)
			}
		case gameui.FriendsWindowActionFriendWhisper:
			m.bindings.Windows.WhisperWindows.Open(ctx, action.Friend.Name)
		case gameui.FriendsWindowActionFriendDelete:
			m.openDeleteFriendConfirm(ctx, action.Friend)
		case gameui.FriendsWindowActionFriendSettings:
			m.bindings.Windows.FriendSettings.Open(ctx)
		case gameui.FriendsWindowActionFriendBlockWhisper:
			name := strings.TrimSpace(action.Friend.Name)
			if ctx.Network == nil {
				m.bindings.Console.AddErrorMessage("Block whisper failed: not connected.")
			} else if err := ctx.Network.SendWhisperIgnore(name, false); err != nil {
				m.bindings.Console.AddErrorMessage("Block whisper failed.")
				glog.Warnf("block whisper failed name=%q: %v", name, err)
			}
		case gameui.FriendsWindowActionPartyMemberInfo:
			m.openPartyMemberInfo(ctx, action.PartyMember)
		case gameui.FriendsWindowActionPartyMemberWhisper:
			m.bindings.Windows.WhisperWindows.Open(ctx, action.PartyMember.Name)
		case gameui.FriendsWindowActionPartyMemberExpel:
			m.openExpelPartyMemberConfirm(ctx, action.PartyMember)
		}
		return true
	}
	return false
}
