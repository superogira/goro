package ui

import "strings"

// itemInfoWebDescHTML converts the description's ^RRGGBB color codes into
// HTML spans, preserving the game's intended highlighting.
func itemInfoWebDescHTML(text string) string {
	runes := []rune(text)
	var b strings.Builder
	active := false
	for i := 0; i < len(runes); i++ {
		if runes[i] == '^' && i+6 < len(runes) && isHexRunes(runes[i+1:i+7]) {
			if active {
				b.WriteString("</span>")
			}
			b.WriteString(`<span style="color:#`)
			for _, r := range runes[i+1 : i+7] {
				b.WriteRune(r)
			}
			b.WriteString(`">`)
			active = true
			i += 6
			continue
		}
		b.WriteRune(runes[i])
	}
	if active {
		b.WriteString("</span>")
	}
	return b.String()
}

func isHexRunes(runes []rune) bool {
	if len(runes) != 6 {
		return false
	}
	for _, r := range runes {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}
