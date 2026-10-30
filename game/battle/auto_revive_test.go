package battle

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestEscapeMenuAutoReviveSendsRequest(t *testing.T) {
	networkClient, serverConn := testutil.NewConnection(t, 20080910)
	sessionState := &session.Session{Dead: true, Inventory: session.Inventory{Items: []session.InventoryItem{{ItemID: client.TokenOfSiegfriedItemID, Amount: 1}}}}
	ctx := client.Context{Network: networkClient, Session: sessionState, World: worldstate.New()}
	var menu gameui.EscapeMenu

	menu.RequestAutoRevive(ctx)

	testutil.ReadPackets(t, serverConn, network.BuildAutoRevivePacket())
}
