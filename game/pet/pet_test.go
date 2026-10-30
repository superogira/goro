package pet

import (
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	worldstate "github.com/kivutar/goro/world"
)

func TestPetTalkNumberMatchesROBrowserLayout(t *testing.T) {
	if got := petTalkNumber(1063, petTalkFeeding, 1); got != 1063010 {
		t.Fatalf("petTalkNumber = %d, want 1063010", got)
	}
}

func TestPetHungryStateMatchesROBrowserBuckets(t *testing.T) {
	cases := []struct {
		fullness int
		want     int
	}{
		{0, 0},
		{10, 0},
		{11, 1},
		{25, 1},
		{26, 2},
		{75, 2},
		{76, 3},
		{90, 3},
		{91, 4},
		{100, 4},
	}
	for _, tc := range cases {
		if got := petHungryState(tc.fullness); got != tc.want {
			t.Fatalf("petHungryState(%d) = %d, want %d", tc.fullness, got, tc.want)
		}
	}
}

func TestPetFriendlyStateMatchesROBrowserBuckets(t *testing.T) {
	cases := []struct {
		relationship int
		want         int
	}{
		{0, 0},
		{100, 0},
		{101, 1},
		{250, 1},
		{251, 2},
		{750, 2},
		{751, 3},
		{900, 3},
		{901, 4},
		{1000, 4},
	}
	for _, tc := range cases {
		if got := petFriendlyState(tc.relationship); got != tc.want {
			t.Fatalf("petFriendlyState(%d) = %d, want %d", tc.relationship, got, tc.want)
		}
	}
}

func TestPetEmotionTableMatchesROBrowserFeedingEntries(t *testing.T) {
	if got := petEmotion(0, 0, petTalkFeeding); got != 32 {
		t.Fatalf("hungry ashamed feeding emotion = %d, want 32", got)
	}
	if got := petEmotion(4, 4, petTalkFeeding); got != 20 {
		t.Fatalf("full familiar feeding emotion = %d, want 20", got)
	}
	if got := petEmotion(0, 0, 99); got != -1 {
		t.Fatalf("out of range emotion = %d, want -1", got)
	}
}

func TestPetTalkJobUsesActorJobBeforePropertyJob(t *testing.T) {
	mode := &System{}
	mode.ID = 123
	mode.Property = network.PetProperty{Job: 271}
	ctx := client.Context{World: &worldstate.World{Actors: map[uint32]worldstate.Actor{
		123: {ID: 123, Job: 1063},
	}}}
	if got := mode.talkJob(ctx); got != 1063 {
		t.Fatalf("petTalkJob = %d, want actor job 1063", got)
	}
}
