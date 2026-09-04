package network

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	PacketZCShowDigit      uint16 = 0x01B1
	PacketZCSkillMessage   uint16 = 0x0215
	PacketZCBossInfo       uint16 = 0x0293
	PacketZCProgress       uint16 = 0x02F0
	PacketCZProgress       uint16 = 0x02F1
	PacketZCProgressCancel uint16 = 0x02F2
)

type ShowDigitMode uint8

const (
	ShowDigitStatic ShowDigitMode = iota
	ShowDigitCountUp
	ShowDigitCountDown
	ShowDigitFastCountDown
)

type ShowDigit struct {
	Mode  ShowDigitMode
	Value int32
}

type SkillMessage struct {
	ID int32
}

type BossInfoKind uint8

const (
	BossInfoNotOnMap BossInfoKind = iota
	BossInfoAlive
	BossInfoAliveAnnounced
	BossInfoDead
)

type BossInfo struct {
	Kind       BossInfoKind
	X          uint32
	Y          uint32
	MinRespawn time.Duration
	MaxRespawn time.Duration
	Name       string
}

type ProgressBar struct {
	Color    uint32
	Duration time.Duration
}

func ParseShowDigit(packet Packet) (ShowDigit, bool, error) {
	if packet.ID != PacketZCShowDigit {
		return ShowDigit{}, false, nil
	}
	if len(packet.Data) < 7 {
		return ShowDigit{}, true, fmt.Errorf("ZC_SHOWDIGIT too short: %d", len(packet.Data))
	}
	mode := ShowDigitMode(packet.Data[2])
	if mode > ShowDigitFastCountDown {
		return ShowDigit{}, true, fmt.Errorf("ZC_SHOWDIGIT invalid mode: %d", mode)
	}
	return ShowDigit{
		Mode:  mode,
		Value: int32(binary.LittleEndian.Uint32(packet.Data[3:7])),
	}, true, nil
}

func ParseSkillMessage(packet Packet) (SkillMessage, bool, error) {
	if packet.ID != PacketZCSkillMessage {
		return SkillMessage{}, false, nil
	}
	if len(packet.Data) < 6 {
		return SkillMessage{}, true, fmt.Errorf("ZC_SKILLMSG too short: %d", len(packet.Data))
	}
	return SkillMessage{ID: int32(binary.LittleEndian.Uint32(packet.Data[2:6]))}, true, nil
}

func ParseBossInfo(packet Packet) (BossInfo, bool, error) {
	if packet.ID != PacketZCBossInfo {
		return BossInfo{}, false, nil
	}
	if len(packet.Data) < 70 {
		return BossInfo{}, true, fmt.Errorf("ZC_BOSS_INFO too short: %d", len(packet.Data))
	}
	kind := BossInfoKind(packet.Data[2])
	if kind > BossInfoDead {
		return BossInfo{}, true, fmt.Errorf("ZC_BOSS_INFO invalid kind: %d", kind)
	}
	return BossInfo{
		Kind:       kind,
		X:          binary.LittleEndian.Uint32(packet.Data[3:7]),
		Y:          binary.LittleEndian.Uint32(packet.Data[7:11]),
		MinRespawn: respawnDuration(packet.Data[11:15]),
		MaxRespawn: respawnDuration(packet.Data[15:19]),
		Name:       decodeROFixedString(packet.Data[19:70]),
	}, true, nil
}

func ParseProgressBar(packet Packet) (ProgressBar, bool, error) {
	if packet.ID != PacketZCProgress {
		return ProgressBar{}, false, nil
	}
	if len(packet.Data) < 10 {
		return ProgressBar{}, true, fmt.Errorf("ZC_PROGRESS too short: %d", len(packet.Data))
	}
	return ProgressBar{
		Color:    binary.LittleEndian.Uint32(packet.Data[2:6]),
		Duration: time.Duration(binary.LittleEndian.Uint32(packet.Data[6:10])) * time.Second,
	}, true, nil
}

func ParseProgressBarCancel(packet Packet) (bool, error) {
	if packet.ID != PacketZCProgressCancel {
		return false, nil
	}
	if len(packet.Data) < 2 {
		return true, fmt.Errorf("ZC_PROGRESS_CANCEL too short: %d", len(packet.Data))
	}
	return true, nil
}

func BuildProgressBarDonePacket() []byte {
	var w Writer
	w.Uint16(PacketCZProgress)
	return w.Bytes()
}

func (c *Client) SendProgressBarDone() error {
	return c.Send(BuildProgressBarDonePacket())
}

func respawnDuration(data []byte) time.Duration {
	hours := binary.LittleEndian.Uint16(data[0:2])
	minutes := binary.LittleEndian.Uint16(data[2:4])
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
}
