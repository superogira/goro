// Package npc manages NPC dialogs, shops, illustrations and script progress.
package npc

import (
	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	gameui "github.com/kivutar/goro/ui"
)

type Windows struct {
	Dialog          gameui.NPCDialog
	Cutin           gameui.NPCCutinOverlay
	Shop            gameui.ShopWindow
	StoragePassword gameui.StoragePasswordWindow
}

func (w *Windows) BindDialogLifecycle() {
	w.Dialog.SetCloseHandler(w.Cutin.Clear)
}

type Bindings struct {
	Windows        *Windows
	Storage        *gameui.StorageWindow
	Map            *maps.System
	PointerBlocked func(client.Context) bool
}

type System struct {
	Progress     ProgressState
	ShowDigit    ShowDigitState
	TimeFontView *scene.SpriteView
	TimeFontMiss bool
	bindings     Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
