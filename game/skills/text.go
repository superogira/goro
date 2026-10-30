package skills

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
)

func (m *System) OpenSkillTextPrompt(ctx client.Context, skill session.Skill, x, y int, source string) {
	m.PendingText = PendingTextTarget{Skill: skill, X: x, Y: y, Source: source}
	m.bindings.TextPrompt.Open(ctx, skillTextPromptTitle(skill), "Message", "Message", skillGroundTextMaxBytes)
}

func (m *System) UpdateSkillTextPrompt(ctx client.Context) bool {
	consumed := m.bindings.TextPrompt.Update(ctx)
	action := m.bindings.TextPrompt.PopAction()
	if action.Submitted {
		m.SendPendingSkillText(ctx, action.Text)
		return true
	}
	if m.PendingText.Skill.ID != 0 && !m.bindings.TextPrompt.IsOpen() {
		glog.Debugf("skill text prompt canceled skill=%d target=%d,%d", m.PendingText.Skill.ID, m.PendingText.X, m.PendingText.Y)
		m.PendingText = PendingTextTarget{}
	}
	return consumed
}

func (m *System) SendPendingSkillText(ctx client.Context, text string) {
	pending := m.PendingText
	m.PendingText = PendingTextTarget{}
	if pending.Skill.ID == 0 {
		return
	}
	if err := m.UseGround(ctx, pending.Skill, pending.X, pending.Y, text, pending.Source); err != nil {
		m.bindings.Error("%s failed.", SkillLabel(pending.Skill))
		glog.Warnf("skill text send failed skill=%d target=%d,%d: %v", pending.Skill.ID, pending.X, pending.Y, err)
		return
	}
	glog.Debugf("skill text requested skill=%d target=%d,%d", pending.Skill.ID, pending.X, pending.Y)
}

func skillTextPromptTitle(skill session.Skill) string {
	return fmt.Sprintf("%s Message", SkillLabel(skill))
}
