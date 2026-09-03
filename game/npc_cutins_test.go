package game

import (
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"golang.org/x/image/bmp"
)

func TestApplyNPCCutinLoadsAndClearsIllustration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "texture", "유저인터페이스", "illust", "guide.bmp")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	source := image.NewNRGBA(image.Rect(0, 0, 18, 24))
	source.Set(4, 5, color.NRGBA{R: 80, G: 120, B: 160, A: 255})
	if err := bmp.Encode(file, source); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	mode := NewWorldMode()
	ctx := client.Context{Resources: &res.Manager{Root: root}}
	if err := mode.applyNPCCutin(ctx, network.NPCCutin{Image: "guide", Position: network.NPCCutinRight}); err != nil {
		t.Fatal(err)
	}
	if !mode.ui.npcCutin.Visible() {
		t.Fatal("loaded cut-in is not visible")
	}

	if err := mode.applyNPCCutin(ctx, network.NPCCutin{Position: network.NPCCutinClear}); err != nil {
		t.Fatal(err)
	}
	if mode.ui.npcCutin.Visible() {
		t.Fatal("clear packet left cut-in visible")
	}
}

func TestMapAndCharacterTransitionsClearNPCCutins(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	mode.startMapFadeOut(network.MapChange{MapName: "geffen"}, testTime())
	if mode.ui.npcCutin.Visible() {
		t.Fatal("map transition left cut-in visible")
	}

	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	mode.startCharacterSelectFadeOut(testTime())
	if mode.ui.npcCutin.Visible() {
		t.Fatal("character transition left cut-in visible")
	}
}

func TestNPCCutinBlocksMapPointerOnlyWhileUIIsVisible(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinRight}, render.NewImage(20, 30))
	inputState := input.NewState()
	inputState.MouseX = 95
	inputState.MouseY = 75
	ctx := client.Context{Input: inputState, ScreenW: 100, ScreenH: 80}
	if !mode.mapPointerBlocked(ctx) {
		t.Fatal("visible cut-in did not block map pointer")
	}

	ctx.Config = config.Config{Render: config.RenderConfig{NoUI: true}}
	if mode.mapPointerBlocked(ctx) {
		t.Fatal("hidden cut-in blocked map pointer with no-UI rendering")
	}
}

func TestNPCDialogCloseCallbackClearsCutinBetweenWorldUpdates(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Hello"})
	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		UIManager: &worldModeTestUIManager{},
		ScreenW:   800,
		ScreenH:   600,
	}
	if !mode.ui.npcDialog.Update(ctx) {
		t.Fatal("dialog did not publish")
	}
	// gogpu/ui button callbacks are dispatched after the coarse Window.Update
	// hit test. Simulate Cancel closing the dialog between world updates.
	mode.ui.npcDialog.Reset()
	if mode.ui.npcDialog.IsOpen() {
		t.Fatal("dialog remained open")
	}
	if mode.ui.npcCutin.Visible() {
		t.Fatal("dialog close left cut-in visible")
	}
}

func TestDisconnectPacketClearsNPCCutins(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	ctx := client.Context{ScreenW: 800, ScreenH: 600}
	packet := network.Packet{ID: network.PacketSCNotifyBan, Data: []byte{0x81, 0x00, 15}}
	mode.handleNetworkPacket(ctx, packet, testTime())
	if mode.ui.npcCutin.Visible() {
		t.Fatal("disconnect packet left cut-in visible")
	}
}

func TestDialogClearOnlyClearsCutinForActiveDialog(t *testing.T) {
	mode := NewWorldMode()
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Hello"})
	mode.ui.npcCutin.Apply(network.NPCCutin{Image: "guide", Position: network.NPCCutinLeft}, render.NewImage(20, 20))
	ctx := client.Context{ScreenW: 800, ScreenH: 600}

	mode.handleNetworkPacket(ctx, npcDialogClearPacket(20), testTime())
	if !mode.ui.npcCutin.Visible() {
		t.Fatal("clear for another NPC removed the active dialog cut-in")
	}

	mode.handleNetworkPacket(ctx, npcDialogClearPacket(10), testTime())
	if mode.ui.npcCutin.Visible() {
		t.Fatal("clear for the active NPC left its cut-in visible")
	}
}

func npcDialogClearPacket(npcID uint32) network.Packet {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[0:2], 0x08D6)
	binary.LittleEndian.PutUint32(data[2:6], npcID)
	return network.Packet{ID: 0x08D6, Data: data}
}

func testTime() time.Time {
	return time.Unix(1, 0)
}
