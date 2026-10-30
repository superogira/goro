package network

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func TestGMRepliesAreFramedAndParsed(t *testing.T) {
	kick := []byte{0xcd, 0, 1}
	manner := []byte{0x4a, 1, 3, 0, 0, 0}
	given := make([]byte, 27)
	copy(given, []byte{0x4b, 1, 1, 'G', 'M', 0})
	status := make([]byte, 42)
	copy(status, []byte{0x14, 2, 99, 11, 98, 10, 97, 9, 96, 8, 95, 7, 94, 6})
	for i, n := range []int16{100, 5, 150, 130, -10, 20, 30, 40, 200, 180, 25, 15, 175, 2} {
		binary.LittleEndian.PutUint16(status[14+i*2:], uint16(n))
	}
	framer := NewFramer(PacketLengths2008())
	var framed []Packet
	for _, data := range [][]byte{kick, manner, given, status} {
		for _, b := range data {
			packets, err := framer.Push([]byte{b})
			if err != nil {
				t.Fatal(err)
			}
			framed = append(framed, packets...)
		}
	}
	if len(framed) != 4 {
		t.Fatalf("framed packets=%d, want 4", len(framed))
	}
	want := []GMReply{
		{Result: 1}, {Result: 3}, {Result: 1, Name: "GM"},
		{Status: StatusSnapshot{
			Str: 99, StrCost: 11, Agi: 98, AgiCost: 10, Vit: 97, VitCost: 9,
			Int: 96, IntCost: 8, Dex: 95, DexCost: 7, Luk: 94, LukCost: 6,
			Attack: 100, AttackBonus: 5, MatkMax: 150, MatkMin: 130,
			Defense: -10, DefenseBonus: 20, MDefense: 30, MDefenseBonus: 40,
			Hit: 200, Flee: 180, FleeBonus: 25, Critical: 15, ASPD: 175, ASPDBonus: 2,
		}},
	}
	for i, packet := range framed {
		reply, handled, err := ParseGMReply(packet)
		if err != nil || !handled || !reflect.DeepEqual(reply, want[i]) {
			t.Fatalf("packet=%x reply=%+v handled=%t err=%v, want %+v", packet.ID, reply, handled, err, want[i])
		}
		for length := 0; length < len(packet.Data); length++ {
			if _, handled, err := ParseGMReply(Packet{ID: packet.ID, Data: packet.Data[:length]}); !handled || err == nil {
				t.Fatalf("accepted truncated packet=%x length=%d", packet.ID, length)
			}
		}
	}
	if _, handled, err := ParseGMReply(Packet{ID: 0x00BD}); handled || err != nil {
		t.Fatal("ordinary status packet handled as GM status")
	}
}

func TestGMNameBuilderRejectsUnrelatedOpcodes(t *testing.T) {
	if _, err := BuildGMNamePacket(PacketCZGMKickAll, "Player"); err == nil {
		t.Fatal("unrelated opcode accepted")
	}
}
