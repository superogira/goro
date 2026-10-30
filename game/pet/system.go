// Package pet manages pet capture, care, commands and presentation.
package pet

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

type Windows struct {
	Egg     gameui.PetEggWindow
	Info    gameui.PetInfoWindow
	Context gameui.PetContextMenu
	Confirm gameui.ConfirmModal
}

type Bindings struct {
	Windows        *Windows
	Console        *gameui.ChatConsole
	Actors         *actors.System
	Skills         *skills.System
	StartAnimation func(client.Context, uint32, int, time.Time, time.Duration)
	Emotion        func(client.Context, network.EmotionNotify)
}

type System struct {
	PendingCapture  CaptureState
	Property        network.PetProperty
	HasProperty     bool
	OldFullness     uint16
	LastTalk        time.Time
	ID              uint32
	SlotMachine     SlotMachineState
	InfoRequested   bool
	slotMachineView *scene.SpriteView
	slotMachineMiss bool
	bindings        Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
