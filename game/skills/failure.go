package skills

import (
	"github.com/kivutar/goro/network"
)

func SkillFailMessage(ack network.SkillFailAck) string {
	if ack.Cause == 0 {
		if messages, ok := skillFailMessagesBySkill[ack.SkillID]; ok {
			if message, ok := messages[ack.Number]; ok {
				return message
			}
		}
	}
	if message, ok := skillFailMessagesByCause[ack.Cause]; ok {
		return message
	}
	return "Action failed."
}

var skillFailMessagesBySkill = map[uint16]map[uint32]string{
	1: {
		0: "Basic skill failed.",
		1: "Cannot use emotions.",
		2: "Cannot sit.",
		3: "Cannot chat.",
		4: "Cannot form a party.",
		5: "Cannot shout.",
		6: "Cannot PK.",
		7: "Cannot align.",
	},
	50: {
		0: "Steal failed.",
	},
}

var skillFailMessagesByCause = map[uint8]string{
	0: "Action failed.",
	1: "Not enough SP.",
	2: "Not enough HP.",
	4: "Action is still on cooldown.",
	5: "Not enough Zeny.",
	9: "Too much weight.",
}
