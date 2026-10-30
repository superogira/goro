package actors

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

const (
	speechBubbleBaseDuration = 3500 * time.Millisecond
	speechBubblePerRune      = 40 * time.Millisecond
	speechBubbleMaxDuration  = 8500 * time.Millisecond
	speechBubbleMaxWidth     = 220
)

type SpeechBubble struct {
	Text    string
	Expires time.Time
	Ground  *[2]int
}

func (m *System) ApplySpeechBubble(ctx client.Context, chat network.ChatMessage, now time.Time) {
	if ctx.Config.Headless {
		return
	}
	if strings.TrimSpace(chat.Text) == "" {
		return
	}
	text := speechBubbleText(chat.Text)
	x, y, onTrap := m.bindings.Skills.SkillUnitModelCell(chat.GID)
	if onTrap {
		text = strings.TrimSpace(chat.Text)
	}
	if text == "" {
		return
	}
	actorIDs := speechBubbleActorIDs(ctx, chat)
	if len(actorIDs) == 0 {
		return
	}
	if m.SpeechBubbles == nil {
		m.SpeechBubbles = make(map[uint32]SpeechBubble)
	}
	duration := speechBubbleBaseDuration + time.Duration(len([]rune(text)))*speechBubblePerRune
	if duration > speechBubbleMaxDuration {
		duration = speechBubbleMaxDuration
	}
	for _, actorID := range actorIDs {
		if actorID == 0 {
			continue
		}
		m.SpeechBubbles[actorID] = SpeechBubble{
			Text:    text,
			Expires: now.Add(duration),
		}
		if onTrap {
			m.SpeechBubbles[actorID] = SpeechBubble{Text: text, Expires: now.Add(5 * time.Second), Ground: &[2]int{x, y}}
		}
	}
}

func speechBubbleActorIDs(ctx client.Context, chat network.ChatMessage) []uint32 {
	if chat.GID != 0 {
		ids := []uint32{chat.GID}
		if ctx.Session != nil {
			if chat.GID == ctx.Session.AccountID && ctx.Session.CharID != 0 {
				ids = append(ids, ctx.Session.CharID)
			} else if chat.GID == ctx.Session.CharID && ctx.Session.AccountID != 0 {
				ids = append(ids, ctx.Session.AccountID)
			}
		}
		return ids
	}
	name, _, ok := strings.Cut(strings.TrimSpace(chat.Text), " : ")
	if !ok || ctx.Session == nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(name), scene.SelectedCharacterName(ctx.Session)) {
		return nil
	}
	ids := make([]uint32, 0, 2)
	if ctx.Session.CharID != 0 {
		ids = append(ids, ctx.Session.CharID)
	}
	if ctx.Session.AccountID != 0 {
		ids = append(ids, ctx.Session.AccountID)
	}
	return ids
}

func speechBubbleText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if _, message, ok := strings.Cut(text, " : "); ok {
		return strings.TrimSpace(message)
	}
	return text
}

const actorSpeechBubbleGap = 6.0

func ActorSpeechBubbleBottomY(baseY, scale float64) float64 {
	return ActorSpriteTopY(baseY, scale) - actorSpeechBubbleGap
}

func (m *System) DrawSpeechBubbles(screen *render.Frame, ctx client.Context, projection scene.Projection, entries []scene.ActorDrawEntry, now time.Time) {
	if len(m.SpeechBubbles) == 0 {
		return
	}
	for id, bubble := range m.SpeechBubbles {
		if now.After(bubble.Expires) {
			delete(m.SpeechBubbles, id)
		} else if bubble.Ground != nil {
			x, y := float64(bubble.Ground[0]), float64(bubble.Ground[1])
			if p, ok := projection.ProjectPoint(scene.CellCenter(x), scene.CellCenter(y), scene.TerrainHeightAt(ctx.World, x, y)); ok {
				render.DrawUISpeechBubble(screen, bubble.Text, float64(p.X), float64(p.Y)-actorSpeechBubbleGap, speechBubbleMaxWidth)
			}
		}
	}
	if len(m.SpeechBubbles) == 0 {
		return
	}
	for _, entry := range entries {
		bubble, ok := m.SpeechBubbles[entry.Actor.ID]
		if !ok || strings.TrimSpace(bubble.Text) == "" {
			continue
		}
		bottomY := ActorSpeechBubbleBottomY(entry.ScreenY, entry.Scale)
		render.DrawUISpeechBubble(screen, bubble.Text, entry.ScreenX, bottomY, speechBubbleMaxWidth)
	}
}
