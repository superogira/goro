package fontdata

import (
	"testing"

	"golang.org/x/image/font/opentype"
)

func TestInterFonts(t *testing.T) {
	for name, data := range map[string][]byte{
		"regular": InterRegular,
		"bold":    InterBold,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := opentype.Parse(data); err != nil {
				t.Fatalf("parse embedded font: %v", err)
			}
		})
	}
}
