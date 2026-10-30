package network

import (
	"encoding/binary"
	"fmt"
)

const (
	PacketZCInstanceQueue         uint16 = 0x02CB
	PacketZCInstanceQueuePosition uint16 = 0x02CC
	PacketZCInstanceInfo          uint16 = 0x02CD
	PacketZCInstanceNotify        uint16 = 0x02CE
	PacketCZInstanceCommand       uint16 = 0x02CF
)

type InstanceNotifyType uint32

const (
	InstanceNotifyTime InstanceNotifyType = iota
	InstanceNotifyExpired
	InstanceNotifyIdleExpired
	InstanceNotifyDestroyed
	InstanceNotifyCreateFailed
)

type InstanceQueue struct {
	Name     string
	Position int16
}

type InstanceInfo struct {
	Name          string
	ExpiresAt     uint32
	IdleExpiresAt uint32
}

type InstanceNotify struct {
	Type          InstanceNotifyType
	IdleExpiresAt uint32
}

func ParseInstanceQueue(p Packet) (InstanceQueue, bool, error) {
	if p.ID != PacketZCInstanceQueue {
		return InstanceQueue{}, false, nil
	}
	if len(p.Data) != 65 {
		return InstanceQueue{}, false, fmt.Errorf("instance queue: invalid length %d", len(p.Data))
	}
	name := fixedString(p.Data[2:63])
	if name == "" {
		return InstanceQueue{}, false, fmt.Errorf("instance queue: empty name")
	}
	return InstanceQueue{Name: name, Position: int16(binary.LittleEndian.Uint16(p.Data[63:]))}, true, nil
}

func ParseInstanceQueuePosition(p Packet) (int16, bool, error) {
	if p.ID != PacketZCInstanceQueuePosition {
		return 0, false, nil
	}
	if len(p.Data) != 4 {
		return 0, false, fmt.Errorf("instance queue position: invalid length %d", len(p.Data))
	}
	return int16(binary.LittleEndian.Uint16(p.Data[2:])), true, nil
}

func ParseInstanceInfo(p Packet) (InstanceInfo, bool, error) {
	if p.ID != PacketZCInstanceInfo {
		return InstanceInfo{}, false, nil
	}
	if len(p.Data) != 71 {
		return InstanceInfo{}, false, fmt.Errorf("instance info: invalid length %d", len(p.Data))
	}
	name := fixedString(p.Data[2:63])
	if name == "" {
		return InstanceInfo{}, false, fmt.Errorf("instance info: empty name")
	}
	return InstanceInfo{
		Name: name, ExpiresAt: binary.LittleEndian.Uint32(p.Data[63:]),
		IdleExpiresAt: binary.LittleEndian.Uint32(p.Data[67:]),
	}, true, nil
}

func ParseInstanceNotify(p Packet) (InstanceNotify, bool, error) {
	if p.ID != PacketZCInstanceNotify {
		return InstanceNotify{}, false, nil
	}
	if len(p.Data) != 10 {
		return InstanceNotify{}, false, fmt.Errorf("instance notification: invalid length %d", len(p.Data))
	}
	notify := InstanceNotify{Type: InstanceNotifyType(binary.LittleEndian.Uint32(p.Data[2:])), IdleExpiresAt: binary.LittleEndian.Uint32(p.Data[6:])}
	if notify.Type > InstanceNotifyCreateFailed {
		return InstanceNotify{}, false, fmt.Errorf("instance notification: unknown type %d", notify.Type)
	}
	return notify, true, nil
}

func (c *Client) SendInstanceDestroy() error {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data, PacketCZInstanceCommand)
	// rAthena and Hercules implement DESTROY_FORCE (2), including queued
	// reservations. The original client's cancel (0) / destroy (1) are ignored.
	binary.LittleEndian.PutUint32(data[2:], 2)
	return c.Send(data)
}
