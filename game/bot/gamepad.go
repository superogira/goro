package bot

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	lua "github.com/yuin/gopher-lua"
)

// The script claims controls before the renderer's pointer fallback.
func (b *Script) HandleGamepad(ctx client.Context, dt float64) input.GamepadCapture {
	if !b.Active() || b.Path != ctx.ScriptPath() {
		return input.GamepadCapture{}
	}
	suspended, available := false, false
	if b.bindings.InputState != nil {
		suspended, available = b.bindings.InputState(ctx)
	}
	if suspended {
		return input.GamepadCapture{Buttons: b.BlockedButtons}
	}
	b.Context = ctx
	previous := b.KeyboardAvailable
	b.KeyboardAvailable = available
	b.gamepadInput = true
	b.gamepadCapture = input.GamepadCapture{Buttons: b.BlockedButtons}
	if err := b.invoke("gamepad", lua.LNumber(dt)); err != nil {
		b.Fail("gamepad", err)
	}
	b.gamepadInput = false
	b.KeyboardAvailable = previous
	return b.gamepadCapture
}

func registerLuaGamepadAPI(state *lua.LState, api *lua.LTable, bot *Script) {
	// The same gameplay focus policy as keyboard input: chat, forms, modal
	// dialogs and death suspend controls. Device metadata remains queryable.
	available := func() bool {
		return bot != nil && bot.KeyboardAvailable && bot.Context.Input != nil && bot.Context.Input.GamepadConnected()
	}
	// The early gamepad callback also sees dialog input. Gameplay input() keeps
	// the existing focus filtering; available() still reports gameplay focus.
	readable := func() bool {
		return bot != nil && (bot.KeyboardAvailable || bot.gamepadInput) && bot.Context.Input != nil
	}
	gamepad := state.NewTable()
	buttonQuery := func(query func(input.GamepadButton) bool, needsConnection bool) lua.LGFunction {
		return func(L *lua.LState) int {
			button, valid := input.GamepadButtonFromName(L.CheckString(1))
			allowed := readable()
			if needsConnection {
				allowed = allowed && bot.Context.Input.GamepadConnected()
			}
			L.Push(lua.LBool(valid && allowed && !bot.BlockedButtons[button] && query(button)))
			return 1
		}
	}
	state.SetFuncs(gamepad, map[string]lua.LGFunction{
		"available": func(L *lua.LState) int { L.Push(lua.LBool(available())); return 1 },
		"connected": func(L *lua.LState) int {
			L.Push(lua.LBool(bot.Context.Input != nil && bot.Context.Input.GamepadConnected()))
			return 1
		},
		"name": func(L *lua.LState) int {
			name := ""
			if bot.Context.Input != nil {
				name = bot.Context.Input.GamepadName()
			}
			L.Push(lua.LString(name))
			return 1
		},
		"axis": func(L *lua.LState) int {
			axis, valid := input.GamepadAxisFromName(L.CheckString(1))
			value := 0.0
			if valid && readable() && bot.Context.Input.GamepadConnected() {
				value = bot.Context.Input.GamepadValue(axis)
			}
			L.Push(lua.LNumber(value))
			return 1
		},
		"is_down":      buttonQuery(func(b input.GamepadButton) bool { return bot.Context.Input.GamepadDown(b) }, true),
		"was_pressed":  buttonQuery(func(b input.GamepadButton) bool { return bot.Context.Input.GamepadJustPressed(b) }, true),
		"was_released": buttonQuery(func(b input.GamepadButton) bool { return bot.Context.Input.GamepadJustReleased(b) }, false),
		"consume": func(L *lua.LState) int {
			button, valid := input.GamepadButtonFromName(L.CheckString(1))
			if valid && bot.gamepadInput {
				bot.gamepadCapture.Buttons[button] = true
			}
			return 0
		},
		"consume_pointer": func(L *lua.LState) int {
			if bot.gamepadInput {
				bot.gamepadCapture.Pointer = true
			}
			return 0
		},
	})
	api.RawSetString("gamepad", gamepad)
}
