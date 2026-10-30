package ui

import (
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

const (
	guildMemberContextMenuWidth = 128
	guildMemberPermissionExpel  = 0x10
)

type GuildMemberActionKind uint8

const (
	GuildMemberActionNone GuildMemberActionKind = iota
	GuildMemberActionLeave
	GuildMemberActionExpel
	GuildMemberActionWhisper
)

type GuildMemberAction struct {
	Kind   GuildMemberActionKind
	Member session.GuildMember
}

type GuildMemberContextMenu struct {
	Window
	member   session.GuildMember
	canExpel bool
	isSelf   bool
	isMaster bool
	action   GuildMemberAction
}

func (m *GuildMemberContextMenu) Open(ctx Context, x, y int, member session.GuildMember, canExpel, isSelf, isMaster bool) {
	rows := guildMemberContextMenuRows(canExpel, isSelf, isMaster)
	if rows == 0 {
		return
	}
	m.EnsureWindow(guildMemberContextMenuWidth, contextMenuHeight(rows))
	m.titleHeight = 0
	m.ctx = ctx
	m.member = member
	m.canExpel = canExpel
	m.isSelf = isSelf
	m.isMaster = isMaster
	m.action = GuildMemberAction{}
	screenW, screenH := ctx.ScreenSize()
	height := m.height()
	m.SetSize(guildMemberContextMenuWidth, height)
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-guildMemberContextMenuWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	m.OpenAt(x, y, m.widgetTree())
	m.Publish(ctx)
}

func (m *GuildMemberContextMenu) Update(ctx Context) bool {
	m.ctx = ctx
	if !m.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, m.x, m.y, guildMemberContextMenuWidth, m.height())
		if ctx.Input.JustPressed(input.KeyEscape) || (!inside && (ctx.Input.MouseJustPressed(input.MouseButtonLeft) || ctx.Input.MouseJustPressed(input.MouseButtonRight))) {
			m.Close()
			return true
		}
	}
	consumed := m.Window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *GuildMemberContextMenu) Rebind(ctx Context) {
	if !m.IsOpen() {
		return
	}
	m.ctx = ctx
	m.SetContent(m.widgetTree())
	m.Publish(ctx)
}

func (m *GuildMemberContextMenu) PopAction() GuildMemberAction {
	action := m.action
	m.action = GuildMemberAction{}
	return action
}

func (m *GuildMemberContextMenu) widgetTree() widget.Widget {
	rows := make([]widget.Widget, 0, guildMemberContextMenuRows(m.canExpel, m.isSelf, m.isMaster))
	if !m.isSelf {
		rows = append(rows, m.button("Whisper", GuildMemberActionWhisper))
	}
	if m.isSelf && !m.isMaster {
		rows = append(rows, m.button("Leave Guild", GuildMemberActionLeave))
	}
	if m.canExpel && !m.isSelf {
		rows = append(rows, m.button("Expel", GuildMemberActionExpel))
	}
	return contextMenu(guildMemberContextMenuWidth, rows...)
}

func (m *GuildMemberContextMenu) button(label string, kind GuildMemberActionKind) widget.Widget {
	return contextMenuItem(label, func() {
		m.action = GuildMemberAction{Kind: kind, Member: m.member}
		m.Close()
	})
}

func (m *GuildMemberContextMenu) height() int {
	return contextMenuHeight(guildMemberContextMenuRows(m.canExpel, m.isSelf, m.isMaster))
}

func guildMemberContextMenuRows(canExpel, isSelf, isMaster bool) int {
	rows := 0
	if !isSelf {
		rows++
	}
	if isSelf && !isMaster {
		rows++
	}
	if canExpel && !isSelf {
		rows++
	}
	return rows
}
