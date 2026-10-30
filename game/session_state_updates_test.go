package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestGuildPositionPacketsUpdateNamesOnlyWhenPresent(t *testing.T) {
	s := &session.Session{Guild: session.Guild{ID: 1, Positions: []session.GuildPosition{{PositionID: 2, PosName: "Old rank"}}}}
	ctx := client.Context{Session: s}
	m := NewWorldMode()
	for _, name := range []string{"New rank", ""} {
		data := make([]byte, 44)
		binary.LittleEndian.PutUint16(data, network.PacketZCAckGuildPosInfo)
		binary.LittleEndian.PutUint16(data[2:], uint16(len(data)))
		binary.LittleEndian.PutUint32(data[4:], 2)
		copy(data[20:], name)
		m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCAckGuildPosInfo, Data: data}, time.Now())
		if got := s.Guild.Positions[0].PosName; got != name {
			t.Fatalf("rank name = %q, want %q", got, name)
		}
		data = data[:20]
		binary.LittleEndian.PutUint16(data, network.PacketZCGuildPositions)
		binary.LittleEndian.PutUint16(data[2:], uint16(len(data)))
		m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCGuildPositions, Data: data}, time.Now())
		if s.Guild.Positions[0].PosName != name {
			t.Fatal("permissions-only packet cleared rank name")
		}
	}
}

func TestSkillInfoUpdatePreservesExistingTargetMetadata(t *testing.T) {
	sessionState := &session.Session{
		Skills: session.Skills{
			List: []session.Skill{
				{
					ID:       db.SkillWECallpartner,
					Type:     skills.SkillTargetFriend,
					Level:    1,
					MaxLevel: 1,
					SPCost:   1,
					Range:    1,
					Name:     "Romantic Rendeavous!!",
				},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applySkillInfoUpdate(ctx, network.SkillInfoUpdate{Skill: network.SkillInfo{
		ID:     db.SkillWECallpartner,
		Level:  1,
		SPCost: 2,
		Range:  1,
	}})

	got := sessionState.Skills.List[0]
	if got.Type != skills.SkillTargetFriend {
		t.Fatalf("skill type = %d, want existing target type %d", got.Type, skills.SkillTargetFriend)
	}
	if got.Name != "Romantic Rendeavous!!" {
		t.Fatalf("skill name = %q, want existing name", got.Name)
	}
	if got.MaxLevel != 1 {
		t.Fatalf("skill max level = %d, want existing max level", got.MaxLevel)
	}
	if got.SPCost != 2 {
		t.Fatalf("skill sp cost = %d, want updated cost", got.SPCost)
	}
}

func TestGuildSkillUpgradeStaysInGuildSkillList(t *testing.T) {
	for _, loaded := range []bool{false, true} {
		s := &session.Session{Skills: session.Skills{List: []session.Skill{{ID: db.SkillALHeal, Level: 5}}}}
		if loaded {
			s.Guild.Skills = []session.Skill{{ID: db.SkillGdRegeneration, Level: 1, Type: skills.SkillTargetSelf, Name: "Regeneration"}}
		}
		applySkillInfoUpdate(client.Context{Session: s}, network.SkillInfoUpdate{Skill: network.SkillInfo{ID: db.SkillGdRegeneration, Level: 2}})
		if len(s.Skills.List) != 1 || s.Skills.List[0].ID != db.SkillALHeal {
			t.Fatal("guild skill appeared in the character skill list")
		}
		if len(s.Guild.Skills) != 1 || s.Guild.Skills[0].Level != 2 {
			t.Fatalf("guild skill update = %+v", s.Guild.Skills)
		}
		if loaded && (s.Guild.Skills[0].Type != skills.SkillTargetSelf || s.Guild.Skills[0].Name != "Regeneration") {
			t.Fatal("guild upgrade lost targeting metadata")
		}
	}
}
