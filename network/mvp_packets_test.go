package network

import "testing"

func TestMVPRewards(t *testing.T) {
	data := []byte{0x0a, 0x01, 0xf5, 0x01, 0x0b, 0x01, 0xff, 0xff, 0xff, 0xff, 0x0d, 0x01}
	packets, err := NewFramer(PacketLengths2008()).Push(data)
	if err != nil || len(packets) != 3 {
		t.Fatalf("framed rewards = %+v, err = %v", packets, err)
	}
	want := []MVPReward{{PacketZCMVPItem, 501}, {PacketZCMVPExp, 0xffffffff}, {PacketZCThrowMVPItem, 0}}
	for i, packet := range packets {
		got, ok, err := ParseMVPReward(packet)
		if err != nil || !ok || got != want[i] {
			t.Fatalf("reward = %+v, ok = %t, err = %v, want %+v", got, ok, err, want[i])
		}
		for length := range len(packet.Data) {
			if _, _, err := ParseMVPReward(Packet{ID: packet.ID, Data: packet.Data[:length]}); err == nil {
				t.Fatalf("accepted truncated reward 0x%04X length %d", packet.ID, length)
			}
		}
	}
	if _, ok, err := ParseMVPReward(Packet{ID: PacketZCMVP}); ok || err != nil {
		t.Fatal("MVP effect interpreted as a reward")
	}
}
