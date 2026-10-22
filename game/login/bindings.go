package login

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

// Bindings connect the login flow to world state and character rendering.
type Bindings struct {
	Console          *gameui.ChatConsole
	ApplyWorldPacket func(client.Context, network.Packet) (bool, error)
	EnterMap         func(client.Context, network.MapAcceptEnter)
	ChangeMap        func(client.Context, network.MapChange)
	Preview          func(client.Context, session.Character, int) *scene.Billboard
}
