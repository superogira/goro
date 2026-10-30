package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	gameui "github.com/kivutar/goro/ui"
)


func (m *WorldMode) openPlayerContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || m.mapPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedPlayerTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorSystem.ActorDeaths)
	if !ok {
		return false
	}
	canManageGuildRelations := social.GuildCanManageRelations(ctx.Session, actor.GuildID)
	m.ui.social.PlayerContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, gameui.PlayerContextOptions{
		CanAddFriend:       !social.FriendNameInSession(ctx.Session, actor.Name),
		CanInviteParty:     social.PartyCanInvite(ctx.Session),
		CanInviteGuild:     social.GuildCanInvitePlayer(ctx.Session, actor.GuildID),
		CanAdopt:           social.CanRequestAdoption(ctx.Session, actor),
		CanRequestAlliance: canManageGuildRelations,
		CanDeclareHostile:  canManageGuildRelations,
		ShowGM:             actors.LocalPlayerIsAdmin(ctx),
	})
	return true
}

const siegeGuildEmblemSize = 24

func (m *WorldMode) drawSiegeGuildEmblems(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, entries []scene.ActorDrawEntry) {
	if screen == nil || ctx.World == nil || !ctx.World.MapProperty.IsSiege() {
		return
	}
	for _, entry := range entries {
		if !siegeActorShowsGuildEmblem(entry) {
			continue
		}
		emblem := m.social().ActorGuildEmblem(ctx, entry.Actor, entry.IsPlayer)
		if emblem == nil {
			continue
		}
		bounds := emblem.Bounds()
		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			continue
		}
		centerX, topY := m.actors().SiegeGuildEmblemAnchor(ctx, projection, now, entry)
		x, y := siegeGuildEmblemPosition(centerX, topY, siegeGuildEmblemSize)
		x, y = render.SnapScreenPoint(screen, x, y)
		var opts render.DrawImageOptions
		opts.Filter = render.FilterLinear
		opts.GeoM.Scale(float64(siegeGuildEmblemSize)/float64(bounds.Dx()), float64(siegeGuildEmblemSize)/float64(bounds.Dy()))
		opts.GeoM.Translate(x, y)
		screen.DrawImage(emblem, &opts)
	}
}

func siegeActorShowsGuildEmblem(entry scene.ActorDrawEntry) bool {
	return !scene.ActorHasStealth(entry.Actor) && entry.Actor.GuildID != 0 && entry.Actor.EmblemVersion != 0
}

func siegeGuildEmblemPosition(centerX, topY float64, size int) (float64, float64) {
	if size <= 0 {
		size = siegeGuildEmblemSize
	}
	x := centerX - float64(size)/2
	y := topY - float64(size) - 4
	return x, y
}

func (m *WorldMode) social() *social.System {
	m.socialSystem.Bind(social.Bindings{Windows: &m.ui.social, Console: &m.ui.console, Minimap: &m.ui.minimap})
	return &m.socialSystem
}

func (m *WorldMode) handleChatMessage(ctx client.Context, chat network.ChatMessage, now time.Time) {
	if chat.Announcement {
		text := social.FormatConsoleMessage(ctx.Resources, chat)
		if text == "" {
			return
		}
		m.ui.announcement.Show(text, social.ROClientColor(chat.Color), gameui.AnnouncementStyle{
			Y:        int(chat.FontY),
			FontSize: int(chat.FontSize),
			Bold:     chat.FontType >= 700,
		}, now)
		social.AddConsoleMessage(&m.ui.console, ctx.Resources, chat)
		return
	}
	if m.ui.social.ChatRoom.IsOpen() && !chat.HasColor {
		m.social().AddChatRoomMessage(ctx, chat)
		return
	}
	m.actors().ApplySpeechBubble(ctx, chat, now)
	social.AddConsoleMessage(&m.ui.console, ctx.Resources, chat)
}
