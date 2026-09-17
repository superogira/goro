//go:build !js || !wasm

package ui

// Native stubs: no DOM panels exist, so the callers fall back to the
// canvas windows.

func chatRoomCreateWebSync(open bool) bool { return false }

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

func chatRoomWebSync(state chatRoomWebState) bool { return false }

func drainChatRoomCreateWebActions() (action ChatRoomCreateWindowAction, ok, cancelled bool) {
	return ChatRoomCreateWindowAction{}, false, false
}

func drainChatRoomWebActions() (message string, leave bool) { return "", false }
