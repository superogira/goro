package network

import (
	"encoding/binary"
	"testing"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

func TestParseNPCCutin(t *testing.T) {
	data := make([]byte, 67)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCShowImage2)
	encoded, _, err := transform.Bytes(korean.EUCKR.NewEncoder(), []byte("도우미"))
	if err != nil {
		t.Fatal(err)
	}
	copy(data[2:66], encoded)
	data[66] = NPCCutinRight

	cutin, ok, err := ParseNPCCutin(Packet{ID: PacketZCShowImage2, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("cut-in packet not recognized")
	}
	if cutin.Image != "도우미" || cutin.Position != NPCCutinRight {
		t.Fatalf("cut-in = %+v", cutin)
	}
}

func TestParseNPCCutinClear(t *testing.T) {
	data := make([]byte, 67)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCShowImage2)
	data[66] = NPCCutinClear

	cutin, ok, err := ParseNPCCutin(Packet{ID: PacketZCShowImage2, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || cutin.Image != "" || cutin.Position != NPCCutinClear {
		t.Fatalf("cut-in = %+v ok=%v", cutin, ok)
	}
}

func TestParseNPCCutinRejectsShortPacket(t *testing.T) {
	_, ok, err := ParseNPCCutin(Packet{ID: PacketZCShowImage2, Data: make([]byte, 66)})
	if !ok || err == nil {
		t.Fatalf("ok=%v err=%v, want recognized error", ok, err)
	}
}

func TestParseNPCSayDialog(t *testing.T) {
	data := make([]byte, 8+len("hello")+1)
	binary.LittleEndian.PutUint16(data[0:2], 0x00B4)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], 1234)
	copy(data[8:], "hello")

	dialog, ok, err := ParseNPCDialog(Packet{ID: 0x00B4, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("dialog packet not recognized")
	}
	if dialog.Kind != NPCDialogSay || dialog.NPCID != 1234 || dialog.Message != "hello" {
		t.Fatalf("dialog = %+v", dialog)
	}
}

func TestParseNPCMenuDialog(t *testing.T) {
	data := make([]byte, 8+len("A:B:C")+1)
	binary.LittleEndian.PutUint16(data[0:2], 0x00B7)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], 77)
	copy(data[8:], "A:B:C")

	dialog, ok, err := ParseNPCDialog(Packet{ID: 0x00B7, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("menu packet not recognized")
	}
	if dialog.Kind != NPCDialogMenu || dialog.NPCID != 77 || len(dialog.Options) != 3 || dialog.Options[1] != "B" {
		t.Fatalf("dialog = %+v", dialog)
	}
}

func TestParseNPCInputDialogs(t *testing.T) {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x0142)
	binary.LittleEndian.PutUint32(data[2:6], 0x11223344)
	dialog, ok, err := ParseNPCDialog(Packet{ID: 0x0142, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || dialog.Kind != NPCDialogNumberInput || dialog.NPCID != 0x11223344 {
		t.Fatalf("number dialog = %+v ok=%v", dialog, ok)
	}

	binary.LittleEndian.PutUint16(data[0:2], 0x01D4)
	dialog, ok, err = ParseNPCDialog(Packet{ID: 0x01D4, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || dialog.Kind != NPCDialogStringInput || dialog.NPCID != 0x11223344 {
		t.Fatalf("string dialog = %+v ok=%v", dialog, ok)
	}
}

func TestBuildNPCDialogPackets(t *testing.T) {
	if got := BuildNPCContactPacket(0x11223344, 0); ID(got) != 0x0090 || len(got) != 7 || binary.LittleEndian.Uint32(got[2:6]) != 0x11223344 || got[6] != 0 {
		t.Fatalf("contact packet = %x", got)
	}
	if got := BuildNPCNextPacket(0x11223344); ID(got) != 0x00B9 || len(got) != 6 || binary.LittleEndian.Uint32(got[2:6]) != 0x11223344 {
		t.Fatalf("next packet = %x", got)
	}
	if got := BuildNPCMenuChoicePacket(0x11223344, 2); ID(got) != 0x00B8 || len(got) != 7 || binary.LittleEndian.Uint32(got[2:6]) != 0x11223344 || got[6] != 2 {
		t.Fatalf("choice packet = %x", got)
	}
	if got := BuildNPCClosePacket(0x11223344); ID(got) != 0x0146 || len(got) != 6 || binary.LittleEndian.Uint32(got[2:6]) != 0x11223344 {
		t.Fatalf("close packet = %x", got)
	}
	if got := BuildNPCNumberInputPacket(0x11223344, -12); ID(got) != 0x0143 || len(got) != 10 || binary.LittleEndian.Uint32(got[2:6]) != 0x11223344 || int32(binary.LittleEndian.Uint32(got[6:10])) != -12 {
		t.Fatalf("number input packet = %x", got)
	}
	if got := BuildNPCStringInputPacket(0x11223344, "hello"); ID(got) != 0x01D5 || len(got) != 14 || binary.LittleEndian.Uint16(got[2:4]) != uint16(len(got)) || binary.LittleEndian.Uint32(got[4:8]) != 0x11223344 || string(got[8:13]) != "hello" || got[13] != 0 {
		t.Fatalf("string input packet = %x", got)
	}
}
