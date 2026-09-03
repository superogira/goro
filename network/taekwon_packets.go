package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketZCStarSkill      uint16 = 0x020E
	PacketZCTaekwonPoint   uint16 = 0x0224
	PacketCZTaekwonRank    uint16 = 0x0225
	PacketZCTaekwonRank    uint16 = 0x0226
	PacketZCStarPlace      uint16 = 0x0253
	PacketCZAgreeStarPlace uint16 = 0x0254
)

const taekwonRankEntryCount = 10

type TaekwonMission struct {
	MonsterName string
	MonsterID   uint32
	Progress    uint8
	Result      uint8
}

type TaekwonPoint struct {
	Point      int32
	TotalPoint int32
}

type TaekwonRankEntry struct {
	Name  string
	Point int32
}

type TaekwonRanking struct {
	Entries []TaekwonRankEntry
}

type StarPlace struct {
	Place uint8
}

func ParseTaekwonMission(packet Packet) (TaekwonMission, bool, error) {
	if packet.ID != PacketZCStarSkill {
		return TaekwonMission{}, false, nil
	}
	if len(packet.Data) < 32 {
		return TaekwonMission{}, true, fmt.Errorf("ZC_STARSKILL too short: %d", len(packet.Data))
	}
	return TaekwonMission{
		MonsterName: decodeROFixedString(packet.Data[2:26]),
		MonsterID:   binary.LittleEndian.Uint32(packet.Data[26:30]),
		Progress:    packet.Data[30],
		Result:      packet.Data[31],
	}, true, nil
}

func ParseTaekwonPoint(packet Packet) (TaekwonPoint, bool, error) {
	if packet.ID != PacketZCTaekwonPoint {
		return TaekwonPoint{}, false, nil
	}
	if len(packet.Data) < 10 {
		return TaekwonPoint{}, true, fmt.Errorf("ZC_TAEKWON_POINT too short: %d", len(packet.Data))
	}
	return TaekwonPoint{
		Point:      int32(binary.LittleEndian.Uint32(packet.Data[2:6])),
		TotalPoint: int32(binary.LittleEndian.Uint32(packet.Data[6:10])),
	}, true, nil
}

func ParseTaekwonRanking(packet Packet) (TaekwonRanking, bool, error) {
	if packet.ID != PacketZCTaekwonRank {
		return TaekwonRanking{}, false, nil
	}
	if len(packet.Data) < 282 {
		return TaekwonRanking{}, true, fmt.Errorf("ZC_TAEKWON_RANK too short: %d", len(packet.Data))
	}
	entries := make([]TaekwonRankEntry, taekwonRankEntryCount)
	for i := range entries {
		nameOffset := 2 + i*24
		pointOffset := 2 + taekwonRankEntryCount*24 + i*4
		entries[i] = TaekwonRankEntry{
			Name:  decodeROFixedString(packet.Data[nameOffset : nameOffset+24]),
			Point: int32(binary.LittleEndian.Uint32(packet.Data[pointOffset : pointOffset+4])),
		}
	}
	return TaekwonRanking{Entries: entries}, true, nil
}

func ParseStarPlace(packet Packet) (StarPlace, bool, error) {
	if packet.ID != PacketZCStarPlace {
		return StarPlace{}, false, nil
	}
	if len(packet.Data) < 3 {
		return StarPlace{}, true, fmt.Errorf("ZC_STARPLACE too short: %d", len(packet.Data))
	}
	place := packet.Data[2]
	if place > 2 {
		return StarPlace{}, true, fmt.Errorf("ZC_STARPLACE invalid place: %d", place)
	}
	return StarPlace{Place: place}, true, nil
}

func BuildTaekwonRankRequestPacket() []byte {
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet, PacketCZTaekwonRank)
	return packet
}

func BuildAgreeStarPlacePacket(place uint8) []byte {
	packet := make([]byte, 3)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZAgreeStarPlace)
	packet[2] = place
	return packet
}

func (c *Client) SendTaekwonRankRequest() error {
	return c.Send(BuildTaekwonRankRequestPacket())
}

func (c *Client) SendAgreeStarPlace(place uint8) error {
	if place > 2 {
		return fmt.Errorf("invalid Star Gladiator place: %d", place)
	}
	return c.Send(BuildAgreeStarPlacePacket(place))
}
