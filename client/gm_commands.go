package client

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kivutar/goro/network"
)

// GM permissions are enforced by the server; clientinfo's admin list only
// controls presentation and is not an authoritative permission list.
func buildGMCommand(ctx Context, command, args string) (packet []byte, handled bool, err error) {
	handled = true
	switch command {
	case "/broadcast", "/b", "/nb", "/localbroadcast", "/lb", "/nlb":
		if args == "" {
			return nil, true, fmt.Errorf("usage: %s <message>", command)
		}
		message := args
		if command != "/nb" && command != "/nlb" {
			message = chatSenderName(ctx) + " : " + message
		}
		packet, err = network.BuildGMBroadcastPacket(message, command == "/localbroadcast" || command == "/lb" || command == "/nlb")
	case "/item", "/monster", "/shift", "/summon", "/rc", "/check":
		name := args
		if strings.HasPrefix(name, "\"") {
			if len(name) < 2 || !strings.HasSuffix(name, "\"") {
				return nil, true, fmt.Errorf("usage: %s <name>", command)
			}
			name = name[1 : len(name)-1]
		}
		opcode := network.PacketCZGMCreate
		switch command {
		case "/shift":
			opcode = network.PacketCZGMShift
		case "/summon":
			opcode = network.PacketCZGMSummon
		case "/rc":
			opcode = network.PacketCZGMRC
		case "/check":
			opcode = network.PacketCZGMCheck
		}
		packet, err = network.BuildGMNamePacket(opcode, name)
	case "/mapmove", "/mm":
		fields := strings.Fields(args)
		if len(fields) != 3 {
			return nil, true, fmt.Errorf("usage: %s <map> <x> <y>", command)
		}
		coords, parseErr := gmNumbers(fields[1:], 16)
		if parseErr != nil {
			return nil, true, parseErr
		}
		packet, err = network.BuildGMMapMovePacket(fields[0], uint16(coords[0]), uint16(coords[1]))
	case "/kill", "/kick", "/mute", "/unmute":
		fields := strings.Fields(args)
		count := 1
		if command == "/mute" {
			count = 2
		}
		if len(fields) != count {
			usage := command + " <account ID>"
			if command == "/mute" {
				usage += " <minutes>"
			}
			return nil, true, fmt.Errorf("usage: %s", usage)
		}
		ids, parseErr := gmNumbers(fields[:1], 32)
		if parseErr != nil {
			return nil, true, parseErr
		}
		if command == "/kill" || command == "/kick" {
			packet, err = network.BuildGMKickPacket(ids[0])
		} else {
			points := uint16(65535)
			if command == "/mute" {
				values, parseErr := gmNumbers(fields[1:], 16)
				if parseErr != nil {
					return nil, true, parseErr
				}
				points = uint16(values[0])
			}
			packet, err = network.BuildGMMannerPacket(ids[0], command == "/mute", points)
		}
	case "/hide", "/killall", "/resetstate", "/resetskill":
		if args != "" {
			return nil, true, fmt.Errorf("usage: %s", command)
		}
		switch command {
		case "/hide":
			packet = network.BuildGMHidePacket()
		case "/killall":
			packet = network.BuildGMKickAllPacket()
		default:
			packet = network.BuildGMResetPacket(command == "/resetskill")
		}
	case "/changemaptype", "/cmt":
		fields := strings.Fields(args)
		if len(fields) != 3 {
			return nil, true, fmt.Errorf("usage: %s <x> <y> <cell type 0–6>", command)
		}
		values, parseErr := gmNumbers(fields, 16)
		if parseErr != nil {
			return nil, true, parseErr
		}
		packet, err = network.BuildGMChangeMapTypePacket(uint16(values[0]), uint16(values[1]), uint16(values[2]))
	default:
		return nil, false, nil
	}
	return
}

func gmNumbers(fields []string, bits int) ([]uint32, error) {
	values := make([]uint32, len(fields))
	for i, field := range fields {
		value, err := strconv.ParseUint(field, 10, bits)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q (expected 0–%d)", field, uint64(1)<<bits-1)
		}
		values[i] = uint32(value)
	}
	return values, nil
}
