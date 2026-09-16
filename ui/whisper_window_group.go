package ui

import (
	"strings"
)

// WhisperWindows keeps a conversation and draft per recipient for the session.
// Closing a window hides the conversation; opening it again restores it.
type WhisperWindows struct {
	conversations []*WhisperWindow
}

func (w *WhisperWindows) Find(target string) *WhisperWindow {
	for _, conversation := range w.conversations {
		if strings.EqualFold(conversation.target, strings.TrimSpace(target)) {
			return conversation
		}
	}
	return nil
}

func (w *WhisperWindows) open(ctx Context, target string) *WhisperWindow {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	conversation := w.Find(target)
	if conversation == nil {
		conversation = &WhisperWindow{}
		w.conversations = append(w.conversations, conversation)
	}
	if !conversation.IsOpen() {
		conversation.open(ctx, target)
	}
	return conversation
}

func (w *WhisperWindows) Open(ctx Context, target string) *WhisperWindow {
	conversation := w.open(ctx, target)
	if conversation != nil {
		conversation.Raise(ctx)
		conversation.focusInput()
	}
	return conversation
}

func (w *WhisperWindows) AddIncoming(ctx Context, sender, message string) {
	if conversation := w.open(ctx, sender); conversation != nil {
		conversation.AddIncoming(ctx, sender, message)
	}
}

func (w *WhisperWindows) IsOpen() bool {
	for _, conversation := range w.conversations {
		if conversation.IsOpen() {
			return true
		}
	}
	return false
}

func (w *WhisperWindows) Update(ctx Context) (bool, WhisperWindowAction) {
	// Widget callbacks run before world input. Drain sends independently of
	// which conversation happens to be under the pointer this frame.
	if whisperWebEnabled() {
		return w.updateWeb(ctx)
	}
	for _, conversation := range w.conversations {
		if conversation.IsOpen() {
			conversation.submitFromFocusedEnter(ctx)
		}
		if action := conversation.PopAction(); action.Target != "" {
			return true, action
		}
	}
	for _, conversation := range w.conversations {
		consumed := conversation.Update(ctx)
		if consumed {
			return true, WhisperWindowAction{}
		}
	}
	return false, WhisperWindowAction{}
}

// updateWeb drives the DOM whisper windows: pushes the conversations and
// services sends and closes from the page.
func (w *WhisperWindows) updateWeb(ctx Context) (bool, WhisperWindowAction) {
	for _, action := range drainSocialWebActions("wh:") {
		parts := strings.SplitN(action, ":", 4)
		if len(parts) < 3 {
			continue
		}
		target := parts[2]
		conversation := w.Find(target)
		if conversation == nil {
			continue
		}
		switch parts[1] {
		case "close":
			conversation.Close()
		case "send":
			if len(parts) == 4 && strings.TrimSpace(parts[3]) != "" {
				conversation.action = WhisperWindowAction{Target: conversation.target, Message: strings.TrimSpace(parts[3])}
			}
		}
	}
	var windows []whisperWebWindow
	for _, conversation := range w.conversations {
		if !conversation.IsOpen() {
			continue
		}
		entry := whisperWebWindow{target: conversation.target}
		for _, line := range conversation.lines {
			entry.lines = append(entry.lines, whisperWebLine{text: line.text, kind: whisperKind(line)})
		}
		windows = append(windows, entry)
	}
	whisperWebSync(windows)
	// The DOM panels do not cover the game input by existing — only a
	// real send consumes, so the rest of the window chain still updates.
	for _, conversation := range w.conversations {
		if action := conversation.PopAction(); action.Target != "" {
			return true, action
		}
	}
	return false, WhisperWindowAction{}
}

func (w *WhisperWindows) Rebind(ctx Context) {
	for _, conversation := range w.conversations {
		conversation.Rebind(ctx)
	}
}
