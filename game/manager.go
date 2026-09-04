package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
)

type Mode interface {
	Name() string
	Enter(client.Context)
	Update(client.Context) (Mode, error)
	Draw(client.Context, *render.Frame)
}

type overlayMode interface {
	DrawOverlay(client.Context, *render.Frame)
}

type uiOverlayMode interface {
	DrawUIOverlay(client.Context, *render.Frame)
}

type frameSubmittedMode interface {
	FrameSubmitted()
}

type Manager struct {
	ctx  client.Context
	mode Mode
}

func NewManager(ctx client.Context, mode Mode) *Manager {
	m := &Manager{ctx: ctx, mode: mode}
	if m.mode != nil {
		m.mode.Enter(ctx)
	}
	return m
}

func (m *Manager) UpdateContext(ctx client.Context) {
	m.ctx = ctx
}

// ModeName returns the name of the active mode ("login", "world", ...).
// Empty when no mode is active.
func (m *Manager) ModeName() string {
	if m.mode == nil {
		return ""
	}
	return m.mode.Name()
}

func (m *Manager) Update() error {
	if m.mode == nil {
		return nil
	}

	next, err := m.mode.Update(m.ctx)
	if err != nil {
		return err
	}
	if next != nil {
		m.mode = next
		m.mode.Enter(m.ctx)
	}
	return nil
}

func (m *Manager) Draw(screen *render.Frame) {
	if m.mode != nil {
		m.mode.Draw(m.ctx, screen)
	}
}

func (m *Manager) DrawOverlay(screen *render.Frame) {
	if mode, ok := m.mode.(overlayMode); ok {
		mode.DrawOverlay(m.ctx, screen)
	}
}

func (m *Manager) DrawUIOverlay(screen *render.Frame) {
	if mode, ok := m.mode.(uiOverlayMode); ok {
		mode.DrawUIOverlay(m.ctx, screen)
	}
}

func (m *Manager) FrameSubmitted() {
	if mode, ok := m.mode.(frameSubmittedMode); ok {
		mode.FrameSubmitted()
	}
}
