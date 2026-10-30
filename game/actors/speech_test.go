package actors

import (
	"testing"
)

func TestSpeechBubbleTextUsesMessagePart(t *testing.T) {
	if got := speechBubbleText("Kivutar : hello there"); got != "hello there" {
		t.Fatalf("speechBubbleText = %q, want hello there", got)
	}
}
