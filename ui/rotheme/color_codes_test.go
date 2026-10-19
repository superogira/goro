package rotheme

import (
	"image/color"
	"testing"
)

func TestROColorCodes(t *testing.T) {
	if got, ok := ParseColorCode("^aB12fFtext"); !ok || got != (color.RGBA{R: 0xab, G: 0x12, B: 0xff, A: 255}) {
		t.Fatalf("parsed color = %v, ok=%t", got, ok)
	}
	for _, text := range []string{"^12345", "^12gg00", "^+12345", "^é1234", "ff0000"} {
		if _, ok := ParseColorCode(text); ok {
			t.Errorf("accepted invalid color code %q", text)
		}
	}
	if got := StripColorCodes("é^33cc00Active^000000 ^badXYZ"); got != "éActive ^badXYZ" {
		t.Fatalf("stripped text = %q", got)
	}
}
