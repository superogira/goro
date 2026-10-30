package social

import (
	"bytes"
	"image"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func TestGuildMemberPortraitRealData(t *testing.T) {
	ctx := client.Context{Resources: testutil.RealDataManager(t)}
	m := &System{}
	member := session.GuildMember{Job: 1, HeadType: 1, Sex: 1, CurrentState: 1}
	first := m.GuildMemberPortrait(ctx, member)
	if first == nil || first.Bounds() != image.Rect(0, 0, 24, 24) {
		t.Fatalf("portrait = %v", first)
	}
	visible := false
	for y := 0; y < 24; y++ {
		for x := 0; x < 24; x++ {
			_, _, _, alpha := first.At(x, y).RGBA()
			visible = visible || alpha != 0
		}
	}
	if !visible {
		t.Fatal("portrait is empty")
	}
	if m.GuildMemberPortrait(ctx, member) != first {
		t.Fatal("unchanged appearance was not cached")
	}
	member.HeadType = 2
	second := m.GuildMemberPortrait(ctx, member)
	if second == nil || bytes.Equal(first.(*image.RGBA).Pix, second.(*image.RGBA).Pix) {
		t.Fatal("hairstyle change did not change portrait")
	}
	member.CurrentState = 0
	if m.GuildMemberPortrait(ctx, member) != nil {
		t.Fatal("offline member showed an online portrait")
	}
	member.CurrentState = 1
	ctx.Resources = &res.Manager{Root: t.TempDir()}
	if m.GuildMemberPortrait(ctx, member) != nil {
		t.Fatal("portrait reused after changing client assets")
	}
}
