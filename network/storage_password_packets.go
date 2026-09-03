package network

import (
	"encoding/binary"
	"fmt"

	"github.com/kivutar/goro/glog"
)

const (
	PacketZCStoragePasswordPrompt uint16 = 0x023A
	PacketCZStoragePasswordReply  uint16 = 0x023B
	PacketZCStoragePasswordResult uint16 = 0x023C
)

type StoragePasswordPromptState uint16

const (
	StoragePasswordNotSet StoragePasswordPromptState = 0
	StoragePasswordSet    StoragePasswordPromptState = 1
	StoragePasswordLocked StoragePasswordPromptState = 8
)

type StoragePasswordReplyType uint16

const (
	StoragePasswordChange StoragePasswordReplyType = 2
	StoragePasswordCheck  StoragePasswordReplyType = 3
)

type StoragePasswordResultCode uint16

const (
	StoragePasswordChangeSucceeded StoragePasswordResultCode = 4
	StoragePasswordChangeFailed    StoragePasswordResultCode = 5
	StoragePasswordCheckSucceeded  StoragePasswordResultCode = 6
	StoragePasswordCheckFailed     StoragePasswordResultCode = 7
	StoragePasswordTooManyFailures StoragePasswordResultCode = 8
)

type StoragePasswordPrompt struct {
	State StoragePasswordPromptState
}

type StoragePasswordResult struct {
	Code       StoragePasswordResultCode
	ErrorCount uint16
}

func ParseStoragePasswordPrompt(packet Packet) (StoragePasswordPrompt, bool, error) {
	if packet.ID != PacketZCStoragePasswordPrompt {
		return StoragePasswordPrompt{}, false, nil
	}
	if len(packet.Data) < 4 {
		return StoragePasswordPrompt{}, false, fmt.Errorf("ZC_REQ_STORE_PASSWORD too short: %d", len(packet.Data))
	}
	return StoragePasswordPrompt{
		State: StoragePasswordPromptState(binary.LittleEndian.Uint16(packet.Data[2:4])),
	}, true, nil
}

func ParseStoragePasswordResult(packet Packet) (StoragePasswordResult, bool, error) {
	if packet.ID != PacketZCStoragePasswordResult {
		return StoragePasswordResult{}, false, nil
	}
	if len(packet.Data) < 6 {
		return StoragePasswordResult{}, false, fmt.Errorf("ZC_RESULT_STORE_PASSWORD too short: %d", len(packet.Data))
	}
	return StoragePasswordResult{
		Code:       StoragePasswordResultCode(binary.LittleEndian.Uint16(packet.Data[2:4])),
		ErrorCount: binary.LittleEndian.Uint16(packet.Data[4:6]),
	}, true, nil
}

// BuildStoragePasswordReplyPacket builds the 36-byte CZ_ACK_STORE_PASSWORD
// layout used by the 2008 client. The active password occupies bytes 4..19;
// a replacement password occupies bytes 20..35.
func BuildStoragePasswordReplyPacket(replyType StoragePasswordReplyType, password, newPassword string) []byte {
	packet := make([]byte, 36)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZStoragePasswordReply)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(replyType))
	writeFixedCString(packet[4:20], password)
	writeFixedCString(packet[20:36], newPassword)
	return packet
}

func (c *Client) SendStoragePasswordReply(replyType StoragePasswordReplyType, password, newPassword string) error {
	packet := BuildStoragePasswordReplyPacket(replyType, password, newPassword)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_ACK_STORE_PASSWORD opcode=0x%04X type=%d client_date=%d", ID(packet), replyType, c.clientDate)
	} else {
		glog.Warnf("send CZ_ACK_STORE_PASSWORD failed opcode=0x%04X type=%d client_date=%d: %v", ID(packet), replyType, c.clientDate, err)
	}
	return err
}
