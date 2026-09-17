//go:build js && wasm

package ui

import (
	"strings"
	"syscall/js"
)

// DOM twins of the chat room windows: the "Make a Room" creation form
// (#goro-chatroom-create) and the in-room chat window (#goro-chatroom).
// The page renders both panels in the family styling; actions flow back
// through the shared "crc:"/"cr:" queues. Native builds keep the canvas
// windows.

// chatRoomCreateWebSync shows or hides the creation form. Field values
// live in the page, so the sync only carries visibility.
func chatRoomCreateWebSync(open bool) bool {
	sync := js.Global().Get("goroChatRoomCreateSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", open)
	sync.Invoke(obj)
	return true
}

type chatRoomWebState struct {
	open    bool
	title   string
	public  bool
	limit   uint16
	count   uint16
	owner   string
	members []string
	lines   []whisperWebLine
}

func chatRoomWebSync(state chatRoomWebState) bool {
	sync := js.Global().Get("goroChatRoomSync")
	if sync.Type() != js.TypeFunction {
		return false
	}
	hudWebInstallHooks()
	obj := js.Global().Get("Object").New()
	obj.Set("open", state.open)
	obj.Set("title", state.title)
	obj.Set("public", state.public)
	obj.Set("limit", int(state.limit))
	obj.Set("count", int(state.count))
	obj.Set("owner", state.owner)
	members := js.Global().Get("Array").New(len(state.members))
	for i, member := range state.members {
		members.SetIndex(i, member)
	}
	obj.Set("members", members)
	lines := js.Global().Get("Array").New(len(state.lines))
	for i, line := range state.lines {
		lo := js.Global().Get("Object").New()
		lo.Set("text", line.text)
		lo.Set("kind", line.kind)
		lines.SetIndex(i, lo)
	}
	obj.Set("lines", lines)
	sync.Invoke(obj)
	return true
}

// drainChatRoomCreateWebActions services the creation form's "crc:"
// actions. OK field values arrive joined with \x1f so a room title may
// contain colons.
func drainChatRoomCreateWebActions() (action ChatRoomCreateWindowAction, ok, cancelled bool) {
	for _, a := range hudWebDrainActions("crc:") {
		switch {
		case a == "crc:cancel":
			cancelled = true
		case strings.HasPrefix(a, "crc:ok"):
			// "crc:ok" + \x1f + title + \x1f + password + \x1f + limit + \x1f + type:
			// the leading \x1f leaves an empty first field after the split.
			fields := strings.Split(strings.TrimPrefix(a, "crc:ok"), "\x1f")
			if len(fields) != 5 {
				continue
			}
			action = ChatRoomCreateWindowAction{
				Title:    strings.TrimSpace(fields[1]),
				Password: strings.TrimSpace(fields[2]),
				Limit:    chatRoomLimit(fields[3]),
				Public:   fields[4] != "private",
			}
			ok = true
		}
	}
	return action, ok, cancelled
}

// drainChatRoomWebActions services the chat room panel's "cr:" actions.
func drainChatRoomWebActions() (message string, leave bool) {
	for _, a := range hudWebDrainActions("cr:") {
		switch {
		case a == "cr:leave":
			leave = true
		case strings.HasPrefix(a, "cr:send:"):
			message = strings.TrimPrefix(a, "cr:send:")
		}
	}
	return message, leave
}
