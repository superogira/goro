package network

import "encoding/binary"

const (
	PacketCZDoriDori uint16 = 0x01E7
	PacketCZChopokgi uint16 = 0x01ED
)

func BuildDoriDoriPacket() []byte {
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet, PacketCZDoriDori)
	return packet
}

func (c *Client) SendDoriDori() error {
	return c.Send(BuildDoriDoriPacket())
}

func BuildSuperNovicePrayerPacket() []byte {
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet, PacketCZChopokgi)
	return packet
}

func (c *Client) SendSuperNovicePrayer() error {
	return c.Send(BuildSuperNovicePrayerPacket())
}
