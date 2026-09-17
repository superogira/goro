package network

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func questListPacket(id uint16, count, header, record int) Packet {
	data := make([]byte, header+count*record)
	binary.LittleEndian.PutUint16(data, id)
	binary.LittleEndian.PutUint16(data[2:], uint16(len(data)))
	if header == 8 {
		binary.LittleEndian.PutUint32(data[4:], uint32(count))
	} else {
		binary.LittleEndian.PutUint16(data[4:], uint16(count))
	}
	return Packet{ID: id, Data: data}
}

func TestQuestListAndMissions2008(t *testing.T) {
	p := questListPacket(PacketZCQuestList, 2, 8, 5)
	binary.LittleEndian.PutUint32(p.Data[8:], 1001)
	p.Data[12] = 1
	binary.LittleEndian.PutUint32(p.Data[13:], 2002)
	states, ok, err := ParseQuestList(p)
	if err != nil || !ok || !reflect.DeepEqual(states, []QuestState{{ID: 1001, Active: true}, {ID: 2002}}) {
		t.Fatalf("list = %+v, %v, %v", states, ok, err)
	}
	p = questListPacket(PacketZCQuestMissions, 2, 8, 104)
	for i, id := range []uint32{1001, 2002} {
		record := p.Data[8+i*104:]
		binary.LittleEndian.PutUint32(record, id)
		binary.LittleEndian.PutUint32(record[8:], 1800000000+id)
		binary.LittleEndian.PutUint16(record[12:], uint16(i+1))
		for j := 0; j <= i; j++ {
			objective := record[14+j*30:]
			binary.LittleEndian.PutUint32(objective, uint32(1002+j))
			binary.LittleEndian.PutUint16(objective[4:], uint16(j))
			copy(objective[6:30], "Poring")
		}
	}
	missions, ok, err := ParseQuestMissions(p)
	if err != nil || !ok || len(missions) != 2 {
		t.Fatalf("missions = %+v, %v, %v", missions, ok, err)
	}
	// The second record follows three reserved slots, not just the used slot.
	if missions[1].ID != 2002 || missions[1].ExpiresAt != 1800002002 || len(missions[1].Objectives) != 2 || missions[1].Objectives[1] != (QuestObjective{MonsterID: 1003, Current: 1, Name: "Poring"}) {
		t.Fatalf("second mission = %+v", missions[1])
	}
}

func TestQuestAddHuntDeleteAndState(t *testing.T) {
	p := Packet{ID: PacketZCQuestAdd, Data: make([]byte, 107)}
	binary.LittleEndian.PutUint32(p.Data[2:], 1001)
	binary.LittleEndian.PutUint32(p.Data[11:], 1800000000)
	binary.LittleEndian.PutUint16(p.Data[15:], 3)
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint32(p.Data[17+30*i:], uint32(1002+i))
		copy(p.Data[23+30*i:47+30*i], "Poring")
	}
	quest, ok, err := ParseQuestAdd(p)
	if err != nil || !ok || quest.ID != 1001 || quest.Active || quest.ExpiresAt != 1800000000 || len(quest.Objectives) != 3 {
		t.Fatalf("add = %+v, %v, %v", quest, ok, err)
	}
	p = questListPacket(PacketZCQuestHunt, 1, 6, 12)
	binary.LittleEndian.PutUint32(p.Data[6:], 1001)
	binary.LittleEndian.PutUint32(p.Data[10:], 1002)
	binary.LittleEndian.PutUint16(p.Data[14:], 10)
	hunt, ok, err := ParseQuestHunt(p)
	if err != nil || !ok || !reflect.DeepEqual(hunt, []QuestHunt{{QuestID: 1001, MonsterID: 1002, Required: 10}}) {
		t.Fatalf("hunt = %+v, %v, %v", hunt, ok, err)
	}
	for _, active := range []bool{true, false} {
		data := BuildQuestActivePacket(1001, active)
		if len(data) != 7 || binary.LittleEndian.Uint16(data) != PacketCZQuestActive {
			t.Fatalf("request = %x", data)
		}
		state, ok, err := ParseQuestActive(Packet{ID: PacketZCQuestActive, Data: data})
		if err != nil || !ok || state != (QuestState{ID: 1001, Active: active}) {
			t.Fatalf("state = %+v, %v", state, err)
		}
	}
	id, ok, err := ParseQuestDelete(Packet{ID: PacketZCQuestDelete, Data: []byte{0xb4, 2, 0xe9, 3, 0, 0}})
	if err != nil || !ok || id != 1001 {
		t.Fatalf("delete = %d, %v, %v", id, ok, err)
	}
}

func TestQuestPacketsRejectMalformedPayloads(t *testing.T) {
	parsers := []struct {
		id     uint16
		length int
		parse  func(Packet) error
	}{
		{PacketZCQuestList, 8, func(p Packet) error { _, _, err := ParseQuestList(p); return err }},
		{PacketZCQuestMissions, 8, func(p Packet) error { _, _, err := ParseQuestMissions(p); return err }},
		{PacketZCQuestAdd, 107, func(p Packet) error { _, _, err := ParseQuestAdd(p); return err }},
		{PacketZCQuestDelete, 6, func(p Packet) error { _, _, err := ParseQuestDelete(p); return err }},
		{PacketZCQuestHunt, 6, func(p Packet) error { _, _, err := ParseQuestHunt(p); return err }},
		{PacketZCQuestActive, 7, func(p Packet) error { _, _, err := ParseQuestActive(p); return err }},
	}
	for _, parser := range parsers {
		for length := 0; length < parser.length; length++ {
			if err := parser.parse(Packet{ID: parser.id, Data: make([]byte, length)}); err == nil {
				t.Fatalf("0x%04X accepted %d bytes", parser.id, length)
			}
		}
	}
	for _, id := range []uint16{PacketZCQuestList, PacketZCQuestMissions, PacketZCQuestHunt} {
		header := 8
		if id == PacketZCQuestHunt {
			header = 6
		}
		p := questListPacket(id, 0, header, 1)
		for _, parser := range parsers {
			if parser.id != id {
				continue
			}
			if err := parser.parse(p); err != nil {
				t.Fatalf("empty 0x%04X: %v", id, err)
			}
			p.Data[4], p.Data[5] = 0xff, 0xff
			if err := parser.parse(p); err == nil {
				t.Fatalf("0x%04X accepted oversized count", id)
			}
		}
	}
	p := questListPacket(PacketZCQuestMissions, 1, 8, 104)
	p.Data[20] = 4
	if _, _, err := ParseQuestMissions(p); err == nil {
		t.Fatal("accepted four legacy objectives")
	}
	p.Data[20] = 0
	p.Data[2]--
	if _, _, err := ParseQuestMissions(p); err == nil {
		t.Fatal("accepted mismatched packet length")
	}
	lengths := PacketLengths2008()
	for _, parser := range parsers {
		if _, ok := lengths[parser.id]; !ok {
			t.Fatalf("quest 0x%04X missing from framer", parser.id)
		}
	}
}
