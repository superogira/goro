package game

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

var (
	expNotifyColor      = color.RGBA{R: 255, G: 255, B: 99, A: 255}
	questExpNotifyColor = color.RGBA{R: 164, G: 66, B: 220, A: 255}
)

func addExpNotifyMessage(console *gameui.ChatConsole, manager *res.Manager, notify network.ExpNotify) {
	if console == nil {
		return
	}
	text, messageColor := formatExpNotifyMessage(manager, notify)
	if text == "" {
		return
	}
	console.AddColoredMessage(messageColor, "%s", text)
}

func formatExpNotifyMessage(manager *res.Manager, notify network.ExpNotify) (string, color.RGBA) {
	switch notify.ExpType {
	case 0:
		switch notify.VarID {
		case network.StatusBaseExp:
			return formatExpNotifyTemplate(manager, 1613, "Gained %d base experience", notify.Amount), expNotifyColor
		case network.StatusJobExp:
			return formatExpNotifyTemplate(manager, 1614, "Gained %d job experience", notify.Amount), expNotifyColor
		default:
			return "", color.RGBA{}
		}
	case 1:
		switch notify.VarID {
		case network.StatusBaseExp:
			return fmt.Sprintf("Experience gained from Quest, Base:%d", notify.Amount), questExpNotifyColor
		case network.StatusJobExp:
			return fmt.Sprintf("Experience gained from Quest, Job:%d", notify.Amount), questExpNotifyColor
		default:
			return "", color.RGBA{}
		}
	default:
		return "", color.RGBA{}
	}
}

func formatExpNotifyTemplate(manager *res.Manager, messageID int, fallback string, amount int32) string {
	text := ""
	if manager != nil {
		text, _ = manager.MsgString(messageID)
	}
	if text == "" {
		return fmt.Sprintf(fallback, amount)
	}
	return strings.Replace(text, "%d", strconv.FormatInt(int64(amount), 10), 1)
}

func pickupFailureMessage(manager *res.Manager, result uint8) string {
	// Match the original client's Zc_Item_Pickup_Ack message IDs. Unknown
	// results remain log-only in applyItemPickupAck.
	var messageID int
	var fallback string
	switch result {
	case 1, 6:
		messageID, fallback = 53, "You can't get this item."
	case 2:
		messageID, fallback = 52, "You can't carry any more weight."
	case 4:
		messageID, fallback = 220, "You can't carry any more items."
	case 5:
		messageID, fallback = 279, "You can't carry any more of this item."
	default:
		return ""
	}
	if manager != nil {
		if message, ok := manager.MsgString(messageID); ok {
			return message
		}
	}
	return fallback
}

func formatPickupConsoleMessage(manager *res.Manager, pickup network.ItemPickupAck) string {
	itemName := fmt.Sprintf("item %d", pickup.ItemID)
	if manager != nil {
		if name, ok := manager.ItemDisplayName(int(pickup.ItemID), pickup.Identified); ok && name != "" {
			itemName = name
		}
	}
	amount := int(pickup.Amount)
	if amount <= 0 {
		amount = 1
	}
	template := ""
	if manager != nil {
		template, _ = manager.MsgString(153)
	}
	if template == "" {
		template = "You got %s %d."
	}
	if strings.Contains(template, "%s") {
		template = strings.Replace(template, "%s", itemName, 1)
	} else {
		template = strings.TrimSpace(template + " " + itemName)
	}
	if strings.Contains(template, "%d") {
		template = strings.Replace(template, "%d", fmt.Sprintf("%d", amount), 1)
	} else if amount != 1 {
		template = strings.TrimSpace(fmt.Sprintf("%s %d", template, amount))
	}
	return template
}

func formatMVPRewardMessage(manager *res.Manager, reward network.MVPReward) string {
	message := func(id int, fallback string) string {
		if manager != nil {
			if text, ok := manager.MsgString(id); ok && text != "" {
				return text
			}
		}
		return fallback
	}
	switch reward.PacketID {
	case network.PacketZCMVPItem:
		name := fmt.Sprintf("item %d", reward.Value)
		if manager != nil {
			if text, ok := manager.ItemDisplayName(int(reward.Value), true); ok && text != "" {
				name = text
			}
		}
		return fmt.Sprintf("%s %s %s", message(143, "Congratulations! You are the MVP! Your reward item is"), name, message(144, "!!"))
	case network.PacketZCMVPExp:
		return fmt.Sprintf("%s %d %s", message(145, "Congratulations! You are the MVP! Your reward EXP Points are"), reward.Value, message(146, "!!"))
	case network.PacketZCThrowMVPItem:
		return message(147, "You are the MVP, but cannot obtain the reward because you are overweight.")
	default:
		return ""
	}
}
