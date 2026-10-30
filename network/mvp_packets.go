package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketZCMVPItem      uint16 = 0x010A
	PacketZCMVPExp       uint16 = 0x010B
	PacketZCThrowMVPItem uint16 = 0x010D
)

type MVPReward struct {
	PacketID uint16
	Value    uint32
}

func ParseMVPReward(packet Packet) (MVPReward, bool, error) {
	var size int
	switch packet.ID {
	case PacketZCMVPItem:
		size = 4
	case PacketZCMVPExp:
		size = 6
	case PacketZCThrowMVPItem:
		size = 2
	default:
		return MVPReward{}, false, nil
	}
	if len(packet.Data) < size {
		return MVPReward{}, false, fmt.Errorf("MVP reward 0x%04X too short: %d", packet.ID, len(packet.Data))
	}
	reward := MVPReward{PacketID: packet.ID}
	switch packet.ID {
	case PacketZCMVPItem:
		reward.Value = uint32(binary.LittleEndian.Uint16(packet.Data[2:4]))
	case PacketZCMVPExp:
		reward.Value = binary.LittleEndian.Uint32(packet.Data[2:6])
	}
	return reward, true, nil
}
