package network

import (
	"encoding/binary"
	"testing"
	"time"
)

func TestParseShowDigit(t *testing.T) {
	data := make([]byte, 7)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCShowDigit)
	data[2] = byte(ShowDigitCountDown)
	value := int32(-90)
	binary.LittleEndian.PutUint32(data[3:7], uint32(value))

	got, ok, err := ParseShowDigit(Packet{ID: PacketZCShowDigit, Data: data})
	if err != nil || !ok {
		t.Fatalf("ParseShowDigit() = %+v, %t, %v", got, ok, err)
	}
	if got.Mode != ShowDigitCountDown || got.Value != -90 {
		t.Fatalf("show digit = %+v", got)
	}
}

func TestParseServerInformationPackets(t *testing.T) {
	skillData := make([]byte, 6)
	binary.LittleEndian.PutUint16(skillData[0:2], PacketZCSkillMessage)
	binary.LittleEndian.PutUint32(skillData[2:6], 0x1f)
	skill, ok, err := ParseSkillMessage(Packet{ID: PacketZCSkillMessage, Data: skillData})
	if err != nil || !ok || skill.ID != 0x1f {
		t.Fatalf("ParseSkillMessage() = %+v, %t, %v", skill, ok, err)
	}

	bossData := make([]byte, 70)
	binary.LittleEndian.PutUint16(bossData[0:2], PacketZCBossInfo)
	bossData[2] = byte(BossInfoDead)
	binary.LittleEndian.PutUint32(bossData[3:7], 123)
	binary.LittleEndian.PutUint32(bossData[7:11], 45)
	binary.LittleEndian.PutUint16(bossData[11:13], 1)
	binary.LittleEndian.PutUint16(bossData[13:15], 5)
	binary.LittleEndian.PutUint16(bossData[15:17], 2)
	binary.LittleEndian.PutUint16(bossData[17:19], 10)
	copy(bossData[19:70], "Baphomet")
	boss, ok, err := ParseBossInfo(Packet{ID: PacketZCBossInfo, Data: bossData})
	if err != nil || !ok {
		t.Fatalf("ParseBossInfo() = %+v, %t, %v", boss, ok, err)
	}
	if boss.Kind != BossInfoDead || boss.X != 123 || boss.Y != 45 || boss.Name != "Baphomet" || boss.MinRespawn != time.Hour+5*time.Minute || boss.MaxRespawn != 2*time.Hour+10*time.Minute {
		t.Fatalf("boss info = %+v", boss)
	}

	progressData := make([]byte, 10)
	binary.LittleEndian.PutUint16(progressData[0:2], PacketZCProgress)
	binary.LittleEndian.PutUint32(progressData[2:6], 0x3366CC)
	binary.LittleEndian.PutUint32(progressData[6:10], 7)
	progress, ok, err := ParseProgressBar(Packet{ID: PacketZCProgress, Data: progressData})
	if err != nil || !ok || progress.Color != 0x3366CC || progress.Duration != 7*time.Second {
		t.Fatalf("ParseProgressBar() = %+v, %t, %v", progress, ok, err)
	}
}

func TestServerInformationPacketValidationAndFraming(t *testing.T) {
	if _, ok, err := ParseShowDigit(Packet{ID: PacketZCShowDigit, Data: []byte{0xB1, 0x01, 4, 0, 0, 0, 0}}); !ok || err == nil {
		t.Fatalf("invalid ShowDigit mode = ok:%t err:%v", ok, err)
	}
	if _, ok, err := ParseBossInfo(Packet{ID: PacketZCBossInfo, Data: make([]byte, 69)}); !ok || err == nil {
		t.Fatalf("short boss packet = ok:%t err:%v", ok, err)
	}
	if ok, err := ParseProgressBarCancel(Packet{ID: PacketZCProgressCancel, Data: []byte{0xF2, 0x02}}); !ok || err != nil {
		t.Fatalf("progress cancel = ok:%t err:%v", ok, err)
	}
	if got := BuildProgressBarDonePacket(); len(got) != 2 || ID(got) != PacketCZProgress {
		t.Fatalf("progress done packet = %v", got)
	}

	wants := map[uint16]int{
		PacketZCShowDigit:      7,
		PacketZCSkillMessage:   6,
		PacketZCBossInfo:       70,
		PacketZCProgress:       10,
		PacketZCProgressCancel: 2,
	}
	for id, want := range wants {
		if got := PacketLengths2008()[id]; got != want {
			t.Errorf("packet 0x%04X length = %d, want %d", id, got, want)
		}
	}
}
