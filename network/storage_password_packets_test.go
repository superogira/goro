package network

import (
	"encoding/binary"
	"testing"
)

func TestParseStoragePasswordPrompt(t *testing.T) {
	data := []byte{0x3A, 0x02, 0x01, 0x00}
	prompt, ok, err := ParseStoragePasswordPrompt(Packet{ID: PacketZCStoragePasswordPrompt, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || prompt.State != StoragePasswordSet {
		t.Fatalf("prompt = %+v, ok=%v; want password set", prompt, ok)
	}
}

func TestParseStoragePasswordResult(t *testing.T) {
	data := []byte{0x3C, 0x02, 0x07, 0x00, 0x02, 0x00}
	result, ok, err := ParseStoragePasswordResult(Packet{ID: PacketZCStoragePasswordResult, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || result.Code != StoragePasswordCheckFailed || result.ErrorCount != 2 {
		t.Fatalf("result = %+v, ok=%v; want check failed with two errors", result, ok)
	}
}

func TestStoragePasswordParsersRejectShortPackets(t *testing.T) {
	if _, _, err := ParseStoragePasswordPrompt(Packet{ID: PacketZCStoragePasswordPrompt, Data: make([]byte, 3)}); err == nil {
		t.Fatal("short password prompt was accepted")
	}
	if _, _, err := ParseStoragePasswordResult(Packet{ID: PacketZCStoragePasswordResult, Data: make([]byte, 5)}); err == nil {
		t.Fatal("short password result was accepted")
	}
}

func TestBuildStoragePasswordCheckReply(t *testing.T) {
	packet := BuildStoragePasswordReplyPacket(StoragePasswordCheck, "rosebud", "")
	if len(packet) != 36 || ID(packet) != PacketCZStoragePasswordReply {
		t.Fatalf("unexpected packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != uint16(StoragePasswordCheck) {
		t.Fatalf("type = %d; want %d", got, StoragePasswordCheck)
	}
	if got := string(packet[4:11]); got != "rosebud" {
		t.Fatalf("password = %q; want rosebud", got)
	}
	for i, value := range packet[11:] {
		if value != 0 {
			t.Fatalf("byte %d = %d; want zero", i+11, value)
		}
	}
}

func TestBuildStoragePasswordChangeReply(t *testing.T) {
	packet := BuildStoragePasswordReplyPacket(StoragePasswordChange, "", "newpass")
	if len(packet) != 36 || ID(packet) != PacketCZStoragePasswordReply {
		t.Fatalf("unexpected packet header: % X", packet)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); got != uint16(StoragePasswordChange) {
		t.Fatalf("type = %d; want %d", got, StoragePasswordChange)
	}
	for i, value := range packet[4:20] {
		if value != 0 {
			t.Fatalf("old-password byte %d = %d; want zero", i+4, value)
		}
	}
	if got := string(packet[20:27]); got != "newpass" {
		t.Fatalf("new password = %q; want newpass", got)
	}
}

func TestPacketLengths2008IncludesStoragePasswordReplies(t *testing.T) {
	lengths := PacketLengths2008()
	if got := lengths[PacketZCStoragePasswordPrompt]; got != 4 {
		t.Fatalf("prompt length = %d; want 4", got)
	}
	if got := lengths[PacketZCStoragePasswordResult]; got != 6 {
		t.Fatalf("result length = %d; want 6", got)
	}
}
