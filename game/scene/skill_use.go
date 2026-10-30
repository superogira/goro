package scene

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func LocalSkillTarget(ctx client.Context) uint32 {
	if ctx.Session == nil {
		return 0
	}
	if ctx.Session.AccountID != 0 {
		return ctx.Session.AccountID
	}
	return ctx.Session.CharID
}

func KnownSkillMaxLevel(skill session.Skill) int {
	if maxLevel, ok := db.SkillMaxLevel(skill.ID); ok {
		return maxLevel
	}
	if skill.MaxLevel > 0 {
		return skill.MaxLevel
	}
	return 0
}

func NormalizeSessionSkillLevelCap(skill session.Skill) session.Skill {
	maxLevel := KnownSkillMaxLevel(skill)
	if maxLevel <= 0 {
		return skill
	}
	skill.MaxLevel = maxLevel
	if skill.Level > maxLevel {
		skill.Level = maxLevel
	}
	return skill
}

func SessionSkillFromNetwork(skill network.SkillInfo) session.Skill {
	return NormalizeSessionSkillLevelCap(session.Skill{
		ID:         skill.ID,
		Type:       skill.Type,
		Level:      skill.Level,
		SPCost:     skill.SPCost,
		Range:      skill.Range,
		Name:       skill.Name,
		Upgradable: skill.Upgradable,
	})
}

func SessionSkillFromNetworkWithResources(manager *res.Manager, skill network.SkillInfo) session.Skill {
	out := SessionSkillFromNetwork(skill)
	if manager != nil {
		if maxLevel, ok := manager.SkillMaxLevel(int(skill.ID)); ok {
			out.MaxLevel = maxLevel
		}
	}
	return NormalizeSessionSkillLevelCap(out)
}

func SessionSkillLevel(s *session.Session, skillID uint16) int {
	if s == nil {
		return 0
	}
	for _, skill := range s.Skills.List {
		if skill.ID == skillID {
			return skill.Level
		}
	}
	return 0
}
