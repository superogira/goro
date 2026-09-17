package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketZCQuestList     uint16 = 0x02B1
	PacketZCQuestMissions uint16 = 0x02B2
	PacketZCQuestAdd      uint16 = 0x02B3
	PacketZCQuestDelete   uint16 = 0x02B4
	PacketZCQuestHunt     uint16 = 0x02B5
	PacketCZQuestActive   uint16 = 0x02B6
	PacketZCQuestActive   uint16 = 0x02B7
)

type QuestState struct {
	ID     uint32
	Active bool
}

type QuestObjective struct {
	MonsterID uint32
	Current   uint16
	Name      string
}

type QuestMission struct {
	ID         uint32
	ExpiresAt  uint32
	Objectives []QuestObjective
}

type QuestAdd struct {
	QuestMission
	Active bool
}

type QuestHunt struct {
	QuestID   uint32
	MonsterID uint32
	Required  uint16
	Current   uint16
}

func ParseQuestList(p Packet) ([]QuestState, bool, error) {
	if p.ID != PacketZCQuestList {
		return nil, false, nil
	}
	if err := validateQuestList(p, 8, 5, true); err != nil {
		return nil, false, err
	}
	entries := make([]QuestState, (len(p.Data)-8)/5)
	for i := range entries {
		offset := 8 + i*5
		entries[i] = QuestState{ID: binary.LittleEndian.Uint32(p.Data[offset:]), Active: p.Data[offset+4] != 0}
	}
	return entries, true, nil
}

func ParseQuestMissions(p Packet) ([]QuestMission, bool, error) {
	if p.ID != PacketZCQuestMissions {
		return nil, false, nil
	}
	// Each legacy record reserves three objective slots, even when unused.
	if err := validateQuestList(p, 8, 104, true); err != nil {
		return nil, false, err
	}
	entries := make([]QuestMission, (len(p.Data)-8)/104)
	for i := range entries {
		var err error
		entries[i], err = parseQuestMission(p.Data[8+i*104 : 8+(i+1)*104])
		if err != nil {
			return nil, false, err
		}
	}
	return entries, true, nil
}

func ParseQuestAdd(p Packet) (QuestAdd, bool, error) {
	if p.ID != PacketZCQuestAdd {
		return QuestAdd{}, false, nil
	}
	if len(p.Data) != 107 {
		return QuestAdd{}, false, fmt.Errorf("quest add: invalid length %d", len(p.Data))
	}
	// Unlike a mission-list record, ADD inserts its active byte after the ID.
	objectives, err := parseQuestObjectives(p.Data[15:])
	if err != nil {
		return QuestAdd{}, false, err
	}
	return QuestAdd{QuestMission: QuestMission{
		ID: binary.LittleEndian.Uint32(p.Data[2:]), ExpiresAt: binary.LittleEndian.Uint32(p.Data[11:]), Objectives: objectives,
	}, Active: p.Data[6] != 0}, true, nil
}

func ParseQuestDelete(p Packet) (uint32, bool, error) {
	if p.ID != PacketZCQuestDelete {
		return 0, false, nil
	}
	if len(p.Data) != 6 {
		return 0, false, fmt.Errorf("quest delete: invalid length %d", len(p.Data))
	}
	return binary.LittleEndian.Uint32(p.Data[2:]), true, nil
}

func ParseQuestHunt(p Packet) ([]QuestHunt, bool, error) {
	if p.ID != PacketZCQuestHunt {
		return nil, false, nil
	}
	if err := validateQuestList(p, 6, 12, false); err != nil {
		return nil, false, err
	}
	entries := make([]QuestHunt, (len(p.Data)-6)/12)
	for i := range entries {
		data := p.Data[6+i*12:]
		entries[i] = QuestHunt{
			QuestID: binary.LittleEndian.Uint32(data), MonsterID: binary.LittleEndian.Uint32(data[4:]),
			Required: binary.LittleEndian.Uint16(data[8:]), Current: binary.LittleEndian.Uint16(data[10:]),
		}
	}
	return entries, true, nil
}

func ParseQuestActive(p Packet) (QuestState, bool, error) {
	if p.ID != PacketZCQuestActive {
		return QuestState{}, false, nil
	}
	if len(p.Data) != 7 {
		return QuestState{}, false, fmt.Errorf("quest active: invalid length %d", len(p.Data))
	}
	return QuestState{ID: binary.LittleEndian.Uint32(p.Data[2:]), Active: p.Data[6] != 0}, true, nil
}

func BuildQuestActivePacket(id uint32, active bool) []byte {
	data := make([]byte, 7)
	binary.LittleEndian.PutUint16(data, PacketCZQuestActive)
	binary.LittleEndian.PutUint32(data[2:], id)
	if active {
		data[6] = 1
	}
	return data
}

func (c *Client) SendQuestActive(id uint32, active bool) error {
	return c.Send(BuildQuestActivePacket(id, active))
}

// Validate counts against the payload before allocating or indexing it.
func validateQuestList(p Packet, header, record int, wideCount bool) error {
	if len(p.Data) < header || int(binary.LittleEndian.Uint16(p.Data[2:])) != len(p.Data) {
		return fmt.Errorf("quest packet 0x%04X: invalid length %d", p.ID, len(p.Data))
	}
	count := uint32(binary.LittleEndian.Uint16(p.Data[4:]))
	if wideCount {
		count = binary.LittleEndian.Uint32(p.Data[4:])
	}
	if (len(p.Data)-header)%record != 0 || count != uint32((len(p.Data)-header)/record) {
		return fmt.Errorf("quest packet 0x%04X: count %d does not match length %d", p.ID, count, len(p.Data))
	}
	return nil
}

func parseQuestMission(data []byte) (QuestMission, error) {
	objectives, err := parseQuestObjectives(data[12:])
	return QuestMission{ID: binary.LittleEndian.Uint32(data), ExpiresAt: binary.LittleEndian.Uint32(data[8:]), Objectives: objectives}, err
}

func parseQuestObjectives(data []byte) ([]QuestObjective, error) {
	count := int(binary.LittleEndian.Uint16(data))
	if count > 3 {
		return nil, fmt.Errorf("legacy quest has %d objectives, maximum is 3", count)
	}
	objectives := make([]QuestObjective, count)
	for i := range objectives {
		entry := data[2+i*30 : 2+(i+1)*30]
		objectives[i] = QuestObjective{MonsterID: binary.LittleEndian.Uint32(entry), Current: binary.LittleEndian.Uint16(entry[4:]), Name: decodeROFixedString(entry[6:])}
	}
	return objectives, nil
}
