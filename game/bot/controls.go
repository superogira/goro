package bot

import (
	"github.com/kivutar/goro/game/scene"
	lua "github.com/yuin/gopher-lua"
)

func (b *Script) registerControlsAPI(api *lua.LTable) {
	world := b.bindings.World
	allowed := func() bool {
		if world == nil {
			return false
		}
		suspended, available := b.bindings.InputState(b.Context)
		return !suspended && available
	}
	b.state.SetFuncs(api, map[string]lua.LGFunction{
		"use_shortcut": func(L *lua.LState) int {
			ctx := b.Context
			slot := L.CheckInt(1)
			var skillID uint16
			var used bool
			if allowed() {
				skillID, used = world.UseShortcut(ctx, slot)
			}
			L.Push(lua.LBool(used))
			L.Push(lua.LNumber(skillID))
			return 2
		},
		"rotate_camera": func(L *lua.LState) int {
			ctx := b.Context
			yaw, pitch := float64(L.CheckNumber(1)), float64(L.CheckNumber(2))
			if allowed() && scene.IsFinite(yaw) && scene.IsFinite(pitch) {
				world.RotateCamera(ctx, yaw, pitch)
			}
			return 0
		},
		"zoom_camera": func(L *lua.LState) int {
			ctx := b.Context
			delta := float64(L.CheckNumber(1))
			if allowed() && scene.IsFinite(delta) {
				world.ZoomCamera(ctx, delta)
			}
			return 0
		},
		"camera_yaw": func(L *lua.LState) int {
			ctx := b.Context
			if world == nil {
				L.Push(lua.LNumber(0))
				return 1
			}
			L.Push(lua.LNumber(world.CameraYaw(ctx)))
			return 1
		},
		"cancel_skill": func(L *lua.LState) int {
			if allowed() {
				world.CancelSkill()
			}
			return 0
		},
		"npc_dialog": func(L *lua.LState) int {
			ctx := b.Context
			open := world != nil && b.bindings.ControlUI(ctx, "")
			if open && L.GetTop() != 0 {
				b.bindings.ControlUI(ctx, L.CheckString(1))
			}
			L.Push(lua.LBool(open))
			return 1
		},
		"pointer_over_ui": func(L *lua.LState) int {
			ctx := b.Context
			blocked := false
			if ui, ok := ctx.UIManager.(interface{ PointerBlocked(int, int) bool }); ok && ctx.Input != nil {
				blocked = ui.PointerBlocked(ctx.Input.MouseX, ctx.Input.MouseY)
			}
			L.Push(lua.LBool(blocked))
			return 1
		},
	})
}
