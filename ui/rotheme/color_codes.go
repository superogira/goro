package rotheme

import (
	"image/color"
	"strconv"
	"strings"
)

// ParseColorCode reads a leading RO ^RRGGBB text color.
func ParseColorCode(text string) (color.RGBA, bool) {
	if len(text) < 7 || text[0] != '^' {
		return color.RGBA{}, false
	}
	for _, c := range text[1:7] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return color.RGBA{}, false
		}
	}
	value, _ := strconv.ParseUint(text[1:7], 16, 32)
	return color.RGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}, true
}

func StripColorCodes(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		if _, ok := ParseColorCode(text[i:]); ok {
			i += 7
		} else {
			out.WriteByte(text[i])
			i++
		}
	}
	return out.String()
}
