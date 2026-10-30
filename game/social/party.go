package social

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func actorIsPartyMember(s *session.Session, actor worldstate.Actor, actorName string) bool {
	if s == nil || actor.ID == 0 {
		return false
	}
	for _, member := range s.Party.Members {
		if member.AccountID == actor.ID {
			return true
		}
		if actorName != "" && strings.EqualFold(scene.SanitizeActorName(member.Name), actorName) {
			return true
		}
	}
	return false
}

func (m *System) SendPartyInvite(ctx client.Context, actorID uint32, name string) {
	name = strings.TrimSpace(name)
	if actorID == 0 && name == "" {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("party invite failed target=%d name=%q: not connected", actorID, name)
		m.bindings.Console.AddErrorMessage("Party invitation failed: not connected.")
		return
	}
	if err := ctx.Network.SendPartyInvite(actorID, name); err != nil {
		glog.Warnf("party invite failed target=%d name=%q: %v", actorID, name, err)
		m.bindings.Console.AddErrorMessage("Party invitation failed.")
		return
	}
	m.bindings.Console.AddBlueMessage("%s has received an invitation to join your party.", partyDisplayName(name))
}

func (m *System) UpdatePartyHelperWindows(ctx client.Context) bool {
	createConsumed := m.bindings.Windows.PartyCreate.Update(ctx)
	if action := m.bindings.Windows.PartyCreate.PopAction(); action.Name != "" {
		m.CreatePartyFromWindow(ctx, action)
		return true
	}
	if createConsumed {
		return true
	}
	inviteConsumed := m.bindings.Windows.PartyInvite.Update(ctx)
	if name := m.bindings.Windows.PartyInvite.PopAction(); name != "" {
		m.SendPartyInvite(ctx, 0, name)
		return true
	}
	return inviteConsumed
}

func (m *System) CreatePartyFromWindow(ctx client.Context, action gameui.PartyCreateWindowAction) {
	name := strings.TrimSpace(action.Name)
	if name == "" {
		return
	}
	if ctx.Session != nil && ctx.Session.Party.Active() {
		m.bindings.Console.AddErrorMessage("You are already in a party.")
		return
	}
	if ctx.Network == nil {
		m.bindings.Console.AddErrorMessage("Create party failed: not connected.")
		return
	}
	if err := ctx.Network.SendMakeParty2(name, action.ItemPickup, action.ItemDivision); err != nil {
		m.bindings.Console.AddErrorMessage("Create party failed.")
		glog.Warnf("party create failed name=%q pickup=%d division=%d: %v", name, action.ItemPickup, action.ItemDivision, err)
		return
	}
}

func (m *System) openPartyMemberInfo(ctx client.Context, member session.PartyMember) {
	name := partyDisplayName(member.Name)
	mapName := strings.TrimSpace(member.MapName)
	if mapName == "" {
		mapName = "Unknown map"
	}
	state := "Offline"
	if member.Online() {
		state = "Online"
	}
	m.bindings.Windows.PartyInfo.OpenAlert(ctx, "Party Member", fmt.Sprintf("%s\n%s, %s", name, mapName, state), nil)
}

func (m *System) openExpelPartyMemberConfirm(ctx client.Context, member session.PartyMember) {
	name := partyDisplayName(member.Name)
	m.bindings.Windows.PartyInfo.Open(ctx, "Expel Party Member", fmt.Sprintf("Expel %s from the party?", name), func() {
		if ctx.Network == nil {
			m.bindings.Console.AddErrorMessage("Expel party member failed: not connected.")
			return
		}
		if err := ctx.Network.SendExpelPartyMember(member.AccountID, member.Name); err != nil {
			m.bindings.Console.AddErrorMessage("Expel party member failed.")
			glog.Warnf("party expel failed aid=%d name=%q: %v", member.AccountID, member.Name, err)
			return
		}
		m.bindings.Console.AddSystemMessage("Expel request sent for %s.", name)
	}, nil)
}

func (m *System) OpenPartyInviteRequest(ctx client.Context, request network.PartyInviteRequest) {
	name := partyDisplayName(request.Name)
	m.bindings.Windows.PartyRequest.Open(ctx, "Party Invitation", fmt.Sprintf("%s has invited you to join a party.", name), func() {
		if ctx.Network == nil {
			glog.Warnf("party invite accept failed: not connected")
			return
		}
		if err := ctx.Network.SendPartyInviteAck(request.RequestID, true); err != nil {
			glog.Warnf("party invite accept failed request=%d name=%q: %v", request.RequestID, request.Name, err)
		}
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("party invite reject failed: not connected")
			return
		}
		if err := ctx.Network.SendPartyInviteAck(request.RequestID, false); err != nil {
			glog.Warnf("party invite reject failed request=%d name=%q: %v", request.RequestID, request.Name, err)
		}
	})
}

func (m *System) HandlePartyCreateResult(ctx client.Context, result network.PartyCreateResult) {
	switch result.Result {
	case 0:
		m.bindings.Console.AddBlueMessage("Party created.")
		SyncLocalPartyVitals(ctx)
	case 1:
		m.bindings.Console.AddErrorMessage("Party name already exists.")
	case 2:
		m.bindings.Console.AddErrorMessage("You are already in a party.")
	case 3:
		m.bindings.Console.AddErrorMessage("Cannot organize a party on this map.")
	default:
		// rAthena documents unknown ZC_ACK_MAKE_GROUP results as "nothing".
	}
}

func (m *System) HandlePartyInviteAnswer(answer network.PartyInviteAnswer) {
	name := partyDisplayName(answer.Name)
	switch answer.Answer {
	case 0:
		m.bindings.Console.AddErrorMessage("%s is already in a party.", name)
	case 1:
		m.bindings.Console.AddErrorMessage("%s denied your party invitation.", name)
	case 2:
		m.bindings.Console.AddBlueMessage("%s joined your party.", name)
	case 3:
		m.bindings.Console.AddErrorMessage("The party is full.")
	case 4:
		m.bindings.Console.AddErrorMessage("A character from the same account is already in the party.")
	case 5:
		m.bindings.Console.AddErrorMessage("%s blocked party invitations.", name)
	case 7:
		m.bindings.Console.AddErrorMessage("%s is not online.", name)
	case 8:
		m.bindings.Console.AddErrorMessage("Cannot invite players on this map.")
	case 9:
		m.bindings.Console.AddErrorMessage("Cannot join a party on this map.")
	default:
		m.bindings.Console.AddErrorMessage("Party invitation failed.")
	}
}

func ApplyPartyList(ctx client.Context, list network.PartyList) {
	if ctx.Session == nil {
		return
	}
	old := partyMembersByAccount(ctx.Session.Party.Members)
	ctx.Session.Party.Name = list.Name
	ctx.Session.Party.Members = ctx.Session.Party.Members[:0]
	for _, member := range list.Members {
		next := sessionPartyMemberFromNetwork(member)
		if prev, ok := old[next.AccountID]; ok {
			mergePartyMemberVitals(&next, prev)
		}
		ctx.Session.Party.Members = append(ctx.Session.Party.Members, next)
	}
	SyncLocalPartyVitals(ctx)
	glog.Debugf("party list received name=%q members=%d", list.Name, len(list.Members))
}

func ApplyPartyMemberJoin(ctx client.Context, member network.PartyMember) {
	if ctx.Session == nil {
		return
	}
	next := sessionPartyMemberFromNetwork(member)
	if member.AccountID == ctx.Session.AccountID {
		ctx.Session.Party.Name = member.GroupName
	}
	if next.Name == "" {
		next.Name = "Player"
	}
	if next.MapName == "" && ctx.World != nil {
		next.MapName = ctx.World.MapName
	}
	upsertPartyMember(&ctx.Session.Party, next)
	SyncLocalPartyVitals(ctx)
	glog.Debugf("party member join aid=%d name=%q map=%q", next.AccountID, next.Name, next.MapName)
}

func ApplyPartyMemberLeave(ctx client.Context, left network.PartyMemberLeave) bool {
	if ctx.Session == nil {
		return false
	}
	switch left.Result {
	case 0, 1:
	case 2:
		return false
	case 3:
		return false
	}
	if left.AccountID == ctx.Session.AccountID {
		ctx.Session.Party = session.Party{}
		return true
	}
	for i := range ctx.Session.Party.Members {
		member := ctx.Session.Party.Members[i]
		if member.AccountID != left.AccountID && (left.Name == "" || member.Name != left.Name) {
			continue
		}
		ctx.Session.Party.Members = append(ctx.Session.Party.Members[:i], ctx.Session.Party.Members[i+1:]...)
		return true
	}
	return true
}

func ApplyPartyOption(ctx client.Context, option network.PartyOption) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Party.ExpShare = option.ExpOption
	glog.Debugf("party option exp=%d", option.ExpOption)
}

func ApplyPartyInviteConfig(ctx client.Context, config network.PartyInviteConfig) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Party.RefuseInvites = config.RefuseInvites
	glog.Debugf("party invite config refuse_invites=%t", config.RefuseInvites)
}

func ApplyPartyMemberHP(ctx client.Context, hp network.PartyMemberHP) {
	if ctx.Session == nil || !ctx.Session.Party.Active() {
		return
	}
	member := findPartyMember(&ctx.Session.Party, hp.AccountID)
	if member == nil {
		return
	}
	member.HP = hp.HP
	member.MaxHP = hp.MaxHP
}

func ApplyPartyMemberPosition(ctx client.Context, pos network.PartyMemberPosition) {
	if ctx.Session == nil || !ctx.Session.Party.Active() {
		return
	}
	member := findPartyMember(&ctx.Session.Party, pos.AccountID)
	if member == nil {
		return
	}
	member.X = pos.X
	member.Y = pos.Y
}

func ApplyPartyChat(ctx client.Context, chat network.PartyChat, console *gameui.ChatConsole) {
	if console == nil {
		return
	}
	msg := strings.TrimSpace(chat.Message)
	if msg == "" {
		return
	}
	console.AddBlueMessage("%s", msg)
}

func (m *System) HandlePartyItemPickup(ctx client.Context, pickup network.PartyItemPickup) {
	name := "Party member"
	if ctx.Session != nil {
		if member := findPartyMember(&ctx.Session.Party, pickup.AccountID); member != nil && member.Name != "" {
			name = member.Name
		}
	}
	itemName := gameui.ItemDisplayName(ctx.Resources, session.InventoryItem{
		ItemID: pickup.ItemID, Identified: pickup.Identified, Type: pickup.Type,
		Refine: pickup.Refine, Cards: pickup.Cards,
	})
	template := "%s has acquired %s."
	if ctx.Resources != nil {
		if text, ok := ctx.Resources.MsgString(1276); ok && text != "" {
			template = text
		}
	}
	m.bindings.Console.AddBlueMessage(template, name, itemName)
}

func upsertPartyMember(p *session.Party, member session.PartyMember) {
	for i := range p.Members {
		if p.Members[i].AccountID != member.AccountID {
			continue
		}
		mergePartyMemberVitals(&member, p.Members[i])
		p.Members[i] = member
		return
	}
	p.Members = append(p.Members, member)
}

func ensurePartyMember(p *session.Party, accountID uint32) *session.PartyMember {
	if member := findPartyMember(p, accountID); member != nil {
		return member
	}
	p.Members = append(p.Members, session.PartyMember{AccountID: accountID, Name: "Player"})
	return &p.Members[len(p.Members)-1]
}

func findPartyMember(p *session.Party, accountID uint32) *session.PartyMember {
	for i := range p.Members {
		if p.Members[i].AccountID == accountID {
			return &p.Members[i]
		}
	}
	return nil
}

func sessionPartyMemberFromNetwork(member network.PartyMember) session.PartyMember {
	return session.PartyMember{
		AccountID: member.AccountID,
		Name:      member.Name,
		MapName:   member.MapName,
		Role:      member.Role,
		State:     member.State,
		X:         member.X,
		Y:         member.Y,
	}
}

func partyMembersByAccount(members []session.PartyMember) map[uint32]session.PartyMember {
	out := make(map[uint32]session.PartyMember, len(members))
	for _, member := range members {
		out[member.AccountID] = member
	}
	return out
}

func mergePartyMemberVitals(dst *session.PartyMember, src session.PartyMember) {
	if dst.HP == 0 {
		dst.HP = src.HP
	}
	if dst.MaxHP == 0 {
		dst.MaxHP = src.MaxHP
	}
	if dst.X == 0 {
		dst.X = src.X
	}
	if dst.Y == 0 {
		dst.Y = src.Y
	}
	dst.Dead = src.Dead
}

func partyDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Player"
	}
	return name
}

func SyncLocalPartyVitals(ctx client.Context) {
	if ctx.Session == nil || !ctx.Session.Party.Active() || ctx.Session.AccountID == 0 {
		return
	}
	member := ensurePartyMember(&ctx.Session.Party, ctx.Session.AccountID)
	if member.Name == "" {
		member.Name = partyDisplayName(ctx.Session.Selected.Name)
	}
	hp, maxHP := localPartyHP(ctx.Session)
	member.HP = hp
	member.MaxHP = maxHP
}

func localPartyHP(s *session.Session) (int, int) {
	if s == nil {
		return 0, 0
	}
	hp := s.Vitals.HP
	maxHP := s.Vitals.MaxHP
	if maxHP <= 0 {
		character := s.SelectedCharacter()
		if character.MaxHP <= 0 && s.Selected.MaxHP > 0 {
			character = s.Selected
		}
		hp = int(character.HP)
		maxHP = int(character.MaxHP)
	}
	return hp, maxHP
}

func partyCanManage(s *session.Session) bool {
	if s == nil || !s.Party.Active() {
		return false
	}
	if len(s.Party.Members) == 0 {
		return true
	}
	for _, member := range s.Party.Members {
		if member.AccountID == s.AccountID {
			return member.Leader()
		}
	}
	return false
}
