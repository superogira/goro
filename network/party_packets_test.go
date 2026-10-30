package network

import (
	"encoding/binary"
	"testing"
)

func TestPartyItemPickup(t *testing.T) {
	data := []byte{0xb8, 0x02, 0x44, 0x33, 0x22, 0x11, 0xf5, 0x01, 1, 0, 7, 1, 0, 2, 0, 3, 0, 4, 0, 0x20, 0, 4}
	framer := NewFramer(PacketLengths2008())
	for _, b := range data[:21] {
		if packets, err := framer.Push([]byte{b}); err != nil || len(packets) != 0 {
			t.Fatalf("partial pickup framed = %+v, err = %v", packets, err)
		}
	}
	packets, err := framer.Push(append(data[21:], 0x0d, 0x01))
	if err != nil || len(packets) != 2 || packets[1].ID != PacketZCThrowMVPItem {
		t.Fatalf("pickup and following packet = %+v, err = %v", packets, err)
	}
	pickup, ok, err := ParsePartyItemPickup(packets[0])
	want := PartyItemPickup{AccountID: 0x11223344, ItemID: 501, Identified: true, Type: 4, Refine: 7, Cards: [4]uint16{1, 2, 3, 4}}
	if err != nil || !ok || pickup != want {
		t.Fatalf("pickup = %+v, ok = %t, err = %v", pickup, ok, err)
	}
	data[8] = 0
	if pickup, ok, err := ParsePartyItemPickup(Packet{ID: PacketZCItemPickupParty, Data: data}); err != nil || !ok || pickup.Identified {
		t.Fatalf("unidentified pickup = %+v, ok = %t, err = %v", pickup, ok, err)
	}
	for length := range len(data) {
		if _, _, err := ParsePartyItemPickup(Packet{ID: PacketZCItemPickupParty, Data: data[:length]}); err == nil {
			t.Fatalf("accepted truncated pickup length %d", length)
		}
	}
	if _, ok, err := ParsePartyItemPickup(Packet{ID: 0x00A0}); ok || err != nil {
		t.Fatal("personal pickup interpreted as party pickup")
	}
}

func TestBuildPartyPackets(t *testing.T) {
	makeParty := BuildMakePartyPacket("Goro")
	if len(makeParty) != 26 || ID(makeParty) != PacketCZMakeGroup || string(makeParty[2:6]) != "Goro" {
		t.Fatalf("BuildMakePartyPacket = len %d id 0x%04x data %q", len(makeParty), ID(makeParty), makeParty[2:6])
	}

	makeParty2 := BuildMakeParty2Packet("Goro", 1, 1)
	if len(makeParty2) != 28 || ID(makeParty2) != PacketCZMakeGroup2 || string(makeParty2[2:6]) != "Goro" || makeParty2[26] != 1 || makeParty2[27] != 1 {
		t.Fatalf("BuildMakeParty2Packet = len %d id 0x%04x data %x", len(makeParty2), ID(makeParty2), makeParty2)
	}

	invite := BuildPartyInvitePacket(0x11223344, "Alice")
	if len(invite) != 26 || ID(invite) != PacketCZPartyJoinReq || string(invite[2:7]) != "Alice" {
		t.Fatalf("BuildPartyInvitePacket = len %d id 0x%04x data %x", len(invite), ID(invite), invite)
	}

	ack := BuildPartyInviteAckPacket(0x01020304, true)
	if len(ack) != 7 || ID(ack) != PacketCZPartyJoinReqAck || binary.LittleEndian.Uint32(ack[2:6]) != 0x01020304 || ack[6] != 1 {
		t.Fatalf("BuildPartyInviteAckPacket = len %d id 0x%04x data %x", len(ack), ID(ack), ack)
	}

	leave := BuildLeavePartyPacket()
	if len(leave) != 2 || ID(leave) != PacketCZReqLeaveGroup {
		t.Fatalf("BuildLeavePartyPacket = len %d id 0x%04x", len(leave), ID(leave))
	}

	opt := BuildPartyOptionPacket(1)
	if len(opt) != 6 || ID(opt) != PacketCZChangeGroupExp || binary.LittleEndian.Uint32(opt[2:6]) != 1 {
		t.Fatalf("BuildPartyOptionPacket = len %d id 0x%04x data %x", len(opt), ID(opt), opt)
	}

	config := BuildPartyInviteConfigPacket(true)
	if len(config) != 3 || ID(config) != PacketCZPartyConfig || config[2] != 1 {
		t.Fatalf("BuildPartyInviteConfigPacket = len %d id 0x%04x data %x", len(config), ID(config), config)
	}

	expel := BuildExpelPartyMemberPacket(0x11223344, "Alice")
	if len(expel) != 30 || ID(expel) != PacketCZReqExpelGroupMember || binary.LittleEndian.Uint32(expel[2:6]) != 0x11223344 || string(expel[6:11]) != "Alice" {
		t.Fatalf("BuildExpelPartyMemberPacket = len %d id 0x%04x data %x", len(expel), ID(expel), expel)
	}
}

func TestParsePartyPackets(t *testing.T) {
	create := []byte{0xfa, 0x00, 0x00}
	parsedCreate, ok, err := ParsePartyCreateResult(Packet{ID: PacketZCAckMakeGroup, Data: create})
	if !ok || err != nil || parsedCreate.Result != 0 {
		t.Fatalf("ParsePartyCreateResult ok=%t err=%v result=%+v", ok, err, parsedCreate)
	}

	list := make([]byte, 28+46)
	binary.LittleEndian.PutUint16(list[0:2], PacketZCGroupList)
	binary.LittleEndian.PutUint16(list[2:4], uint16(len(list)))
	copy(list[4:28], []byte("Beta Party"))
	binary.LittleEndian.PutUint32(list[28:32], 0x11223344)
	copy(list[32:56], []byte("Alice"))
	copy(list[56:72], []byte("prontera"))
	list[72] = 0
	list[73] = 0
	parsedList, ok, err := ParsePartyList(Packet{ID: PacketZCGroupList, Data: list})
	if !ok || err != nil || parsedList.Name != "Beta Party" || len(parsedList.Members) != 1 || parsedList.Members[0].Name != "Alice" || parsedList.Members[0].AccountID != 0x11223344 {
		t.Fatalf("ParsePartyList ok=%t err=%v list=%+v", ok, err, parsedList)
	}

	inviteRequest := make([]byte, 30)
	binary.LittleEndian.PutUint16(inviteRequest[0:2], PacketZCPartyJoinReq)
	binary.LittleEndian.PutUint32(inviteRequest[2:6], 0x01020304)
	copy(inviteRequest[6:30], []byte("Beta Party"))
	parsedInviteRequest, ok, err := ParsePartyInviteRequest(Packet{ID: PacketZCPartyJoinReq, Data: inviteRequest})
	if !ok || err != nil || parsedInviteRequest.RequestID != 0x01020304 || parsedInviteRequest.Name != "Beta Party" {
		t.Fatalf("ParsePartyInviteRequest ok=%t err=%v request=%+v", ok, err, parsedInviteRequest)
	}

	if _, ok, err := ParsePartyInviteRequest(Packet{ID: PacketZCReqJoinGroup, Data: inviteRequest}); ok || err != nil {
		t.Fatalf("legacy 0x00FE parsed as 2008 party invite ok=%t err=%v", ok, err)
	}

	inviteAnswer := make([]byte, 30)
	binary.LittleEndian.PutUint16(inviteAnswer[0:2], PacketZCPartyJoinReqAck)
	copy(inviteAnswer[2:26], []byte("Alice"))
	binary.LittleEndian.PutUint32(inviteAnswer[26:30], 2)
	parsedInviteAnswer, ok, err := ParsePartyInviteAnswer(Packet{ID: PacketZCPartyJoinReqAck, Data: inviteAnswer})
	if !ok || err != nil || parsedInviteAnswer.Name != "Alice" || parsedInviteAnswer.Answer != 2 {
		t.Fatalf("ParsePartyInviteAnswer ok=%t err=%v answer=%+v", ok, err, parsedInviteAnswer)
	}

	if _, ok, err := ParsePartyInviteAnswer(Packet{ID: PacketZCAckReqJoinGroup, Data: inviteAnswer}); ok || err != nil {
		t.Fatalf("legacy 0x00FD parsed as 2008 party invite ack ok=%t err=%v", ok, err)
	}

	join := make([]byte, 79)
	binary.LittleEndian.PutUint16(join[0:2], PacketZCAddMemberToGroup)
	binary.LittleEndian.PutUint32(join[2:6], 0x22334455)
	binary.LittleEndian.PutUint32(join[6:10], 1)
	binary.LittleEndian.PutUint16(join[10:12], 12)
	binary.LittleEndian.PutUint16(join[12:14], 34)
	join[14] = 0
	copy(join[15:39], []byte("Beta Party"))
	copy(join[39:63], []byte("Bob"))
	copy(join[63:79], []byte("payon"))
	parsedJoin, ok, err := ParsePartyMemberJoin(Packet{ID: PacketZCAddMemberToGroup, Data: join})
	if !ok || err != nil || parsedJoin.Name != "Bob" || parsedJoin.MapName != "payon" || parsedJoin.X != 12 || parsedJoin.Y != 34 {
		t.Fatalf("ParsePartyMemberJoin ok=%t err=%v member=%+v", ok, err, parsedJoin)
	}

	hp := make([]byte, 10)
	binary.LittleEndian.PutUint16(hp[0:2], PacketZCNotifyHPToGroup)
	binary.LittleEndian.PutUint32(hp[2:6], 0x22334455)
	binary.LittleEndian.PutUint16(hp[6:8], 123)
	binary.LittleEndian.PutUint16(hp[8:10], 456)
	parsedHP, ok, err := ParsePartyMemberHP(Packet{ID: PacketZCNotifyHPToGroup, Data: hp})
	if !ok || err != nil || parsedHP.HP != 123 || parsedHP.MaxHP != 456 {
		t.Fatalf("ParsePartyMemberHP ok=%t err=%v hp=%+v", ok, err, parsedHP)
	}

	config := []byte{0xc9, 0x02, 0x01}
	parsedConfig, ok, err := ParsePartyInviteConfig(Packet{ID: PacketZCPartyConfig, Data: config})
	if !ok || err != nil || !parsedConfig.RefuseInvites {
		t.Fatalf("ParsePartyInviteConfig ok=%t err=%v config=%+v", ok, err, parsedConfig)
	}
}
