package social

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestApplyFriendDeleteReturnsRemovedFriend(t *testing.T) {
	sessionState := &session.Session{
		Friends: session.Friends{List: []session.Friend{
			{AccountID: 10, CharID: 20, Name: "Alice"},
			{AccountID: 11, CharID: 21, Name: "Bob"},
		}},
	}

	removed, ok := ApplyFriendDelete(client.Context{Session: sessionState}, network.FriendDelete{AccountID: 10, CharID: 20})

	if !ok || removed.Name != "Alice" {
		t.Fatalf("removed=%+v ok=%t, want Alice", removed, ok)
	}
	if len(sessionState.Friends.List) != 1 || sessionState.Friends.List[0].Name != "Bob" {
		t.Fatalf("friends = %+v, want only Bob", sessionState.Friends.List)
	}
}
