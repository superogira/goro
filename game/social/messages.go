package social

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

func FormatConsoleMessage(manager *res.Manager, chat network.ChatMessage) string {
	if chat.Text != "" {
		return chat.Text
	}
	if chat.MessageID < 0 {
		return ""
	}
	text := ""
	if manager != nil {
		text, _ = manager.MsgString(chat.MessageID)
	}
	if text == "" {
		text = fmt.Sprintf("message #%d", chat.MessageID)
	}
	if chat.Value != 0 {
		if strings.Contains(text, "%") {
			text = fmt.Sprintf(text, chat.Value)
		} else {
			text = fmt.Sprintf("%s %d", text, chat.Value)
		}
	}
	if chat.SkillID != 0 {
		text = fmt.Sprintf("skill %d: %s", chat.SkillID, text)
	}
	return text
}

func AddConsoleMessage(console *gameui.ChatConsole, manager *res.Manager, chat network.ChatMessage) {
	if console == nil {
		return
	}
	text := FormatConsoleMessage(manager, chat)
	if text == "" {
		return
	}
	if chat.HasColor {
		console.AddColoredMessage(ROClientColor(chat.Color), "%s", text)
		return
	}
	if chat.Text == "" || !strings.Contains(text, " : ") {
		console.AddSystemMessage("%s", text)
		return
	}
	console.AddMessage("%s", text)
}

func ROClientColor(value uint32) color.RGBA {
	return color.RGBA{
		R: uint8(value),
		G: uint8(value >> 8),
		B: uint8(value >> 16),
		A: 255,
	}
}

func AddWhisperMessage(console *gameui.ChatConsole, whisper network.WhisperMessage) {
	if console == nil {
		return
	}
	if whisper.Sender == "" || whisper.Message == "" {
		return
	}
	console.AddBlueMessage("[ From %s ] : %s", whisper.Sender, whisper.Message)
}

func AddWhisperAck(console *gameui.ChatConsole, manager *res.Manager, ack network.WhisperAck) {
	if console == nil || ack.Result == 0 {
		return
	}
	console.AddErrorMessage("%s", whisperAckMessage(manager, ack))
}

func AddWhisperIgnoreAck(console *gameui.ChatConsole, ack network.WhisperIgnoreAck) {
	if console == nil {
		return
	}
	if ack.Result != 0 {
		console.AddErrorMessage("%s", whisperIgnoreAckFailure(ack))
		return
	}
	console.AddSystemMessage("%s", whisperIgnoreAckSuccess(ack))
}

func AddWhisperIgnoreList(console *gameui.ChatConsole, manager *res.Manager, names []string) {
	if console == nil {
		return
	}
	messageID, heading := 63, "Whisper block list:"
	if len(names) == 0 {
		messageID, heading = 62, "Your whisper block list is empty."
	}
	if manager != nil {
		if text, ok := manager.MsgString(messageID); ok && text != "" {
			heading = text
		}
	}
	messageColor := color.RGBA{G: 255, B: 255, A: 255}
	console.AddColoredMessage(messageColor, "%s", heading)
	for _, name := range names {
		console.AddColoredMessage(messageColor, "%s", name)
	}
}

func whisperIgnoreAckSuccess(ack network.WhisperIgnoreAck) string {
	if ack.TargetAll {
		if ack.Allow {
			return "Whispers from everyone are allowed."
		}
		return "Whispers from everyone are blocked."
	}
	if ack.Allow {
		return "Whispers from that player are allowed."
	}
	return "Whispers from that player are blocked."
}

func whisperIgnoreAckFailure(ack network.WhisperIgnoreAck) string {
	if !ack.TargetAll && ack.Result == 2 {
		return "Whisper block list is full."
	}
	if ack.TargetAll {
		if ack.Allow {
			return "Allow all whispers failed."
		}
		return "Block all whispers failed."
	}
	if ack.Allow {
		return "Allow whisper failed."
	}
	return "Block whisper failed."
}

func whisperAckMessage(manager *res.Manager, ack network.WhisperAck) string {
	message := ""
	if manager != nil {
		message, _ = manager.MsgString(int(147 + ack.Result))
	}
	if message == "" {
		message = "Whisper failed."
	}
	return message
}
