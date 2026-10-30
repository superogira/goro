package gm

import (
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

func HandlePacket(packet network.Packet, console *gameui.ChatConsole) bool {
	reply, handled, err := network.ParseGMReply(packet)
	if err != nil {
		glog.Warnf("parse GM reply: %v", err)
		return true
	}
	if !handled || console == nil {
		return handled
	}
	switch packet.ID {
	case network.PacketZCGMKickAck:
		if reply.Result == 0 {
			console.AddErrorMessage("Unable to disconnect the target.")
		} else {
			console.AddSystemMessage("Target disconnected.")
		}
	case network.PacketZCGMMannerAck:
		switch reply.Result {
		case 0:
			console.AddSystemMessage("Manner points updated.")
		case 1:
			console.AddErrorMessage("You have already used your manner points today.")
		case 2:
			console.AddErrorMessage("You must wait a month before giving this player another manner point.")
		case 3:
			console.AddErrorMessage("A GM has blocked your chat.")
		case 4:
			console.AddErrorMessage("The anti-spam system has blocked your chat.")
		case 5:
			// The following manner notification supplies the GM name and action.
		default:
			glog.Debugf("unknown manner result: %d", reply.Result)
		}
	case network.PacketZCGMMannerGiven:
		switch reply.Result {
		case 0:
			console.AddSystemMessage("You received positive manner points from %s.", reply.Name)
		case 1:
			console.AddSystemMessage("You received negative manner points from %s.", reply.Name)
		default:
			glog.Debugf("unknown manner adjustment: %d", reply.Result)
		}
	case network.PacketZCGMStatus:
		s := reply.Status
		// The reply has no target ID/name; do not mislabel overlapping checks.
		console.AddSystemMessage("GM character status:")
		console.AddSystemMessage("STR %d  AGI %d  VIT %d  INT %d  DEX %d  LUK %d", s.Str, s.Agi, s.Vit, s.Int, s.Dex, s.Luk)
		console.AddSystemMessage("Point costs: STR %d  AGI %d  VIT %d  INT %d  DEX %d  LUK %d", s.StrCost, s.AgiCost, s.VitCost, s.IntCost, s.DexCost, s.LukCost)
		console.AddSystemMessage("ATK %d+%d  MATK %d–%d  HIT %d  CRIT %d", s.Attack, s.AttackBonus, s.MatkMin, s.MatkMax, s.Hit, s.Critical)
		console.AddSystemMessage("DEF %d+%d  MDEF %d+%d  FLEE %d+%d  ASPD %d+%d", s.Defense, s.DefenseBonus, s.MDefense, s.MDefenseBonus, s.Flee, s.FleeBonus, s.ASPD, s.ASPDBonus)
	}
	return true
}
