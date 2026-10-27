package bot

import (
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/scripts"
	lua "github.com/yuin/gopher-lua"
)

const TickInterval = 150 * time.Millisecond

// Bindings connects the script to the active mode without owning game state.
type Bindings struct {
	World          *World
	InputState     func(client.Context) (suspended, keyboardAvailable bool)
	ControlUI      func(client.Context, string) bool
	ClearHighlight func()
}

// The manager owns one script for the active mode. WorldMode borrows it to run
// gameplay callbacks at the appropriate points in its update.
type Script struct {
	Path              string
	state             *lua.LState
	Context           client.Context
	bindings          Bindings
	NextTick          time.Time
	Disabled          bool
	KeyboardAvailable bool
	gamepadInput      bool
	gamepadCapture    input.GamepadCapture
	BlockedButtons    [input.GamepadButtonCount]bool
}

func New(ctx client.Context, path string, bindings Bindings) (*Script, error) {
	b := &Script{
		Path: path, state: lua.NewState(), Context: ctx, bindings: bindings,
		NextTick: time.Now().Add(TickInterval),
	}
	api := b.state.NewTable()
	b.registerAPI(api)
	registerLuaKeyboardAPI(b.state, api, b)
	registerLuaGamepadAPI(b.state, api, b)
	registerLuaUIAPI(b.state, api, b)
	b.state.SetGlobal("goro", api)
	var err error
	if name, builtin := strings.CutPrefix(path, "builtin:"); builtin {
		var source []byte
		source, err = scripts.Builtin.ReadFile(name + ".lua")
		if err == nil {
			err = b.state.DoString(string(source))
		}
	} else {
		err = b.state.DoFile(path)
	}
	if err != nil {
		b.Close()
		return nil, err
	}
	return b, nil
}

func (b *Script) InGame() bool { return b.bindings.World != nil }

func (b *Script) worldContext() client.Context {
	if !b.InGame() {
		return client.Context{}
	}
	return b.Context
}

func (b *Script) Close() {
	if b == nil {
		return
	}
	if b.bindings.ClearHighlight != nil {
		b.bindings.ClearHighlight()
	}
	if b.state != nil {
		b.state.Close()
		b.state = nil
	}
}

func (b *Script) Fail(callback string, err error) {
	glog.Warnf("lua script %s failed path=%q: %v", callback, b.Path, err)
	b.Close()
	b.Disabled = true
}

func (b *Script) invoke(name string, args ...lua.LValue) error {
	if !b.Active() {
		return nil
	}
	fn := b.state.GetGlobal(name)
	if fn == lua.LNil {
		return nil
	}
	return b.state.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}, args...)
}

func (b *Script) Tick() error { return b.invoke("tick") }

func (b *Script) InputFrame(available bool) error {
	b.KeyboardAvailable = available
	return b.invoke("input")
}

func (b *Script) Active() bool {
	return b != nil && !b.Disabled && b.state != nil
}
