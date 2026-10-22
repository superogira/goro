package network

import (
	"encoding/binary"
	"testing"
)

func TestParseActorDevotion(t *testing.T) {
	data := make([]byte, 28)
	binary.LittleEndian.PutUint32(data[2:], 2000000)
	for i := 0; i < 5; i++ {
		binary.LittleEndian.PutUint32(data[6+i*4:], uint32(100+i))
	}
	got, ok, err := ParseActorDevotion(Packet{ID: 0x01CF, Data: data})
	if err != nil || !ok || got.SourceID != 2000000 || got.Targets != [5]uint32{100, 101, 102, 103, 104} {
		t.Fatalf("Devotion list = %+v ok=%t err=%v", got, ok, err)
	}
	for n := 0; n < 28; n++ {
		if _, _, err := ParseActorDevotion(Packet{ID: 0x01CF, Data: data[:n]}); err == nil {
			t.Fatalf("accepted truncated list of %d bytes", n)
		}
	}
	if _, ok, err := ParseActorDevotion(Packet{ID: 0x01D1, Data: data}); ok || err != nil {
		t.Fatal("parsed another packet as a Devotion list")
	}
}
