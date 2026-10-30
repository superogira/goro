package game

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func instanceTestPacket(id uint16, name string, first, second uint32) network.Packet {
	data := make([]byte, network.PacketLengths2008()[id])
	binary.LittleEndian.PutUint16(data, id)
	switch id {
	case network.PacketZCInstanceQueue:
		copy(data[2:63], name)
		binary.LittleEndian.PutUint16(data[63:], uint16(first))
	case network.PacketZCInstanceQueuePosition:
		binary.LittleEndian.PutUint16(data[2:], uint16(first))
	case network.PacketZCInstanceInfo:
		copy(data[2:63], name)
		binary.LittleEndian.PutUint32(data[63:], first)
		binary.LittleEndian.PutUint32(data[67:], second)
	case network.PacketZCInstanceNotify:
		binary.LittleEndian.PutUint32(data[2:], first)
		binary.LittleEndian.PutUint32(data[6:], second)
	}
	return network.Packet{ID: id, Data: data}
}

func TestInstancePacketsAndMapTransition(t *testing.T) {
	ctx := client.Context{Session: &session.Session{}, Input: input.NewState(), UIManager: &worldModeTestUIManager{}, ScreenW: 1024, ScreenH: 768}
	m := &WorldMode{}
	send := func(id uint16, first, second uint32) {
		m.handleNetworkPacket(ctx, instanceTestPacket(id, "Endless Tower", first, second), time.Now())
	}
	send(network.PacketZCInstanceQueue, 3, 0)
	if ctx.Session.Instance.QueuePosition != 3 || !m.ui.instanceWindow.IsOpen() {
		t.Fatal("queue not displayed")
	}
	send(network.PacketZCInstanceQueuePosition, 1, 0)
	if ctx.Session.Instance.QueuePosition != 1 {
		t.Fatal("queue update ignored")
	}
	send(network.PacketZCInstanceInfo, 1800007200, 1800000300)
	if ctx.Session.Instance.Phase() != session.InstanceReady {
		t.Fatal("not ready")
	}
	send(network.PacketZCInstanceNotify, 0, 0)
	if ctx.Session.Instance.Phase() != session.InstanceInProgress || ctx.Session.Instance.ExpiresAt != 1800007200 {
		t.Fatal("entry lost lifetime")
	}
	snapshot := ctx.Session.Instance
	m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCInstanceInfo, Data: []byte{0xcd, 2}}, time.Now())
	send(network.PacketZCInstanceNotify, 99, 0)
	if ctx.Session.Instance != snapshot {
		t.Fatal("malformed packet changed instance")
	}
	m.ui.instanceWindow.Close()
	next := m.nextWorldMode()
	next.rebindPersistentUI(ctx)
	if ctx.Session.Instance != snapshot || next.ui.instanceWindow.IsOpen() {
		t.Fatal("map transition lost state or reopened window")
	}
	next.ui.console.OnInstanceWindow()
	if !next.ui.instanceWindow.IsOpen() || m.ui.instanceWindow.IsOpen() {
		t.Fatal("console callback retained old mode")
	}
	next.handleNetworkPacket(ctx, instanceTestPacket(network.PacketZCInstanceNotify, "", 3, 0), time.Now())
	if ctx.Session.Instance.Name != "" || next.ui.instanceWindow.IsOpen() {
		t.Fatal("destruction did not clear instance")
	}
}

func TestInstanceTerminationAndDisconnect(t *testing.T) {
	packets := []network.Packet{
		instanceTestPacket(network.PacketZCInstanceQueuePosition, "", 0xffff, 0),
		{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0, 15}},
	}
	for reason := uint32(1); reason <= 4; reason++ {
		// Hercules does not initialize the trailing deadline for terminal notices.
		packets = append(packets, instanceTestPacket(network.PacketZCInstanceNotify, "", reason, 0xffffffff))
	}
	for _, p := range packets {
		ctx := client.Context{Session: &session.Session{Instance: session.Instance{Name: "Tower", Queued: true}}, ScreenW: 800, ScreenH: 600}
		m := &WorldMode{}
		m.ui.instanceWindow.Sync(ctx)
		m.handleNetworkPacket(ctx, p, time.Now())
		if ctx.Session.Instance != (session.Instance{}) || m.ui.instanceWindow.IsOpen() {
			t.Fatalf("packet %+v did not clear instance", p)
		}
		if p.ID == network.PacketZCInstanceNotify && p.Data[2] == 4 && len(m.ui.console.Messages()) != 0 {
			t.Fatal("silent instance close reported an error")
		}
	}
}

func TestInstanceReconnectAndPartyLeave(t *testing.T) {
	ctx := client.Context{Session: &session.Session{AccountID: 7}, ScreenW: 800, ScreenH: 600}
	m := &WorldMode{}
	// A reconnect or party join can send INFO without a preceding reservation.
	info := instanceTestPacket(network.PacketZCInstanceInfo, "Tower", 1800007200, 0)
	m.handleNetworkPacket(ctx, info, time.Now())
	if ctx.Session.Instance.Phase() != session.InstanceInProgress || !m.ui.instanceWindow.IsOpen() {
		t.Fatal("direct instance info did not restore the window")
	}
	for _, leave := range []struct {
		account uint32
		result  byte
		closed  bool
	}{{8, 0, false}, {7, 2, false}, {7, 0, true}} {
		data := make([]byte, 31)
		binary.LittleEndian.PutUint32(data[2:], leave.account)
		data[30] = leave.result
		m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCDeleteMemberFromGrp, Data: data}, time.Now())
		if (ctx.Session.Instance.Name == "") != leave.closed || m.ui.instanceWindow.IsOpen() == leave.closed {
			t.Fatalf("party leave %+v: instance=%+v, open=%t", leave, ctx.Session.Instance, m.ui.instanceWindow.IsOpen())
		}
	}
	m.handleNetworkPacket(ctx, network.Packet{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0, 15}}, time.Now())
	m.handleNetworkPacket(ctx, info, time.Now())
	if ctx.Session.Instance.Name != "" || m.ui.instanceWindow.IsOpen() {
		t.Fatal("late instance packet reopened the window after disconnect")
	}
}

func TestInstanceDestroyRequestWaitsForServer(t *testing.T) {
	netClient, conn := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Session: &session.Session{AccountID: 7}}
	ctx.Session.Party.Members = []session.PartyMember{{AccountID: 7}}
	ctx.Session.Instance = session.Instance{Name: "Tower", Queued: true, QueuePosition: 2}
	m := &WorldMode{}
	snapshot := ctx.Session.Instance
	m.destroyInstance(ctx)
	testutil.ReadPackets(t, conn, []byte{0xcf, 2, 2, 0, 0, 0})
	if ctx.Session.Instance != snapshot {
		t.Fatal("request cleared instance without server confirmation")
	}
	// A forbidden action must not send another packet. A following allowed
	// action exposes extra writes without waiting for a socket timeout.
	ctx.Session.Party.Members[0].Role = 1
	m.destroyInstance(ctx)
	ctx.Session.Party.Members[0].Role = 0
	ctx.Session.Dead = true
	m.destroyInstance(ctx)
	if err := netClient.SendQuestActive(1, true); err != nil {
		t.Fatal(err)
	}
	testutil.ReadPackets(t, conn, []byte{0xb6, 2, 1, 0, 0, 0, 1})
	ctx.Session.Dead = false
	ctx.Session.Instance.Queued = false
	ctx.Session.Instance.IdleExpiresAt = 1800000300
	m.destroyInstance(ctx)
	testutil.ReadPackets(t, conn, []byte{0xcf, 2, 2, 0, 0, 0})
	if ctx.Session.Instance.Phase() != session.InstanceReady {
		t.Fatal("ready instance changed before reply")
	}
}
