package res

import (
	"fmt"
	"image/color"
	"strings"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

var defaultBookBackground = color.RGBA{R: 245, G: 245, B: 220, A: 255}

type BookContent struct {
	Background color.RGBA
	Lines      []string
}

func BookResourcePath(itemID uint16) string {
	return fmt.Sprintf("data/book/%d.txt", itemID)
}

func (m *Manager) HasBook(itemID uint16) bool {
	return itemID != 0 && m.HasFileExact(BookResourcePath(itemID))
}

func (m *Manager) LoadBook(itemID uint16) (BookContent, error) {
	if m == nil || itemID == 0 {
		return BookContent{}, fmt.Errorf("invalid book item %d", itemID)
	}
	data, err := m.ReadFileExact(BookResourcePath(itemID))
	if err != nil {
		return BookContent{}, err
	}
	return ParseBookContent(data), nil
}

func ParseBookContent(data []byte) BookContent {
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), data)
	if err != nil {
		decoded = data
	}
	text := strings.TrimPrefix(string(decoded), "\ufeff")
	background := defaultBookBackground
	if len(text) >= 7 && text[0] == '%' {
		if parsed, ok := parseBookColor(text[1:7]); ok {
			background = parsed
			text = text[7:]
		}
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return BookContent{
		Background: background,
		Lines:      strings.Split(text, "\n"),
	}
}

func parseBookColor(text string) (color.RGBA, bool) {
	if len(text) != 6 {
		return color.RGBA{}, false
	}
	var rgb [3]uint8
	for i := range rgb {
		var value uint8
		for _, digit := range text[i*2 : i*2+2] {
			value *= 16
			switch {
			case digit >= '0' && digit <= '9':
				value += uint8(digit - '0')
			case digit >= 'a' && digit <= 'f':
				value += uint8(digit-'a') + 10
			case digit >= 'A' && digit <= 'F':
				value += uint8(digit-'A') + 10
			default:
				return color.RGBA{}, false
			}
		}
		rgb[i] = value
	}
	return color.RGBA{R: rgb[0], G: rgb[1], B: rgb[2], A: 255}, true
}
