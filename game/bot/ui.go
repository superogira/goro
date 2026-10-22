package bot

import (
	lua "github.com/yuin/gopher-lua"
)

func registerLuaUIAPI(L *lua.LState, api *lua.LTable, script *Script) {
	ui := L.NewTable()
	control := func(action string) bool {
		return script.bindings.ControlUI != nil && script.bindings.ControlUI(script.Context, action)
	}
	L.SetFuncs(ui, map[string]lua.LGFunction{
		"active":  func(L *lua.LState) int { L.Push(lua.LBool(control(""))); return 1 },
		"control": func(L *lua.LState) int { L.Push(lua.LBool(control(L.CheckString(1)))); return 1 },
	})
	api.RawSetString("ui", ui)
	api.RawSetString("in_game", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(script.InGame()))
		return 1
	}))
}
