package social

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func (m *System) SendTradeRequest(ctx client.Context, actorID uint32, name string) {
	if actorID == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("trade request failed target=%d name=%q: not connected", actorID, name)
		return
	}
	if err := ctx.Network.SendTradeRequest(actorID); err != nil {
		glog.Warnf("trade request failed target=%d name=%q: %v", actorID, name, err)
		return
	}
	m.PendingTradeName = strings.TrimSpace(name)
}

func (m *System) OpenTradeRequest(ctx client.Context, request network.TradeRequest) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Player"
	}
	message := fmt.Sprintf("%s wants to trade with you.", name)
	if request.TargetID != 0 || request.Level != 0 {
		message = fmt.Sprintf("%s\nLv.%d", message, request.Level)
	}
	m.bindings.Windows.TradeRequest.Open(ctx, "Trade Request", message, func() {
		if ctx.Network == nil {
			glog.Warnf("trade request accept failed: not connected")
			return
		}
		if err := ctx.Network.SendTradeAck(true); err != nil {
			glog.Warnf("trade request accept failed name=%q: %v", request.Name, err)
			return
		}
		m.PendingTradeName = name
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("trade request reject failed: not connected")
			return
		}
		if err := ctx.Network.SendTradeAck(false); err != nil {
			glog.Warnf("trade request reject failed name=%q: %v", request.Name, err)
		}
	})
}

func (m *System) HandleTradeResponse(ctx client.Context, response network.TradeResponse) {
	name := m.PendingTradeName
	m.PendingTradeName = ""
	switch response.Result {
	case 0:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("That character is too far away.")
	case 1:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("Character does not exist.")
	case 2:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("Trade failed.")
	case 3:
		if name == "" {
			name = "Player"
		}
		m.bindings.Windows.TradeWindow.Open(ctx, name)
	case 4:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("Trade canceled.")
	case 5:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("That character is busy.")
	default:
		m.bindings.Windows.TradeWindow.Close(ctx)
		m.bindings.Console.AddErrorMessage("Trade failed.")
	}
}

func (m *System) HandleTradeExec(ctx client.Context, exec network.TradeExec) {
	m.bindings.Windows.TradeWindow.Close(ctx)
	if exec.Result == 0 {
		m.bindings.Console.AddBlueMessage("Trade completed.")
		return
	}
	m.bindings.Console.AddErrorMessage("Trade failed.")
}
