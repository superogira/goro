package game

import (
	"bytes"
	"context"
	"io"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestTaekwonFamilySkillEffectsMatchRobrowser(t *testing.T) {
	spiritLinks := []uint16{
		db.SkillSLAlchemist, db.SkillSLMonk, db.SkillSLStar, db.SkillSLSage,
		db.SkillSLCrusader, db.SkillSLSupernovice, db.SkillSLKnight, db.SkillSLWizard,
		db.SkillSLPriest, db.SkillSLBarddancer, db.SkillSLRogue, db.SkillSLAssasin,
		db.SkillSLBlacksmith, db.SkillSLHunter, db.SkillSLSoullinker, db.SkillSLHigh,
	}
	for _, skillID := range spiritLinks {
		expectEffectIDs(t, "spirit link", skillEffectIDs(skillID), 424, 503)
	}
	expectEffectIDs(t, "SG_FUSION", skillEffectIDs(db.SkillSGFusion))
	expectEffectIDs(t, "SG_FUSION success", skillSuccessEffectSelfIDs(db.SkillSGFusion), effects.Quake)
	expectEffectIDs(t, "SL_SKA", skillEffectIDs(db.SkillSLSka), effects.SteelBody, effects.Gumgang2, effects.Quake)
}

func TestStarWarmAndSoulLinkStatusTints(t *testing.T) {
	swoo := actorStateTint(worldstate.Actor{Job: 1002, Opt3State: db.Opt3Overthrust})
	if swoo.R != 255 || swoo.G != 255 || swoo.B != 255 || swoo.A != 255 {
		t.Fatalf("Eswoo must not apply the player Overthrust tint: %+v", swoo)
	}
	warm := actorStateTint(worldstate.Actor{Opt3State: db.Opt3Warm})
	if warm.R != 255 || warm.G != 255 || warm.B != 255 || warm.A != 255 {
		t.Fatalf("Warmth uses an additive red overlay, not a flat tint = %+v", warm)
	}
	soulLink := actorStateTint(worldstate.Actor{Opt3State: db.Opt3Soullink})
	if soulLink.R != 255 || soulLink.G != 255 || soulLink.B != 255 || soulLink.A != 255 {
		t.Fatalf("Soul Link uses additive silhouettes, not a flat tint: %+v", soulLink)
	}
}

func TestSoulLinkAndEskeToggleLocalNightLighting(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobLinker}
	world.GND = testutil.GNDWithTopHeights(1, 1, func(int, int) [4]float32 { return [4]float32{} })
	world.RSW = &res.RSW{Models: []res.RSWModel{{Filename: "test.rsm", Scale: res.RSWVector3{X: 1, Y: 1, Z: 1}}}}
	world.RSW.Light.Diffuse = [3]float32{1, 1, 1}
	world.RSM = map[string]*res.RSM{"test.rsm": {MainNodeName: "root", Nodes: []res.RSMNode{{
		Name:     "root",
		Matrix:   [9]float32{1, 0, 0, 0, 1, 0, 0, 0, 1},
		Scale:    res.RSMVector3{X: 1, Y: 1, Z: 1},
		Vertices: []res.RSMVector3{{}, {X: 1}, {Z: 1}},
		Faces:    []res.RSMFace{{VertexIndices: [3]uint16{0, 1, 2}}},
	}}}}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}
	mode := &WorldMode{}
	defer mode.Leave()
	drawColors := func() [2][3]float64 {
		t.Helper()
		frame := render.NewFrame(800, 600)
		projection := testutil.Projection(800, 600, 1, 1, 0)
		mode.mapScene.DrawGNDMeshes(frame, nil, world.GND, world.RSW, projection)
		mode.mapScene.DrawRSMModels(frame, nil, world.RSW, world.RSM, world.GND, projection, time.Unix(100, 0))
		meshes := reflect.ValueOf(frame).Elem().FieldByName("worldMeshes")
		if meshes.Len() != 2 {
			t.Fatalf("drawn meshes = %d, want ground and model", meshes.Len())
		}
		var colors [2][3]float64
		for i := range colors {
			vertex := meshes.Index(i).FieldByName("Mesh").Elem().FieldByName("vertices").Index(0)
			for c, name := range []string{"ColorR", "ColorG", "ColorB"} {
				colors[i][c] = vertex.FieldByName(name).Float()
			}
		}
		return colors
	}
	day := drawColors()

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSoullink, ActorID: 2000000, Active: true})
	if !mode.mapScene.Night {
		t.Fatalf("night state=%t", mode.mapScene.Night)
	}
	lighting := mode.mapScene.Lighting(nil)
	if lighting.Diffuse.X != 0.5 || lighting.Diffuse.Y != 0.5 || lighting.Diffuse.Z != 1 || lighting.Env.X != 0.5 || lighting.Env.Y != 0.5 || lighting.Env.Z != 1 {
		t.Fatalf("night lighting = %+v", lighting)
	}
	night := drawColors()
	for i, name := range []string{"ground", "model"} {
		if night[i][0] >= day[i][0] || night[i][1] >= day[i][1] || night[i][2] != day[i][2] {
			t.Errorf("%s mesh did not acquire night lighting: day=%v night=%v", name, day[i], night[i])
		}
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSke, ActorID: 2000000, Active: false})
	if mode.mapScene.Night {
		t.Fatal("inactive Eske did not restore day lighting")
	}
	lighting = mode.mapScene.Lighting(nil)
	if lighting.Diffuse.X != 1 || lighting.Diffuse.Y != 1 || lighting.Diffuse.Z != 1 {
		t.Fatalf("restored lighting = %+v", lighting)
	}
	if restored := drawColors(); restored != day {
		t.Fatalf("meshes did not regain day lighting: got=%v want=%v", restored, day)
	}
}

func TestTaekwonMissionAndRankingMessages(t *testing.T) {
	mode := &WorldMode{}
	mode.applyTaekwonMission(client.Context{}, network.TaekwonMission{MonsterName: "Spore", MonsterID: 1014, Progress: 37, Result: 1})
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Taekwon mission: Spore (37%)" || messages[0].Color != taekwonAnnouncementColor {
		t.Fatalf("mission messages = %+v", messages)
	}

	entries := make([]network.FameRankingEntry, 10)
	entries[0] = network.FameRankingEntry{Name: "Kicker", Points: 123}
	presentation := fameRankingPresentationFor(network.FameRankingTaekwon)
	if presentation.professionMessageID != 2388 || presentation.pointMessageID != 926 {
		t.Fatalf("taekwon message IDs=(%d, %d) want=(2388, 926)", presentation.professionMessageID, presentation.pointMessageID)
	}
	mode.applyFameRanking(client.Context{}, network.FameRanking{Kind: network.FameRankingTaekwon, Entries: entries})
	messages = mode.ui.console.Messages()
	if len(messages) != 12 {
		t.Fatalf("message count = %d, want mission + header + ten ranks", len(messages))
	}
	if messages[1].Text != "=========== Taekwon Rank ===========" || messages[2].Text != "[1] Kicker : 123 Points" || messages[11].Text != "[10] None : 0 Points" {
		t.Fatalf("ranking messages = %+v", messages[1:])
	}

	mode.applyFamePointUpdate(client.Context{}, network.FamePointUpdate{Kind: network.FameRankingTaekwon, GainedPoints: 1, TotalPoints: 42})
	messages = mode.ui.console.Messages()
	if len(messages) != 13 || messages[12].Text != "[POINT] You have been rewarded with 1 Tae-Kwon Mission rank points. Your point total is 42." {
		t.Fatalf("taekwon point message = %+v", messages)
	}
}

func TestStarPlaceRequestOpensConfirmationAndAccepts(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	address := listener.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), address.IP.String(), address.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	ctx := client.Context{Network: netClient, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	packet := network.Packet{ID: network.PacketZCStarPlace, Data: []byte{0x53, 0x02, 0x01}}
	if next, stop := mode.handleNetworkPacket(ctx, packet, time.Now()); next != nil || stop {
		t.Fatalf("star place request changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.starPlaceConfirm.IsOpen() {
		t.Fatal("star place request did not open confirmation")
	}
	if !mode.ui.interactionModalOpen() {
		t.Fatal("star place confirmation did not block world interactions")
	}

	mode.ui.starPlaceConfirm.Confirm(ctx)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	want := network.BuildAgreeStarPlacePacket(1)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(serverConn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("star place reply = %x, want %x", got, want)
	}
}

func TestStarPlaceCancelDoesNotSendAgreement(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Network: network.NewClient(20080910, false), ScreenW: 800, ScreenH: 600}

	mode.applyStarPlaceRequest(ctx, network.StarPlace{Place: 2})
	mode.ui.starPlaceConfirm.Cancel(ctx)

	if mode.ui.starPlaceConfirm.IsOpen() {
		t.Fatal("star place confirmation stayed open after cancel")
	}
	if messages := mode.ui.console.Messages(); len(messages) != 0 {
		t.Fatalf("cancel attempted a network reply: %+v", messages)
	}
}
