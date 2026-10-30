package npc

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func (m *System) ApplyCutin(ctx client.Context, cutin network.NPCCutin) error {
	if ctx.Config.Headless {
		return nil
	}
	if cutin.Position == network.NPCCutinClear || cutin.Image == "" || cutin.Position > network.NPCCutinWindowless {
		m.bindings.Windows.Cutin.Apply(cutin, nil)
		return nil
	}
	texture, err := m.cutinTexture(ctx.Resources, cutin.Image)
	m.bindings.Windows.Cutin.Apply(cutin, texture)
	return err
}

func (m *System) cutinTexture(manager *res.Manager, resource string) (*render.Image, error) {
	resource = strings.TrimSpace(resource)
	if manager == nil || resource == "" {
		return nil, fmt.Errorf("cut-in resource manager unavailable")
	}
	candidates := res.NPCCutinTextureCandidates(resource)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("invalid cut-in resource %q", resource)
	}
	key := "__npc_cutin_" + strings.ToLower(strings.ReplaceAll(candidates[0], "\\", "/"))
	if m.bindings.Map.Textures == nil {
		m.bindings.Map.Textures = make(map[string]*render.Image)
	}
	if m.bindings.Map.TextureMiss == nil {
		m.bindings.Map.TextureMiss = make(map[string]struct{})
	}
	if texture := m.bindings.Map.Textures[key]; texture != nil {
		return texture, nil
	}
	if _, missed := m.bindings.Map.TextureMiss[key]; missed {
		return nil, nil
	}
	img, _, err := res.LoadImage(manager, candidates)
	if err != nil {
		m.bindings.Map.TextureMiss[key] = struct{}{}
		return nil, err
	}
	texture := render.NewImageFromImage(img)
	m.bindings.Map.Textures[key] = texture
	return texture, nil
}

func (m *System) CutinPointerBlocked(ctx client.Context) bool {
	if ctx.Config.Render.NoUI || ctx.Input == nil {
		return false
	}
	width, height := ctx.ScreenSize()
	return m.bindings.Windows.Cutin.PointerBlocked(width, height, ctx.Input.MouseX, ctx.Input.MouseY)
}
