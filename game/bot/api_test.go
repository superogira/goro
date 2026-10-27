package bot

import (
	"testing"

	"github.com/kivutar/goro/client"
)

func TestScriptMessageRejectsInvalidMessages(t *testing.T) {
	for _, message := range []string{"", "   ", "hello", "%", "$", "/w Alice"} {
		if sendMessage(client.Context{}, message) {
			t.Fatalf("scriptMessage(%q) = true, want false", message)
		}
	}
}
