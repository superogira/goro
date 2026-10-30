package social

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
)

func TestTradeAcceptancePreservesPartnerName(t *testing.T) {
	for _, tc := range []struct {
		name     string
		incoming bool
		partner  string
		want     string
	}{
		{name: "incoming", incoming: true, partner: " Zambla ", want: "Zambla"},
		{name: "outgoing", partner: " Kivy ", want: "Kivy"},
		{name: "unnamed incoming", incoming: true, want: "Player"},
		{name: "unnamed outgoing", want: "Player"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			netClient, server := testutil.NewConnection(t, 20080910)
			ctx := client.Context{Network: netClient, ScreenW: 1024, ScreenH: 768}
			mode := newTestSystem()
			if tc.incoming {
				mode.OpenTradeRequest(ctx, network.TradeRequest{Name: tc.partner})
				if !mode.bindings.Windows.TradeRequest.IsOpen() || mode.bindings.Windows.TradeWindow.IsOpen() {
					t.Fatal("incoming request should only open the confirmation")
				}
				mode.bindings.Windows.TradeRequest.Confirm(ctx)
				testutil.ReadPackets(t, server, network.BuildTradeAckPacket(true))
				if mode.bindings.Windows.TradeRequest.IsOpen() {
					t.Fatal("accepted request left its confirmation open")
				}
			} else {
				mode.SendTradeRequest(ctx, 200, tc.partner)
				testutil.ReadPackets(t, server, network.BuildTradeRequestPacket(200))
			}
			if mode.bindings.Windows.TradeWindow.IsOpen() {
				t.Fatal("trade opened before server acceptance")
			}

			mode.HandleTradeResponse(ctx, network.TradeResponse{Result: 3})
			if !mode.bindings.Windows.TradeWindow.IsOpen() {
				t.Fatal("server acceptance did not open trade")
			}
			if text := testutil.DrawnWidgetText(mode.bindings.Windows.TradeWindow.Widget()); !strings.Contains(text, "Trade with "+tc.want) {
				t.Fatalf("trade window text = %q, want partner %q", text, tc.want)
			}
			if mode.PendingTradeName != "" {
				t.Fatalf("accepted trade retained pending name %q", mode.PendingTradeName)
			}
		})
	}
}

func TestTradeRejectionDoesNotOpenTrade(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient}
	mode := newTestSystem()
	mode.OpenTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
	mode.bindings.Windows.TradeRequest.Cancel(ctx)
	testutil.ReadPackets(t, server, network.BuildTradeAckPacket(false))
	if mode.bindings.Windows.TradeRequest.IsOpen() || mode.bindings.Windows.TradeWindow.IsOpen() || mode.PendingTradeName != "" {
		t.Fatal("rejected request left an open window or a pending trade")
	}
}

func TestTradeFailureClearsPendingName(t *testing.T) {
	for _, result := range []uint8{0, 1, 2, 4, 5, 255} {
		t.Run(fmt.Sprintf("result_%d", result), func(t *testing.T) {
			netClient, server := testutil.NewConnection(t, 20080910)
			ctx := client.Context{Network: netClient}
			mode := newTestSystem()
			mode.OpenTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
			mode.bindings.Windows.TradeRequest.Confirm(ctx)
			testutil.ReadPackets(t, server, network.BuildTradeAckPacket(true))
			mode.HandleTradeResponse(ctx, network.TradeResponse{Result: result})
			if mode.bindings.Windows.TradeWindow.IsOpen() || mode.PendingTradeName != "" {
				t.Fatal("failed trade left an open window or a pending name")
			}
		})
	}
}

func TestTradeSendFailureDoesNotOpenTrade(t *testing.T) {
	for _, netClient := range []*network.Client{nil, network.NewClient(20080910, false)} {
		ctx := client.Context{Network: netClient}
		mode := newTestSystem()
		mode.OpenTradeRequest(ctx, network.TradeRequest{Name: "Zambla"})
		mode.bindings.Windows.TradeRequest.Confirm(ctx)
		if mode.bindings.Windows.TradeWindow.IsOpen() || mode.PendingTradeName != "" {
			t.Fatal("failed acknowledgement opened a trade or retained its name")
		}
		mode.SendTradeRequest(ctx, 200, "Kivy")
		if mode.bindings.Windows.TradeWindow.IsOpen() || mode.PendingTradeName != "" {
			t.Fatal("failed request opened a trade or retained its name")
		}
	}
}
