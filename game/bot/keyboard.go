package bot

import (
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

func (b *Script) KeyPress(code input.KeyCode) error {
	if b == nil {
		return nil
	}
	available := b.KeyboardAvailable
	b.KeyboardAvailable = true
	defer func() { b.KeyboardAvailable = available }()
	return b.invoke("keypress", lua.LString(input.KeyCodeName(code)))
}

func registerLuaKeyboardAPI(state *lua.LState, api *lua.LTable, bot *Script) {
	available := func() bool {
		return bot != nil && bot.KeyboardAvailable && bot.Context.Input != nil
	}
	keyboard := state.NewTable()
	state.SetFuncs(keyboard, map[string]lua.LGFunction{
		"available": func(L *lua.LState) int {
			L.Push(lua.LBool(available()))
			return 1
		},
		"is_down": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.Context.Input.KeyCodeDown(code)))
			return 1
		},
		"was_pressed": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.Context.Input.KeyCodeJustPressed(code)))
			return 1
		},
		"was_released": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.Context.Input.KeyCodeJustReleased(code)))
			return 1
		},
		"consume_press": func(L *lua.LState) int {
			code, ok := input.KeyCodeFromName(L.CheckString(1))
			L.Push(lua.LBool(ok && available() && bot.Context.Input.ConsumeKeyCodePress(code)))
			return 1
		},
		"text": func(L *lua.LState) int {
			if !available() {
				L.Push(lua.LString(""))
			} else {
				L.Push(lua.LString(bot.Context.Input.TextInput()))
			}
			return 1
		},
	})
	api.RawSetString("keyboard", keyboard)
}
