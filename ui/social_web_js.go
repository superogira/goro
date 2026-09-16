//go:build js && wasm

package ui

import (
	"strconv"
	"strings"
	"syscall/js"

	"github.com/kivutar/goro/session"
)

// DOM twins of the social windows: the confirm modal (friend requests,
// party invitations, party member info, expel confirms — plus quit and
// disconnect), the Friends window (friends/party tabs), Friend Setup,
// Party Settings and the 1:1 whisper chat windows. The page renders the
// panels in the family styling; actions flow back through the shared
// queues below. Native builds keep the canvas windows.

// --- confirm modal (#goro-confirm) ---

func confirmWebSync(title, message string, okOnly, open bool) bool {
	sync := js.Global().Get("goroConfirmSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	obj.Set("title", title)
	obj.Set("message", message)
	obj.Set("okOnly", okOnly)
	sync.Invoke(obj)
	return true
}

// --- friends window (#goro-friends) ---

type friendsWebState struct {
	open      bool
	tab       string
	title     string
	friends   []session.Friend
	party     session.Party
	canManage bool
	selfAID   uint32
}

func friendsWebSync(state friendsWebState) bool {
	sync := js.Global().Get("goroFriendsSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.open)
	obj.Set("tab", state.tab)
	obj.Set("title", state.title)
	friends := js.Global().Get("Array").New(len(state.friends))
	for i, friend := range state.friends {
		fo := js.Global().Get("Object").New()
		fo.Set("aid", strconv.FormatUint(uint64(friend.AccountID), 10))
		fo.Set("name", friendDisplayName(friend))
		fo.Set("online", friend.Online())
		friends.SetIndex(i, fo)
	}
	obj.Set("friends", friends)
	party := js.Global().Get("Object").New()
	party.Set("active", state.party.Active())
	party.Set("name", state.party.Name)
	members := js.Global().Get("Array").New(len(state.party.Members))
	for i, member := range state.party.Members {
		mo := js.Global().Get("Object").New()
		mo.Set("aid", strconv.FormatUint(uint64(member.AccountID), 10))
		mo.Set("name", member.Name)
		mo.Set("leader", member.Leader())
		mo.Set("online", member.Online())
		mo.Set("map", member.MapName)
		mo.Set("hp", member.HP)
		mo.Set("maxHp", member.MaxHP)
		mo.Set("self", member.AccountID == state.selfAID)
		members.SetIndex(i, mo)
	}
	party.Set("members", members)
	obj.Set("party", party)
	obj.Set("canManage", state.canManage)
	sync.Invoke(obj)
	return true
}

func friendDisplayName(friend session.Friend) string {
	name := strings.TrimSpace(friend.Name)
	if name == "" {
		return "Unknown"
	}
	return name
}

// --- friend setup (#goro-friendsetup) ---

type friendSetupWebState struct {
	open         bool
	openStranger bool
	openFriends  bool
	alert        bool
}

func friendSetupWebSync(state friendSetupWebState) bool {
	sync := js.Global().Get("goroFriendSetupSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.open)
	obj.Set("strangers", state.openStranger)
	obj.Set("friends", state.openFriends)
	obj.Set("alert", state.alert)
	sync.Invoke(obj)
	return true
}

// --- party settings (#goro-partysetup) ---

type partySetupWebState struct {
	open         bool
	expShare     uint32
	refuseInvite bool
}

func partySetupWebSync(state partySetupWebState) bool {
	sync := js.Global().Get("goroPartySetupSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.open)
	obj.Set("expShare", state.expShare)
	obj.Set("refuseInvites", state.refuseInvite)
	sync.Invoke(obj)
	return true
}

// --- whisper windows (#goro-whisper) ---

type whisperWebWindow struct {
	target string
	lines  []whisperWebLine
}

type whisperWebLine struct {
	text string
	kind string // "in", "out", "err"
}

// whisperWebEnabled reports whether the page provides the DOM whisper
// panels, without touching their state.
func whisperWebEnabled() bool {
	return js.Global().Get("goroWhisperSync").Type() == js.TypeFunction
}

func whisperWebSync(windows []whisperWebWindow) bool {
	sync := js.Global().Get("goroWhisperSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	arr := js.Global().Get("Array").New(len(windows))
	for i, window := range windows {
		wo := js.Global().Get("Object").New()
		wo.Set("target", window.target)
		lines := js.Global().Get("Array").New(len(window.lines))
		for j, line := range window.lines {
			lo := js.Global().Get("Object").New()
			lo.Set("text", line.text)
			lo.Set("kind", line.kind)
			lines.SetIndex(j, lo)
		}
		wo.Set("lines", lines)
		arr.SetIndex(i, wo)
	}
	sync.Invoke(arr)
	return true
}

// drainSocialWebActions services one panel's action queue.
func drainSocialWebActions(prefix string) []string {
	return hudWebDrainActions(prefix)
}
