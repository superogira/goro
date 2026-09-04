package network

import (
	"bytes"
	"testing"
)

func TestBuildDoriDoriPacket(t *testing.T) {
	if got, want := BuildDoriDoriPacket(), []byte{0xE7, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("Dori Dori packet = % X, want % X", got, want)
	}
}
