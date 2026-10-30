// Package social manages interactions between players.
package social

import (
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

type Bindings struct {
	Windows *Windows
	Console *gameui.ChatConsole
	Minimap *gameui.Minimap
}

type Windows struct {
	MailWindow           gameui.MailWindow
	FriendRequest        gameui.ConfirmModal
	FriendConfirm        gameui.ConfirmModal
	PartyRequest         gameui.ConfirmModal
	GuildRequest         gameui.ConfirmModal
	GuildAllianceRequest gameui.ConfirmModal
	GuildRelationConfirm gameui.ConfirmModal
	GuildMemberPrompt    gameui.TextPromptWindow
	AdoptionRequest      gameui.ConfirmModal
	FriendsWindow        gameui.FriendsWindow
	GuildWindow          gameui.GuildWindow
	FriendSettings       gameui.FriendSettingsWindow
	WhisperWindows       gameui.WhisperWindows
	ChatRoomCreate       gameui.ChatRoomCreateWindow
	ChatRoomPassword     gameui.TextPromptWindow
	ChatRoom             gameui.ChatRoomWindow
	PartySettings        gameui.PartySettingsWindow
	PartyCreate          gameui.PartyCreateWindow
	PartyInvite          gameui.PartyInviteWindow
	PartyInfo            gameui.ConfirmModal
	PlayerContext        gameui.PlayerContextMenu
	TradeRequest         gameui.ConfirmModal
	TradeWindow          gameui.TradeWindow
	VendingWindow        gameui.VendingWindow
}

type System struct {
	Mail              MailState
	GuildAction       gameui.GuildMemberAction
	GuildOpenPending  bool
	PendingChatRoom   network.ChatRoomCreate
	pendingChatRoomID uint32
	PendingTradeName  string

	guildEmblems   map[uint32]guildEmblem
	guildPortraits guildPortraitCache
	bindings       Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
