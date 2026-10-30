package npc

import (
	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	gameui "github.com/kivutar/goro/ui"
)

func newTestSystem() *System {
	s := &System{}
	windows := &Windows{}
	windows.BindDialogLifecycle()
	s.Bind(Bindings{Windows: windows, Storage: &gameui.StorageWindow{}, Map: &maps.System{}, PointerBlocked: func(client.Context) bool { return false }})
	return s
}
