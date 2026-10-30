package network

import (
	"encoding/binary"
	"testing"
)

func TestInstancePacketLayouts2008(t *testing.T) {
	queue := make([]byte, 65)
	copy(queue[2:63], "Endless Tower")
	binary.LittleEndian.PutUint16(queue[63:], 32767)
	q, ok, err := ParseInstanceQueue(Packet{ID: PacketZCInstanceQueue, Data: queue})
	if err != nil || !ok || q != (InstanceQueue{Name: "Endless Tower", Position: 32767}) {
		t.Fatalf("queue = %+v, %v, %v", q, ok, err)
	}
	position, ok, err := ParseInstanceQueuePosition(Packet{ID: PacketZCInstanceQueuePosition, Data: []byte{0xcc, 2, 0xff, 0xff}})
	if err != nil || !ok || position != -1 {
		t.Fatalf("cancel = %d, %v, %v", position, ok, err)
	}
	info := make([]byte, 71)
	copy(info[2:63], "Sealed Catacomb")
	binary.LittleEndian.PutUint32(info[63:], 1800007200)
	binary.LittleEndian.PutUint32(info[67:], 1800000300)
	i, ok, err := ParseInstanceInfo(Packet{ID: PacketZCInstanceInfo, Data: info})
	if err != nil || !ok || i != (InstanceInfo{Name: "Sealed Catacomb", ExpiresAt: 1800007200, IdleExpiresAt: 1800000300}) {
		t.Fatalf("info = %+v, %v, %v", i, ok, err)
	}
	for kind := InstanceNotifyTime; kind <= InstanceNotifyCreateFailed; kind++ {
		data := make([]byte, 10)
		binary.LittleEndian.PutUint32(data[2:], uint32(kind))
		binary.LittleEndian.PutUint32(data[6:], 1800000300)
		n, ok, err := ParseInstanceNotify(Packet{ID: PacketZCInstanceNotify, Data: data})
		if err != nil || !ok || n.Type != kind || n.IdleExpiresAt != 1800000300 {
			t.Fatalf("notify = %+v, %v, %v", n, ok, err)
		}
	}
}

func TestInstancePacketsRejectMalformedAndIgnoreUnrelated(t *testing.T) {
	parsers := []struct {
		id     uint16
		length int
		parse  func(Packet) (bool, error)
	}{
		{PacketZCInstanceQueue, 65, func(p Packet) (bool, error) { _, ok, err := ParseInstanceQueue(p); return ok, err }},
		{PacketZCInstanceQueuePosition, 4, func(p Packet) (bool, error) { _, ok, err := ParseInstanceQueuePosition(p); return ok, err }},
		{PacketZCInstanceInfo, 71, func(p Packet) (bool, error) { _, ok, err := ParseInstanceInfo(p); return ok, err }},
		{PacketZCInstanceNotify, 10, func(p Packet) (bool, error) { _, ok, err := ParseInstanceNotify(p); return ok, err }},
	}
	for _, parser := range parsers {
		for n := 0; n <= parser.length+1; n++ {
			if n == parser.length {
				continue
			}
			if ok, err := parser.parse(Packet{ID: parser.id, Data: make([]byte, n)}); err == nil || ok {
				t.Fatalf("0x%04x accepted %d bytes", parser.id, n)
			}
		}
		if ok, err := parser.parse(Packet{ID: 0x0080}); ok || err != nil {
			t.Fatalf("0x%04x handled unrelated packet", parser.id)
		}
	}
	if _, _, err := ParseInstanceNotify(Packet{ID: PacketZCInstanceNotify, Data: []byte{0xce, 2, 255, 255, 255, 255, 0, 0, 0, 0}}); err == nil {
		t.Fatal("unknown notification accepted")
	}
}

func TestInstancePacketFramingAcrossPartialReads(t *testing.T) {
	var stream []byte
	ids := []uint16{PacketZCInstanceQueue, PacketZCInstanceQueuePosition, PacketZCInstanceInfo, PacketZCInstanceNotify, 0x0080}
	lengths := []int{65, 4, 71, 10, 7}
	for n, id := range ids {
		data := make([]byte, lengths[n])
		binary.LittleEndian.PutUint16(data, id)
		stream = append(stream, data...)
	}
	framer := NewFramer(PacketLengths2008())
	var packets []Packet
	for _, b := range stream {
		part, err := framer.Push([]byte{b})
		if err != nil {
			t.Fatal(err)
		}
		packets = append(packets, part...)
	}
	if len(packets) != len(ids) {
		t.Fatalf("packets = %d", len(packets))
	}
	for n, p := range packets {
		if p.ID != ids[n] || len(p.Data) != lengths[n] {
			t.Fatalf("packet %d = %v", n, p)
		}
	}
	if _, exists := PacketLengths2008()[PacketCZInstanceCommand]; exists {
		t.Fatal("outgoing command registered as an incoming packet")
	}
}
