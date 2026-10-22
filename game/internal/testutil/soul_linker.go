package testutil

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func SoulLinkerEffectContext() client.Context {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20, Job: db.JobLinker}
	w.Actors[300] = worldstate.Actor{ID: 300, X: 18, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 150000}}
}

func SoulLinkerSkillPacket(skill uint16, result byte) network.Packet {
	var w network.Writer
	w.Uint16(0x011A)
	w.Uint16(skill)
	w.Uint16(1)
	w.Uint32(300)
	w.Uint32(2000000)
	w.Uint8(result)
	return network.Packet{ID: 0x011A, Data: w.Bytes()}
}

func NpcEffectUpdatePacket(id, opt3 uint32) network.Packet {
	var w network.Writer
	w.Uint16(network.PacketZCNPCShowEFSTUpdate)
	w.Uint32(id)
	w.Uint32(0)
	w.Uint32(75)
	w.Uint32(opt3)
	return network.Packet{ID: network.PacketZCNPCShowEFSTUpdate, Data: w.Bytes()}
}
