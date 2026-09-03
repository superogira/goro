package res

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

func TestParseBookContentColorAndLines(t *testing.T) {
	book := ParseBookContent([]byte("%F5F5DC^000088First line\r\nsecond line"))
	if book.Background != (color.RGBA{R: 0xF5, G: 0xF5, B: 0xDC, A: 255}) {
		t.Fatalf("background = %#v", book.Background)
	}
	want := []string{"^000088First line", "second line"}
	if !reflect.DeepEqual(book.Lines, want) {
		t.Fatalf("lines = %#v, want %#v", book.Lines, want)
	}
}

func TestParseBookContentDecodesEUCKR(t *testing.T) {
	encoded, _, err := transform.Bytes(korean.EUCKR.NewEncoder(), []byte("책 내용"))
	if err != nil {
		t.Fatal(err)
	}
	book := ParseBookContent(encoded)
	if got := book.Lines; !reflect.DeepEqual(got, []string{"책 내용"}) {
		t.Fatalf("lines = %#v", got)
	}
}

func TestParseBookContentKeepsMalformedHeaderAsText(t *testing.T) {
	book := ParseBookContent([]byte("%ZZZZZZbroken"))
	if book.Background != defaultBookBackground {
		t.Fatalf("background = %#v, want default %#v", book.Background, defaultBookBackground)
	}
	if got := book.Lines; !reflect.DeepEqual(got, []string{"%ZZZZZZbroken"}) {
		t.Fatalf("lines = %#v", got)
	}
}

func TestManagerBookRequiresExactResource(t *testing.T) {
	m := &Manager{}
	if m.HasBook(7277) {
		t.Fatal("missing book reported as present")
	}
	if _, err := m.LoadBook(7277); err == nil {
		t.Fatal("missing book loaded without an error")
	}
}

func TestManagerLoadsLooseBookResource(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "book", "7277.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("%FFFFFFhello"), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Root: root}
	if !m.HasBook(7277) {
		t.Fatal("loose book resource was not detected")
	}
	book, err := m.LoadBook(7277)
	if err != nil {
		t.Fatal(err)
	}
	if got := book.Lines; !reflect.DeepEqual(got, []string{"hello"}) {
		t.Fatalf("lines = %#v", got)
	}
}

func TestManagerLoadsBookFromRealClientData(t *testing.T) {
	m := realDataManager(t)
	for _, archive := range m.Archives {
		for _, name := range archive.Names() {
			normalized := strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
			if !strings.HasPrefix(normalized, "data/book/") || !strings.HasSuffix(normalized, ".txt") {
				continue
			}
			itemID, err := strconv.ParseUint(strings.TrimSuffix(filepath.Base(normalized), ".txt"), 10, 16)
			if err != nil {
				continue
			}
			book, err := m.LoadBook(uint16(itemID))
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			if len(book.Lines) == 0 {
				t.Fatalf("%s parsed without lines", name)
			}
			t.Logf("loaded book resource %s", name)
			return
		}
	}
	t.Skip("client data has no numeric data/book/*.txt resource")
}
