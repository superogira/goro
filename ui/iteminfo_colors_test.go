package ui

import "testing"

func TestItemInfoWebDescHTMLColorCodes(t *testing.T) {
	got := itemInfoWebDescHTML("^777777Gray^000000 normal")
	want := `<span style="color:#777777">Gray</span><span style="color:#000000"> normal</span>`
	if got != want {
		t.Fatalf("descHTML = %q, want %q", got, want)
	}
	// Codes shorter than 6 hex digits stay literal.
	if got := itemInfoWebDescHTML("100% + 1^12"); got != "100% + 1^12" {
		t.Fatalf("short code mangled: %q", got)
	}
}
