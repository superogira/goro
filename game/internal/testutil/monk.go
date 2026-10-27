package testutil

import (
	"encoding/binary"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func MonkEffectContext() client.Context {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: db.JobMonk}
	w.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, Job: db.JobMonk, Appearance: true}
	return client.Context{World: w, Session: &session.Session{
		AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: db.JobMonk},
	}}
}

func SpiritSpherePacket(id uint16, actorID uint32, count uint16) network.Packet {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data, id)
	binary.LittleEndian.PutUint32(data[2:], actorID)
	binary.LittleEndian.PutUint16(data[6:], count)
	return network.Packet{ID: id, Data: data}
}
