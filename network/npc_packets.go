package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type NPCDialogKind int

const (
	NPCDialogSay NPCDialogKind = iota
	NPCDialogNext
	NPCDialogClose
	NPCDialogMenu
	NPCDialogClear
	NPCDialogNumberInput
	NPCDialogStringInput
)

type NPCDialog struct {
	Kind    NPCDialogKind
	NPCID   uint32
	Message string
	Options []string
}

const (
	PacketZCShowImage2 uint16 = 0x01B3

	NPCCutinLeft       uint8 = 0
	NPCCutinCenter     uint8 = 1
	NPCCutinRight      uint8 = 2
	NPCCutinWindow     uint8 = 3
	NPCCutinWindowless uint8 = 4
	NPCCutinClear      uint8 = 255
)

type NPCCutin struct {
	Image    string
	Position uint8
}

func ParseNPCCutin(packet Packet) (NPCCutin, bool, error) {
	if packet.ID != PacketZCShowImage2 {
		return NPCCutin{}, false, nil
	}
	if len(packet.Data) < 67 {
		return NPCCutin{}, true, fmt.Errorf("ZC_SHOW_IMAGE2 too short: %d", len(packet.Data))
	}
	return NPCCutin{
		Image:    decodeROFixedString(packet.Data[2:66]),
		Position: packet.Data[66],
	}, true, nil
}

func ParseNPCDialog(packet Packet) (NPCDialog, bool, error) {
	switch packet.ID {
	case 0x00B4:
		if len(packet.Data) < 8 {
			return NPCDialog{}, true, fmt.Errorf("ZC_SAY_DIALOG too short: %d", len(packet.Data))
		}
		return NPCDialog{
			Kind:    NPCDialogSay,
			NPCID:   binary.LittleEndian.Uint32(packet.Data[4:8]),
			Message: trimPacketCString(packet.Data[8:]),
		}, true, nil
	case 0x00B5:
		if len(packet.Data) < 6 {
			return NPCDialog{}, true, fmt.Errorf("ZC_WAIT_DIALOG too short: %d", len(packet.Data))
		}
		return NPCDialog{Kind: NPCDialogNext, NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
	case 0x00B6:
		if len(packet.Data) < 6 {
			return NPCDialog{}, true, fmt.Errorf("ZC_CLOSE_DIALOG too short: %d", len(packet.Data))
		}
		return NPCDialog{Kind: NPCDialogClose, NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
	case 0x00B7:
		if len(packet.Data) < 8 {
			return NPCDialog{}, true, fmt.Errorf("ZC_MENU_LIST too short: %d", len(packet.Data))
		}
		raw := trimPacketCString(packet.Data[8:])
		return NPCDialog{
			Kind:    NPCDialogMenu,
			NPCID:   binary.LittleEndian.Uint32(packet.Data[4:8]),
			Message: raw,
			Options: splitNPCMenuOptions(raw),
		}, true, nil
	case 0x08D6:
		if len(packet.Data) < 6 {
			return NPCDialog{}, true, fmt.Errorf("ZC_CLEAR_DIALOG too short: %d", len(packet.Data))
		}
		return NPCDialog{Kind: NPCDialogClear, NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
	case 0x0142:
		if len(packet.Data) < 6 {
			return NPCDialog{}, true, fmt.Errorf("ZC_OPEN_EDITDLG too short: %d", len(packet.Data))
		}
		return NPCDialog{Kind: NPCDialogNumberInput, NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
	case 0x01D4:
		if len(packet.Data) < 6 {
			return NPCDialog{}, true, fmt.Errorf("ZC_OPEN_EDITDLGSTR too short: %d", len(packet.Data))
		}
		return NPCDialog{Kind: NPCDialogStringInput, NPCID: binary.LittleEndian.Uint32(packet.Data[2:6])}, true, nil
	default:
		return NPCDialog{}, false, nil
	}
}

func BuildNPCContactPacket(npcID uint32, contactType uint8) []byte {
	var w Writer
	w.Uint16(0x0090)
	w.Uint32(npcID)
	w.Uint8(contactType)
	return w.Bytes()
}

func BuildNPCMenuChoicePacket(npcID uint32, choice uint8) []byte {
	var w Writer
	w.Uint16(0x00B8)
	w.Uint32(npcID)
	w.Uint8(choice)
	return w.Bytes()
}

func BuildNPCNextPacket(npcID uint32) []byte {
	var w Writer
	w.Uint16(0x00B9)
	w.Uint32(npcID)
	return w.Bytes()
}

func BuildNPCClosePacket(npcID uint32) []byte {
	var w Writer
	w.Uint16(0x0146)
	w.Uint32(npcID)
	return w.Bytes()
}

func BuildNPCNumberInputPacket(npcID uint32, value int32) []byte {
	var w Writer
	w.Uint16(0x0143)
	w.Uint32(npcID)
	w.Uint32(uint32(value))
	return w.Bytes()
}

func BuildNPCStringInputPacket(npcID uint32, value string) []byte {
	data := []byte(value)
	var w Writer
	w.Uint16(0x01D5)
	w.Uint16(uint16(8 + len(data) + 1))
	w.Uint32(npcID)
	_, _ = w.Write(data)
	w.Uint8(0)
	return w.Bytes()
}

func trimPacketCString(data []byte) string {
	if i := bytes.IndexByte(data, 0); i >= 0 {
		data = data[:i]
	}
	return strings.TrimSpace(string(data))
}

func splitNPCMenuOptions(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ":")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
