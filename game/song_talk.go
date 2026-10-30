package game

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
)

func (m *WorldMode) applyWorldEffectSideEffects(ctx client.Context, effect effects.Instance, starts time.Time) {
	switch effect.EffectID {
	case effects.TalkScream:
		m.applySongTalk(ctx, effect.ActorID, res.SongTalkScream, starts)
	case effects.TalkFrostJoke:
		m.applySongTalk(ctx, effect.ActorID, res.SongTalkFrostJoke, starts)
	}
}

func (m *WorldMode) applySongTalk(ctx client.Context, actorID uint32, kind res.SongTalkKind, now time.Time) {
	if actorID == 0 || ctx.Resources == nil || !scene.IsLocalActor(ctx, actorID) {
		return
	}
	text, ok := ctx.Resources.SongTalkLine(kind)
	if !ok {
		return
	}
	if ctx.Network != nil {
		name := strings.TrimSpace(scene.SelectedCharacterName(ctx.Session))
		if name != "" {
			if err := ctx.Network.SendGlobalChat(name, text); err == nil {
				return
			} else {
				glog.Warnf("song talk chat send failed actor=%d kind=%d: %v", actorID, kind, err)
			}
		}
	}
	m.actors().ApplySpeechBubble(ctx, network.ChatMessage{GID: actorID, Text: text}, now)
}
