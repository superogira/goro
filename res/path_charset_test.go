package res

import (
	"testing"

	"golang.org/x/text/encoding/korean"
)

func TestEuckrVariantDecodesRawMapPaths(t *testing.T) {
	encoded, err := korean.EUCKR.NewEncoder().Bytes([]byte("texture\\필드바닥\\prt_흙03.bmp"))
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	got := euckrVariant(string(encoded))
	want := "texture\\필드바닥\\prt_흙03.bmp"
	if got != want {
		t.Fatalf("euckrVariant = %q, want %q", got, want)
	}
}

func TestEuckrVariantLeavesUtf8Alone(t *testing.T) {
	for _, path := range []string{
		"",
		"data/sprite/cursors.spr",
		"data\\texture\\유저인터페이스\\bgi_temp.bmp",
	} {
		if got := euckrVariant(path); got != "" {
			t.Fatalf("euckrVariant(%q) = %q, want empty", path, got)
		}
	}
}

func TestWithCharsetVariantsAppendsDecodedCopies(t *testing.T) {
	encoded, err := korean.EUCKR.NewEncoder().String("model\\외부소품\\다리.rsm")
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	got := withCharsetVariants([]string{"data\\cursors.spr", encoded})
	want := []string{"data\\cursors.spr", encoded, "model\\외부소품\\다리.rsm"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
