package game

import (
	"encoding/binary"
	"image/color"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestWhisperIgnoreListCommandAndReply(t *testing.T) {
	netClient, server := testutil.NewConnection(t, 20080910)
	ctx := client.Context{Network: netClient, Session: session.New()}
	mode := NewWorldMode()
	if !mode.ui.console.SendText(ctx, " /EX ") {
		t.Fatal("list command fell through to public chat")
	}
	testutil.ReadPackets(t, server, []byte{0xd3, 0})
	if len(mode.ui.console.Messages()) != 0 {
		t.Fatal("displayed a list before the server replied")
	}
	data := make([]byte, 52)
	copy(data, []byte{0xd4, 0, 52, 0})
	copy(data[4:28], "Alice")
	copy(data[28:52], "Bob %s")
	mode.handleNetworkPacket(ctx, network.Packet{ID: 0x00d4, Data: data}, time.Now())
	mode.handleNetworkPacket(ctx, network.Packet{ID: 0x00d4, Data: []byte{0xd4, 0, 4, 0}}, time.Now())
	messages := mode.ui.console.Messages()
	want := []string{"Whisper block list:", "Alice", "Bob %s", "Your whisper block list is empty."}
	if len(messages) != len(want) {
		t.Fatalf("messages = %+v", messages)
	}
	for i, message := range messages {
		if message.Text != want[i] || message.Color != (color.RGBA{G: 255, B: 255, A: 255}) {
			t.Errorf("message %d = %+v, want %q in cyan", i, message, want[i])
		}
	}
	for _, command := range []string{`/ex "Bob Smith"`, `/in "Bob Smith"`} {
		mode.ui.console.SendText(ctx, command)
		packet := make([]byte, 27)
		packet[0] = 0xcf
		copy(packet[2:26], "Bob Smith")
		if strings.HasPrefix(command, "/in") {
			packet[26] = 1
		}
		testutil.ReadPackets(t, server, packet)
	}
}

func TestPartyPickupShowsEquipmentDetails(t *testing.T) {
	resources, err := res.NewManagerFS(fstest.MapFS{
		"data/idnum2itemdisplaynametable.txt": &fstest.MapFile{Data: []byte("1101#Sword#\n1102#Sword [3]#\n")},
		"data/num2itemdisplaynametable.txt":   &fstest.MapFile{Data: []byte("1101#Weapon#\n")},
		"data/itemslotcounttable.txt":         &fstest.MapFile{Data: []byte("1101#3#\n1102#3#\n")},
		"data/cardprefixnametable.txt":        &fstest.MapFile{Data: []byte("4001#Poring#\n4002#Fabre#\n")},
		"data/cardpostfixnametable.txt":       &fstest.MapFile{Data: []byte("4002#\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name       string
		itemID     uint16
		identified bool
		refine     uint8
		cards      [4]uint16
		want       string
	}{
		{"refined", 1101, true, 7, [4]uint16{}, "+7 Sword [3]"},
		{"carded", 1101, true, 7, [4]uint16{4001, 4001, 4002}, "+7 Double Poring Sword Fabre [3]"},
		{"existing slot suffix", 1102, true, 7, [4]uint16{4001}, "+7 Poring Sword [3]"},
		{"unidentified", 1101, false, 7, [4]uint16{4001}, "Weapon"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := client.Context{Resources: resources, Session: &session.Session{
				AccountID: 100,
				Party:     session.Party{Name: "Party", Members: []session.PartyMember{{AccountID: 300, Name: "Alice"}}},
			}}
			data := make([]byte, 22)
			binary.LittleEndian.PutUint16(data, network.PacketZCItemPickupParty)
			binary.LittleEndian.PutUint32(data[2:], 300)
			binary.LittleEndian.PutUint16(data[6:], tt.itemID)
			if tt.identified {
				data[8] = 1
			}
			data[10] = tt.refine
			for i, card := range tt.cards {
				binary.LittleEndian.PutUint16(data[11+2*i:], card)
			}
			binary.LittleEndian.PutUint16(data[19:], db.EquipWeapon)
			data[21] = db.ItemTypeWeapon
			mode := NewWorldMode()
			mode.handleNetworkPacket(ctx, network.Packet{ID: network.ID(data), Data: data}, time.Now())
			messages := mode.ui.console.Messages()
			want := "Alice has acquired " + tt.want + "."
			if len(messages) != 1 || messages[0].Text != want {
				t.Fatalf("pickup messages = %+v, want %q", messages, want)
			}
		})
	}
}

func TestRewardNotificationsDoNotChangeInventoryOrExperience(t *testing.T) {
	table := strings.Repeat("unused#\n", 143) + "MVP item:#\n!#\nMVP EXP:#\npoints!#\nToo heavy.#\n" +
		strings.Repeat("unused#\n", 1276-148) + "%s received %s.#\n"
	resources, err := res.NewManagerFS(fstest.MapFS{
		"data/msgstringtable.txt":             &fstest.MapFile{Data: []byte(table)},
		"data/idnum2itemdisplaynametable.txt": &fstest.MapFile{Data: []byte("501#Red Potion#\n")},
		"data/num2itemdisplaynametable.txt":   &fstest.MapFile{Data: []byte("501#Potion#\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, manager := range []*res.Manager{nil, resources} {
		mode := NewWorldMode()
		ctx := client.Context{Resources: manager, Session: &session.Session{
			AccountID: 100,
			Inventory: session.Inventory{Items: []session.InventoryItem{{Index: 2, ItemID: 501, Amount: 7}}},
			Progress:  session.Progress{BaseExp: 99},
			Party:     session.Party{Name: "Party", Members: []session.PartyMember{{AccountID: 300, Name: "Alice%s"}}},
		}}
		data := [][]byte{
			{0x0a, 0x01, 0xf5, 0x01},
			{0x0b, 0x01, 0xff, 0xff, 0xff, 0xff},
			{0x0b, 0x01, 0, 0, 0, 0},
			{0x0d, 0x01},
			{0xb8, 0x02, 0x2c, 0x01, 0, 0, 0xf5, 0x01, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0xb8, 0x02, 0x2c, 0x01, 0, 0, 0xf5, 0x01, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0xb8, 0x02, 0x2c, 0x01, 0, 0, 0xf5, 0x01, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0xb8, 0x02, 0x2d, 0x01, 0, 0, 0xf5, 0x01, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		}
		for _, packet := range data {
			mode.handleNetworkPacket(ctx, network.Packet{ID: network.ID(packet), Data: packet}, time.Now())
		}
		want := []string{
			"Congratulations! You are the MVP! Your reward item is item 501 !!",
			"Congratulations! You are the MVP! Your reward EXP Points are 4294967295 !!",
			"Congratulations! You are the MVP! Your reward EXP Points are 0 !!",
			"You are the MVP, but cannot obtain the reward because you are overweight.",
			"Alice%s has acquired item 501.", "Alice%s has acquired item 501.", "Alice%s has acquired item 501.", "Party member has acquired item 501.",
		}
		if manager != nil {
			want = []string{
				"MVP item: Red Potion !", "MVP EXP: 4294967295 points!", "MVP EXP: 0 points!", "Too heavy.",
				"Alice%s received Red Potion.", "Alice%s received Red Potion.", "Alice%s received Potion.", "Party member received Red Potion.",
			}
		}
		messages := mode.ui.console.Messages()
		got := make([]string, len(messages))
		for i, message := range messages {
			got[i] = message.Text
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("messages = %q, want %q", got, want)
		}
		if messages[3].Color == messages[0].Color || messages[0].Color != messages[4].Color {
			t.Fatal("reward messages must be blue, with overweight feedback in the error color")
		}
		if len(ctx.Session.Inventory.Items) != 1 || ctx.Session.Inventory.Items[0].Amount != 7 || ctx.Session.Progress.BaseExp != 99 {
			t.Fatal("reward notification changed inventory or experience")
		}
		if len(mode.effects().Active) != 0 {
			t.Fatal("reward notification duplicated the separately sent MVP effect")
		}
	}
}
