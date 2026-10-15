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

func TestBuildSuperNovicePrayerPacket(t *testing.T) {
	if got, want := BuildSuperNovicePrayerPacket(), []byte{0xED, 0x01}; !bytes.Equal(got, want) {
		t.Fatalf("Super Novice prayer packet = % X, want % X", got, want)
	}
}
