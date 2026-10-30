package network

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	PacketCZBroadcast       uint16 = 0x0099
	PacketCZGMKick          uint16 = 0x00CC
	PacketZCGMKickAck       uint16 = 0x00CD
	PacketCZGMKickAll       uint16 = 0x00CE
	PacketCZGMCreate        uint16 = 0x013F
	PacketCZGMMapMove       uint16 = 0x0140
	PacketCZGMManner        uint16 = 0x0149
	PacketZCGMMannerAck     uint16 = 0x014A
	PacketZCGMMannerGiven   uint16 = 0x014B
	PacketCZGMReset         uint16 = 0x0197
	PacketCZGMChangeMapType uint16 = 0x0198
	PacketCZLocalBroadcast  uint16 = 0x019C
	PacketCZGMHide          uint16 = 0x019D
	PacketCZGMShift         uint16 = 0x01BB
	PacketCZGMSummon        uint16 = 0x01BD
	PacketCZGMRC            uint16 = 0x0212
	PacketCZGMCheck         uint16 = 0x0213
	PacketZCGMStatus        uint16 = 0x0214
)

func gmPacket(opcode uint16, size int) []byte {
	p := make([]byte, size)
	binary.LittleEndian.PutUint16(p, opcode)
	return p
}

func validateGMText(text string, maxBytes int) error {
	if strings.TrimSpace(text) == "" || len(text) > maxBytes || strings.ContainsAny(text, "\x00\r\n") {
		return fmt.Errorf("text must contain 1–%d bytes without line breaks or NUL", maxBytes)
	}
	return nil
}

func BuildGMBroadcastPacket(text string, local bool) ([]byte, error) {
	if err := validateGMText(text, 255); err != nil {
		return nil, err
	}
	opcode := PacketCZBroadcast
	if local {
		opcode = PacketCZLocalBroadcast
	}
	p := gmPacket(opcode, 4+len(text)+1)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	copy(p[4:], text)
	return p, nil
}

// These 2008 requests share a 24-byte, NUL-terminated name field.
func BuildGMNamePacket(opcode uint16, name string) ([]byte, error) {
	switch opcode {
	case PacketCZGMCreate, PacketCZGMShift, PacketCZGMSummon, PacketCZGMRC, PacketCZGMCheck:
	default:
		return nil, fmt.Errorf("unsupported GM name request: 0x%04X", opcode)
	}
	if err := validateGMText(name, 23); err != nil {
		return nil, err
	}
	p := gmPacket(opcode, 26)
	copy(p[2:], name)
	return p, nil
}

func BuildGMMapMovePacket(mapName string, x, y uint16) ([]byte, error) {
	if err := validateGMText(mapName, 15); err != nil {
		return nil, err
	}
	p := gmPacket(PacketCZGMMapMove, 22)
	copy(p[2:18], mapName)
	binary.LittleEndian.PutUint16(p[18:], x)
	binary.LittleEndian.PutUint16(p[20:], y)
	return p, nil
}

func BuildGMKickPacket(actorID uint32) ([]byte, error) {
	if actorID == 0 {
		return nil, fmt.Errorf("target ID must be positive")
	}
	p := gmPacket(PacketCZGMKick, 6)
	binary.LittleEndian.PutUint32(p[2:], actorID)
	return p, nil
}

func BuildGMKickAllPacket() []byte { return gmPacket(PacketCZGMKickAll, 2) }

func BuildGMHidePacket() []byte {
	p := gmPacket(PacketCZGMHide, 6)
	binary.LittleEndian.PutUint32(p[2:], 0x40) // Invisible actor option.
	return p
}

func BuildGMResetPacket(skills bool) []byte {
	p := gmPacket(PacketCZGMReset, 4)
	if skills {
		binary.LittleEndian.PutUint16(p[2:], 1)
	}
	return p
}

func BuildGMMannerPacket(actorID uint32, negative bool, points uint16) ([]byte, error) {
	if actorID == 0 || points == 0 {
		return nil, fmt.Errorf("target ID and manner points must be positive")
	}
	if negative && points > 32767 {
		return nil, fmt.Errorf("mute duration must be between 1 and 32767 minutes")
	}
	p := gmPacket(PacketCZGMManner, 9)
	binary.LittleEndian.PutUint32(p[2:], actorID)
	if negative {
		p[6] = 1
	}
	binary.LittleEndian.PutUint16(p[7:], points)
	return p, nil
}

func BuildGMChangeMapTypePacket(x, y, cellType uint16) ([]byte, error) {
	if cellType > 6 {
		return nil, fmt.Errorf("cell type must be between 0 and 6")
	}
	p := gmPacket(PacketCZGMChangeMapType, 8)
	binary.LittleEndian.PutUint16(p[2:], x)
	binary.LittleEndian.PutUint16(p[4:], y)
	binary.LittleEndian.PutUint16(p[6:], cellType)
	return p, nil
}

type GMReply struct {
	Result uint32
	Name   string
	Status StatusSnapshot
}

func ParseGMReply(packet Packet) (GMReply, bool, error) {
	var size int
	switch packet.ID {
	case PacketZCGMKickAck:
		size = 3
	case PacketZCGMMannerAck:
		size = 6
	case PacketZCGMMannerGiven:
		size = 27
	case PacketZCGMStatus:
		size = 42
	default:
		return GMReply{}, false, nil
	}
	if len(packet.Data) < size {
		return GMReply{}, true, fmt.Errorf("GM reply 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	p := packet.Data
	var reply GMReply
	switch packet.ID {
	case PacketZCGMKickAck:
		reply.Result = uint32(p[2])
	case PacketZCGMMannerAck:
		reply.Result = binary.LittleEndian.Uint32(p[2:])
	case PacketZCGMMannerGiven:
		reply.Result = uint32(p[2])
		reply.Name = packetCString(p[3:27])
	case PacketZCGMStatus:
		s := &reply.Status
		s.Str, s.StrCost = int(p[2]), int(p[3])
		s.Agi, s.AgiCost = int(p[4]), int(p[5])
		s.Vit, s.VitCost = int(p[6]), int(p[7])
		s.Int, s.IntCost = int(p[8]), int(p[9])
		s.Dex, s.DexCost = int(p[10]), int(p[11])
		s.Luk, s.LukCost = int(p[12]), int(p[13])
		values := []*int{&s.Attack, &s.AttackBonus, &s.MatkMax, &s.MatkMin, &s.Defense, &s.DefenseBonus, &s.MDefense, &s.MDefenseBonus, &s.Hit, &s.Flee, &s.FleeBonus, &s.Critical, &s.ASPD, &s.ASPDBonus}
		for i, value := range values {
			*value = int(int16(binary.LittleEndian.Uint16(p[14+i*2:])))
		}
	}
	return reply, true, nil
}
