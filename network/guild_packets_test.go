package network

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

func TestBuildGuildPackets(t *testing.T) {
	menuInterface := BuildGuildMenuInterfaceRequestPacket()
	if len(menuInterface) != 2 || ID(menuInterface) != PacketCZReqGuildMenuInterface {
		t.Fatalf("BuildGuildMenuInterfaceRequestPacket len=%d id=0x%04x", len(menuInterface), ID(menuInterface))
	}

	create := BuildCreateGuildPacket(0x01020304, "Knights")
	if len(create) != 30 || ID(create) != PacketCZReqMakeGuild {
		t.Fatalf("BuildCreateGuildPacket len=%d id=0x%04x", len(create), ID(create))
	}
	if got := binary.LittleEndian.Uint32(create[2:6]); got != 0x01020304 {
		t.Fatalf("create char id = 0x%08x", got)
	}
	if got := string(create[6:13]); got != "Knights" {
		t.Fatalf("create name = %q", got)
	}

	invite := BuildRequestGuildInvitePacket(0x11111111, 0x22222222, 0x33333333)
	if len(invite) != 14 || ID(invite) != PacketCZReqJoinGuild {
		t.Fatalf("BuildRequestGuildInvitePacket len=%d id=0x%04x", len(invite), ID(invite))
	}
	if got := binary.LittleEndian.Uint32(invite[2:6]); got != 0x11111111 {
		t.Fatalf("invite target = 0x%08x", got)
	}

	reply := BuildGuildInviteReplyPacket(0x01020304, true)
	if len(reply) != 10 || ID(reply) != PacketCZJoinGuild {
		t.Fatalf("BuildGuildInviteReplyPacket len=%d id=0x%04x", len(reply), ID(reply))
	}
	if got := binary.LittleEndian.Uint32(reply[6:10]); got != 1 {
		t.Fatalf("reply accept = %d", got)
	}

	leave := BuildLeaveGuildPacket(0x01020304, 0x11111111, 0x22222222, "Moving on")
	if len(leave) != 54 || ID(leave) != PacketCZReqLeaveGuild {
		t.Fatalf("BuildLeaveGuildPacket len=%d id=0x%04x", len(leave), ID(leave))
	}
	if got := binary.LittleEndian.Uint32(leave[2:6]); got != 0x01020304 {
		t.Fatalf("leave guild id = 0x%08x", got)
	}
	if got := binary.LittleEndian.Uint32(leave[6:10]); got != 0x11111111 {
		t.Fatalf("leave account id = 0x%08x", got)
	}
	if got := binary.LittleEndian.Uint32(leave[10:14]); got != 0x22222222 {
		t.Fatalf("leave char id = 0x%08x", got)
	}
	if got := decodeROFixedString(leave[14:54]); got != "Moving on" {
		t.Fatalf("leave reason = %q", got)
	}

	expel := BuildExpelGuildMemberPacket(0x01020304, 0x33333333, 0x44444444, "Inactive")
	if len(expel) != 54 || ID(expel) != PacketCZReqExpelGuildMember || decodeROFixedString(expel[14:54]) != "Inactive" {
		t.Fatalf("BuildExpelGuildMemberPacket = %x", expel)
	}

	disband := BuildDisbandGuildPacket("Mandala")
	if len(disband) != 42 || ID(disband) != PacketCZReqDisbandGuild || decodeROFixedString(disband[2:42]) != "Mandala" {
		t.Fatalf("BuildDisbandGuildPacket = %x", disband)
	}

	menu := BuildGuildMenuRequestPacket(3)
	if len(menu) != 6 || ID(menu) != PacketCZReqGuildMenu {
		t.Fatalf("BuildGuildMenuRequestPacket len=%d id=0x%04x", len(menu), ID(menu))
	}
	if got := binary.LittleEndian.Uint32(menu[2:6]); got != 3 {
		t.Fatalf("menu tab = %d", got)
	}

	notice := BuildGuildNoticePacket(0x01020304, "Maintenance", "Gather in Prontera.")
	if len(notice) != 186 || ID(notice) != PacketCZGuildNotice {
		t.Fatalf("BuildGuildNoticePacket len=%d id=0x%04x", len(notice), ID(notice))
	}
	if got := binary.LittleEndian.Uint32(notice[2:6]); got != 0x01020304 {
		t.Fatalf("notice guild id = 0x%08x", got)
	}
	if got := string(notice[6 : 6+len("Maintenance")]); got != "Maintenance" {
		t.Fatalf("notice subject = %q", got)
	}
	if got := string(notice[66 : 66+len("Gather in Prontera.")]); got != "Gather in Prontera." {
		t.Fatalf("notice body = %q", got)
	}
	if notice[6+len("Maintenance")] != 0 || notice[66+len("Gather in Prontera.")] != 0 {
		t.Fatal("notice strings should be null padded")
	}

	message := BuildGuildMessagePacket("Kivutar : hello")
	if len(message) != 20 || ID(message) != PacketCZGuildMessage {
		t.Fatalf("BuildGuildMessagePacket len=%d id=0x%04x", len(message), ID(message))
	}
	if got := binary.LittleEndian.Uint16(message[2:4]); got != uint16(len(message)) {
		t.Fatalf("guild message packet length = %d, want %d", got, len(message))
	}
	if got := string(message[4 : len(message)-1]); got != "Kivutar : hello" {
		t.Fatalf("guild message = %q", got)
	}
	if message[len(message)-1] != 0 {
		t.Fatal("guild message should be null terminated")
	}
	if packet := BuildGuildMessagePacket("   "); packet != nil {
		t.Fatalf("empty guild message packet = %x, want nil", packet)
	}
	if packet := BuildGuildMessagePacket(strings.Repeat("x", 0xffff)); packet != nil {
		t.Fatalf("oversized guild message packet length = %d, want nil", len(packet))
	}

	longNotice := BuildGuildNoticePacket(1, strings.Repeat("s", 80), strings.Repeat("n", 140))
	if longNotice[65] != 0 || longNotice[185] != 0 {
		t.Fatal("notice strings should reserve a trailing null byte when truncated")
	}

	changeMember := BuildChangeGuildMemberPositionPacket([]GuildMemberPosition{
		{AccountID: 0x01020304, CharID: 0x05060708, PositionID: 7},
	})
	if len(changeMember) != 16 || ID(changeMember) != PacketCZReqChangeMember {
		t.Fatalf("BuildChangeGuildMemberPositionPacket len=%d id=0x%04x", len(changeMember), ID(changeMember))
	}
	if got := binary.LittleEndian.Uint32(changeMember[4:8]); got != 0x01020304 {
		t.Fatalf("member account id = 0x%08x", got)
	}
	if got := binary.LittleEndian.Uint32(changeMember[12:16]); got != 7 {
		t.Fatalf("member position id = %d", got)
	}

	changePositions := BuildRegisterGuildPositionsPacket([]GuildPosition{
		{PositionID: 2, Right: 0x11, Ranking: 3, PayRate: 50, PosName: "Leader"},
	})
	if len(changePositions) != 44 || ID(changePositions) != PacketCZRegGuildPosInfo {
		t.Fatalf("BuildRegisterGuildPositionsPacket len=%d id=0x%04x", len(changePositions), ID(changePositions))
	}
	if got := binary.LittleEndian.Uint32(changePositions[4:8]); got != 2 {
		t.Fatalf("position id = %d", got)
	}
	if got := string(changePositions[20:26]); got != "Leader" {
		t.Fatalf("position name = %q", got)
	}
}

func TestParseGuildChat(t *testing.T) {
	message := []byte("Kivutar : hello guild\x00")
	data := make([]byte, 4+len(message))
	binary.LittleEndian.PutUint16(data[0:2], PacketZCGuildChat)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	copy(data[4:], message)

	chat, ok, err := ParseGuildChat(Packet{ID: PacketZCGuildChat, Data: data})
	if !ok || err != nil {
		t.Fatalf("ParseGuildChat ok=%t err=%v", ok, err)
	}
	if chat.Message != "Kivutar : hello guild" {
		t.Fatalf("guild chat message = %q", chat.Message)
	}
	if _, ok, err := ParseGuildChat(Packet{ID: PacketZCGuildChat, Data: data[:4]}); !ok || err == nil {
		t.Fatalf("short guild chat ok=%t err=%v", ok, err)
	}
	if _, ok, err := ParseGuildChat(Packet{ID: PacketZCGuildNotice, Data: data}); ok || err != nil {
		t.Fatalf("unrelated packet ok=%t err=%v", ok, err)
	}
}

func TestParseGuildPackets(t *testing.T) {
	menuAccessData := make([]byte, 6)
	binary.LittleEndian.PutUint16(menuAccessData[0:2], PacketZCAckGuildMenuInterface)
	binary.LittleEndian.PutUint32(menuAccessData[2:6], 0xD7)
	menuAccess, ok, err := ParseGuildMenuAccess(Packet{ID: PacketZCAckGuildMenuInterface, Data: menuAccessData})
	if !ok || err != nil || menuAccess.Mask != 0xD7 {
		t.Fatalf("ParseGuildMenuAccess ok=%t err=%v access=%+v", ok, err, menuAccess)
	}
	if _, ok, err := ParseGuildMenuAccess(Packet{ID: PacketZCAckGuildMenuInterface, Data: menuAccessData[:5]}); !ok || err == nil {
		t.Fatalf("short guild menu access ok=%t err=%v", ok, err)
	}

	info := make([]byte, 110)
	binary.LittleEndian.PutUint16(info[0:2], PacketZCGuildInfo)
	binary.LittleEndian.PutUint32(info[2:6], 0x01020304)
	binary.LittleEndian.PutUint32(info[6:10], 5)
	binary.LittleEndian.PutUint32(info[10:14], 7)
	binary.LittleEndian.PutUint32(info[14:18], 16)
	binary.LittleEndian.PutUint32(info[18:22], 42)
	binary.LittleEndian.PutUint32(info[22:26], 1234)
	binary.LittleEndian.PutUint32(info[26:30], 9000)
	binary.LittleEndian.PutUint32(info[30:34], 11)
	binary.LittleEndian.PutUint32(info[34:38], 12)
	binary.LittleEndian.PutUint32(info[38:42], 13)
	binary.LittleEndian.PutUint32(info[42:46], 7)
	copy(info[46:70], []byte("Mandala"))
	copy(info[70:94], []byte("Arcer"))
	copy(info[94:110], []byte("Prontera"))
	parsedInfo, ok, err := ParseGuildInfo(Packet{ID: PacketZCGuildInfo, Data: info})
	if !ok || err != nil || parsedInfo.GuildID != 0x01020304 || parsedInfo.Level != 5 || parsedInfo.UserNum != 7 || parsedInfo.MaxUserNum != 16 || parsedInfo.UserAverageLevel != 42 || parsedInfo.Exp != 1234 || parsedInfo.MaxExp != 9000 || parsedInfo.Point != 11 || parsedInfo.Honor != 12 || parsedInfo.Virtue != 13 || parsedInfo.EmblemVersion != 7 || parsedInfo.GuildName != "Mandala" || parsedInfo.MasterName != "Arcer" || parsedInfo.ManageLand != "Prontera" {
		t.Fatalf("ParseGuildInfo ok=%t err=%v info=%+v", ok, err, parsedInfo)
	}

	info2 := append([]byte(nil), info...)
	info2 = append(info2, 0, 0, 0, 0)
	binary.LittleEndian.PutUint16(info2[0:2], PacketZCGuildInfo2)
	binary.LittleEndian.PutUint32(info2[110:114], 765)
	parsedInfo2, ok, err := ParseGuildInfo(Packet{ID: PacketZCGuildInfo2, Data: info2})
	if !ok || err != nil || parsedInfo2.Zeny != 765 || parsedInfo2.GuildName != "Mandala" {
		t.Fatalf("ParseGuildInfo2 ok=%t err=%v info=%+v", ok, err, parsedInfo2)
	}

	members := make([]byte, 4+104)
	binary.LittleEndian.PutUint16(members[0:2], PacketZCGuildMembers)
	binary.LittleEndian.PutUint16(members[2:4], uint16(len(members)))
	member := members[4:]
	binary.LittleEndian.PutUint32(member[0:4], 0x01020304)
	binary.LittleEndian.PutUint32(member[4:8], 0x05060708)
	binary.LittleEndian.PutUint16(member[8:10], 3)
	binary.LittleEndian.PutUint16(member[10:12], 4)
	binary.LittleEndian.PutUint16(member[12:14], 1)
	binary.LittleEndian.PutUint16(member[14:16], 4)
	binary.LittleEndian.PutUint16(member[16:18], 55)
	binary.LittleEndian.PutUint32(member[18:22], 12345)
	binary.LittleEndian.PutUint32(member[22:26], 1)
	binary.LittleEndian.PutUint32(member[26:30], 2)
	copyFixedName(member[30:80], "Memo")
	copyFixedName(member[80:104], "Arcer")
	parsedMembers, ok, err := ParseGuildMembers(Packet{ID: PacketZCGuildMembers, Data: members})
	if !ok || err != nil || len(parsedMembers) != 1 || parsedMembers[0].CharName != "Arcer" || parsedMembers[0].Memo != "Memo" || parsedMembers[0].Level != 55 || parsedMembers[0].PositionID != 2 {
		t.Fatalf("ParseGuildMembers ok=%t err=%v members=%+v", ok, err, parsedMembers)
	}

	memberInfo := make([]byte, 106)
	binary.LittleEndian.PutUint16(memberInfo[0:2], PacketZCGuildMemberInfo)
	copy(memberInfo[2:], member)
	parsedMemberInfo, ok, err := ParseGuildMemberInfo(Packet{ID: PacketZCGuildMemberInfo, Data: memberInfo})
	if !ok || err != nil || parsedMemberInfo.CharName != "Arcer" || parsedMemberInfo.PositionID != 2 || parsedMemberInfo.Level != 55 {
		t.Fatalf("ParseGuildMemberInfo ok=%t err=%v member=%+v", ok, err, parsedMemberInfo)
	}
	memberPositions := make([]byte, 4+12)
	binary.LittleEndian.PutUint16(memberPositions[0:2], PacketZCAckChangeMember)
	binary.LittleEndian.PutUint16(memberPositions[2:4], uint16(len(memberPositions)))
	binary.LittleEndian.PutUint32(memberPositions[4:8], 0x01020304)
	binary.LittleEndian.PutUint32(memberPositions[8:12], 0x05060708)
	binary.LittleEndian.PutUint32(memberPositions[12:16], 7)
	parsedMemberPositions, ok, err := ParseGuildMemberPositions(Packet{ID: PacketZCAckChangeMember, Data: memberPositions})
	if !ok || err != nil || len(parsedMemberPositions) != 1 || parsedMemberPositions[0].AccountID != 0x01020304 || parsedMemberPositions[0].CharID != 0x05060708 || parsedMemberPositions[0].PositionID != 7 {
		t.Fatalf("ParseGuildMemberPositions ok=%t err=%v positions=%+v", ok, err, parsedMemberPositions)
	}

	positions := make([]byte, 4+16)
	binary.LittleEndian.PutUint16(positions[0:2], PacketZCGuildPositions)
	binary.LittleEndian.PutUint16(positions[2:4], uint16(len(positions)))
	position := positions[4:]
	binary.LittleEndian.PutUint32(position[0:4], 2)
	binary.LittleEndian.PutUint32(position[4:8], 0x11)
	binary.LittleEndian.PutUint32(position[8:12], 3)
	binary.LittleEndian.PutUint32(position[12:16], 50)
	parsedPositions, ok, err := ParseGuildPositions(Packet{ID: PacketZCGuildPositions, Data: positions})
	if !ok || err != nil || len(parsedPositions) != 1 || parsedPositions[0].PositionID != 2 || parsedPositions[0].Right != 0x11 || parsedPositions[0].Ranking != 3 || parsedPositions[0].PayRate != 50 {
		t.Fatalf("ParseGuildPositions ok=%t err=%v positions=%+v", ok, err, parsedPositions)
	}
	ackPositions := make([]byte, 4+40)
	binary.LittleEndian.PutUint16(ackPositions[0:2], PacketZCAckGuildPosInfo)
	binary.LittleEndian.PutUint16(ackPositions[2:4], uint16(len(ackPositions)))
	ackPosition := ackPositions[4:]
	binary.LittleEndian.PutUint32(ackPosition[0:4], 2)
	binary.LittleEndian.PutUint32(ackPosition[4:8], 0x11)
	binary.LittleEndian.PutUint32(ackPosition[8:12], 3)
	binary.LittleEndian.PutUint32(ackPosition[12:16], 50)
	copyFixedName(ackPosition[16:40], "Leader")
	parsedAckPositions, ok, err := ParseGuildPositions(Packet{ID: PacketZCAckGuildPosInfo, Data: ackPositions})
	if !ok || err != nil || len(parsedAckPositions) != 1 || parsedAckPositions[0].PositionID != 2 || parsedAckPositions[0].PosName != "Leader" {
		t.Fatalf("ParseGuildPositions ack ok=%t err=%v positions=%+v", ok, err, parsedAckPositions)
	}

	positionNames := make([]byte, 4+28)
	binary.LittleEndian.PutUint16(positionNames[0:2], PacketZCGuildPosNames)
	binary.LittleEndian.PutUint16(positionNames[2:4], uint16(len(positionNames)))
	binary.LittleEndian.PutUint32(positionNames[4:8], 2)
	copyFixedName(positionNames[8:32], "Leader")
	parsedPositionNames, ok, err := ParseGuildPositionNames(Packet{ID: PacketZCGuildPosNames, Data: positionNames})
	if !ok || err != nil || len(parsedPositionNames) != 1 || parsedPositionNames[0].PositionID != 2 || parsedPositionNames[0].PosName != "Leader" {
		t.Fatalf("ParseGuildPositionNames ok=%t err=%v positions=%+v", ok, err, parsedPositionNames)
	}

	guildSkills := make([]byte, 6+37)
	binary.LittleEndian.PutUint16(guildSkills[0:2], PacketZCGuildSkillInfo)
	binary.LittleEndian.PutUint16(guildSkills[2:4], uint16(len(guildSkills)))
	binary.LittleEndian.PutUint16(guildSkills[4:6], 3)
	writeSkillInfoEntry(guildSkills[6:], 1, 10000, 1, 0, 1, "Official Guild Approval", true)
	parsedGuildSkills, ok, err := ParseGuildSkillInfo(Packet{ID: PacketZCGuildSkillInfo, Data: guildSkills})
	if !ok || err != nil || parsedGuildSkills.SkillPoints != 3 || len(parsedGuildSkills.Skills) != 1 || parsedGuildSkills.Skills[0].ID != 10000 || parsedGuildSkills.Skills[0].Name != "Official Guild Approval" {
		t.Fatalf("ParseGuildSkillInfo ok=%t err=%v info=%+v", ok, err, parsedGuildSkills)
	}

	expelHistory := make([]byte, 4+88)
	binary.LittleEndian.PutUint16(expelHistory[0:2], PacketZCGuildBanList)
	binary.LittleEndian.PutUint16(expelHistory[2:4], uint16(len(expelHistory)))
	copyFixedName(expelHistory[4:28], "Chjara")
	copyFixedName(expelHistory[28:52], "Kivutar")
	copyFixedName(expelHistory[52:92], "Testing")
	parsedExpelHistory, ok, err := ParseGuildExpelHistory(Packet{ID: PacketZCGuildBanList, Data: expelHistory})
	if !ok || err != nil || len(parsedExpelHistory) != 1 || parsedExpelHistory[0].CharName != "Chjara" || parsedExpelHistory[0].Account != "Kivutar" || parsedExpelHistory[0].Reason != "Testing" {
		t.Fatalf("ParseGuildExpelHistory ok=%t err=%v history=%+v", ok, err, parsedExpelHistory)
	}

	departure := make([]byte, 66)
	binary.LittleEndian.PutUint16(departure[0:2], PacketZCAckLeaveGuild)
	copyFixedName(departure[2:26], "Alice")
	copyFixedName(departure[26:66], "A new adventure")
	parsedDeparture, ok, err := ParseGuildMemberDeparture(Packet{ID: PacketZCAckLeaveGuild, Data: departure})
	if !ok || err != nil || parsedDeparture != (GuildMemberDeparture{CharName: "Alice", Reason: "A new adventure"}) {
		t.Fatalf("ParseGuildMemberDeparture ok=%t err=%v departure=%+v", ok, err, parsedDeparture)
	}

	expulsion := make([]byte, 90)
	binary.LittleEndian.PutUint16(expulsion[0:2], PacketZCAckExpelGuildMember)
	copyFixedName(expulsion[2:26], "Bob")
	copyFixedName(expulsion[26:66], "Inactive")
	copyFixedName(expulsion[66:90], "bob_account")
	parsedExpulsion, ok, err := ParseGuildMemberExpulsion(Packet{ID: PacketZCAckExpelGuildMember, Data: expulsion})
	if !ok || err != nil || parsedExpulsion != (GuildMemberExpulsion{CharName: "Bob", Reason: "Inactive", Account: "bob_account"}) {
		t.Fatalf("ParseGuildMemberExpulsion ok=%t err=%v expulsion=%+v", ok, err, parsedExpulsion)
	}

	disbandResult := make([]byte, 6)
	binary.LittleEndian.PutUint16(disbandResult[0:2], PacketZCAckDisbandGuild)
	binary.LittleEndian.PutUint32(disbandResult[2:6], 2)
	parsedDisband, ok, err := ParseGuildDisbandResult(Packet{ID: PacketZCAckDisbandGuild, Data: disbandResult})
	if !ok || err != nil || parsedDisband.Result != 2 {
		t.Fatalf("ParseGuildDisbandResult ok=%t err=%v result=%+v", ok, err, parsedDisband)
	}

	notice := make([]byte, 182)
	binary.LittleEndian.PutUint16(notice[0:2], PacketZCGuildNotice)
	copyFixedName(notice[2:62], "Maintenance")
	copyFixedName(notice[62:182], "Gather in Prontera.")
	parsedNotice, ok, err := ParseGuildNotice(Packet{ID: PacketZCGuildNotice, Data: notice})
	if !ok || err != nil || parsedNotice.Subject != "Maintenance" || parsedNotice.Notice != "Gather in Prontera." {
		t.Fatalf("ParseGuildNotice ok=%t err=%v notice=%+v", ok, err, parsedNotice)
	}

	belonging := make([]byte, 43)
	binary.LittleEndian.PutUint16(belonging[0:2], PacketZCUpdateGuildID)
	binary.LittleEndian.PutUint32(belonging[2:6], 0x01020304)
	binary.LittleEndian.PutUint32(belonging[6:10], 7)
	binary.LittleEndian.PutUint32(belonging[10:14], 0x11)
	belonging[14] = 1
	copy(belonging[19:43], []byte("Mandala"))
	parsedBelonging, ok, err := ParseGuildBelonging(Packet{ID: PacketZCUpdateGuildID, Data: belonging})
	if !ok || err != nil || parsedBelonging.GuildID != 0x01020304 || parsedBelonging.EmblemVersion != 7 || parsedBelonging.Mode != 0x11 || !parsedBelonging.IsMaster || parsedBelonging.GuildName != "Mandala" {
		t.Fatalf("ParseGuildBelonging ok=%t err=%v belonging=%+v", ok, err, parsedBelonging)
	}

	createResult := []byte{0x67, 0x01, 2}
	parsedCreate, ok, err := ParseGuildCreationResult(Packet{ID: PacketZCResultMakeGuild, Data: createResult})
	if !ok || err != nil || parsedCreate.Result != 2 {
		t.Fatalf("ParseGuildCreationResult ok=%t err=%v result=%+v", ok, err, parsedCreate)
	}

	inviteAck := []byte{0x69, 0x01, 1}
	parsedAck, ok, err := ParseGuildInviteAck(Packet{ID: PacketZCAckReqJoinGuild, Data: inviteAck})
	if !ok || err != nil || parsedAck.Result != 1 {
		t.Fatalf("ParseGuildInviteAck ok=%t err=%v ack=%+v", ok, err, parsedAck)
	}

	request := make([]byte, 30)
	binary.LittleEndian.PutUint16(request[0:2], PacketZCReqJoinGuild)
	binary.LittleEndian.PutUint32(request[2:6], 0x01020304)
	copy(request[6:30], []byte("Knights"))
	parsedRequest, ok, err := ParseGuildInviteRequest(Packet{ID: PacketZCReqJoinGuild, Data: request})
	if !ok || err != nil || parsedRequest.GuildID != 0x01020304 || parsedRequest.GuildName != "Knights" {
		t.Fatalf("ParseGuildInviteRequest ok=%t err=%v request=%+v", ok, err, parsedRequest)
	}

	emblem := make([]byte, 16)
	binary.LittleEndian.PutUint16(emblem[0:2], PacketZCGuildEmblem)
	binary.LittleEndian.PutUint16(emblem[2:4], uint16(len(emblem)))
	binary.LittleEndian.PutUint32(emblem[4:8], 0x01020304)
	binary.LittleEndian.PutUint32(emblem[8:12], 9)
	copy(emblem[12:], []byte{1, 2, 3, 4})
	parsedEmblem, ok, err := ParseGuildEmblemImage(Packet{ID: PacketZCGuildEmblem, Data: emblem})
	if !ok || err != nil || parsedEmblem.GuildID != 0x01020304 || parsedEmblem.EmblemVersion != 9 || string(parsedEmblem.Data) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("ParseGuildEmblemImage ok=%t err=%v emblem=%+v", ok, err, parsedEmblem)
	}

	change := make([]byte, 12)
	binary.LittleEndian.PutUint16(change[0:2], PacketZCChangeGuild)
	binary.LittleEndian.PutUint32(change[2:6], 0x11111111)
	binary.LittleEndian.PutUint32(change[6:10], 0x01020304)
	binary.LittleEndian.PutUint16(change[10:12], 11)
	parsedChange, ok, err := ParseGuildEmblemChange(Packet{ID: PacketZCChangeGuild, Data: change})
	if !ok || err != nil || parsedChange.ActorID != 0x11111111 || parsedChange.GuildID != 0x01020304 || parsedChange.EmblemVersion != 11 {
		t.Fatalf("ParseGuildEmblemChange ok=%t err=%v change=%+v", ok, err, parsedChange)
	}
}

func TestGuildPacketDirections(t *testing.T) {
	lengths := PacketLengths2008()
	for _, id := range []uint16{PacketCZReqMakeGuild, PacketCZReqJoinGuild, PacketCZJoinGuild, PacketCZReqGuildMenuInterface, PacketCZReqGuildMenu, PacketCZReqChangeMember, PacketCZReqOpenMember, PacketCZReqLeaveGuild, PacketCZReqExpelGuildMember, PacketCZReqDisbandGuild, PacketCZRegGuildPosInfo, PacketCZGuildNotice, PacketCZReqGuildMember, PacketCZGuildMessage, PacketCZReqGuildEmblem, PacketCZRegGuildEmblem} {
		if _, ok := lengths[id]; ok {
			t.Fatalf("0x%04X is client-to-server and must not be in the receive framer", id)
		}
	}
	for id, want := range map[uint16]int{
		PacketZCAckGuildMenuInterface: 6,
		PacketZCGuildInfo:             110,
		PacketZCGuildInfo2:            114,
		PacketZCResultMakeGuild:       3,
		PacketZCAckReqJoinGuild:       3,
		PacketZCReqJoinGuild:          30,
		PacketZCGuildMembers:          -1,
		PacketZCAckChangeMember:       -1,
		PacketZCAckOpenMember:         2,
		PacketZCAckLeaveGuild:         66,
		PacketZCAckExpelGuildMember:   90,
		PacketZCAckDisbandGuild:       6,
		PacketZCGuildPositions:        -1,
		PacketZCGuildSkillInfo:        -1,
		PacketZCGuildBanList:          -1,
		PacketZCGuildPosNames:         -1,
		PacketZCGuildNotice:           182,
		PacketZCUpdateGuildID:         43,
		PacketZCAckGuildPosInfo:       -1,
		PacketZCGuildMemberInfo:       106,
		PacketZCGuildChat:             -1,
		PacketZCGuildEmblem:           -1,
		PacketZCChangeGuild:           12,
	} {
		if got := lengths[id]; got != want {
			t.Fatalf("0x%04X receive length = %d, want %d", id, got, want)
		}
	}
}

func TestBuildRegisterGuildEmblemPacketValidatesBMP(t *testing.T) {
	for _, depth := range []uint16{8, 24} {
		offset := uint32(54)
		if depth == 8 {
			offset += 256 * 4
		}
		data := make([]byte, offset+24*24*uint32(depth/8))
		copy(data, "BM")
		binary.LittleEndian.PutUint32(data[2:], uint32(len(data)))
		binary.LittleEndian.PutUint32(data[10:], offset)
		binary.LittleEndian.PutUint32(data[14:], 40)
		binary.LittleEndian.PutUint32(data[18:], 24)
		binary.LittleEndian.PutUint32(data[22:], 24)
		binary.LittleEndian.PutUint16(data[26:], 1)
		binary.LittleEndian.PutUint16(data[28:], depth)
		packet, err := BuildRegisterGuildEmblemPacket(data)
		if err != nil {
			t.Fatalf("%d-bit emblem: %v", depth, err)
		}
		zr, err := zlib.NewReader(bytes.NewReader(packet[4:]))
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := io.ReadAll(zr)
		zr.Close()
		if err != nil || !bytes.Equal(decoded, data) {
			t.Fatalf("emblem round trip failed: %v", err)
		}
		for _, field := range []int{0, 2, 10, 14, 18, 22, 26, 28, 30} {
			bad := bytes.Clone(data)
			bad[field]++
			if _, err := BuildRegisterGuildEmblemPacket(bad); err == nil {
				t.Fatalf("%d-bit emblem accepted invalid header field at %d", depth, field)
			}
		}
		if _, err := BuildRegisterGuildEmblemPacket(data[:len(data)-1]); err == nil {
			t.Fatal("accepted truncated emblem")
		}
	}
	for _, data := range [][]byte{nil, []byte("BM"), make([]byte, 1801)} {
		if _, err := BuildRegisterGuildEmblemPacket(data); err == nil {
			t.Fatal("accepted invalid BMP size")
		}
	}
}
