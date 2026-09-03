package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kivutar/goro/res"
)

func TestWrapBookTextLinesPreservesParagraphsAndColor(t *testing.T) {
	lines := wrapBookTextLines([]string{"^112233alpha beta gamma", "", "omega"})
	if len(lines) != 3 {
		t.Fatalf("wrapped lines = %d, want 3", len(lines))
	}
	if got := itemInfoLinePlainText(lines[0]); got != "alpha beta gamma" {
		t.Fatalf("first line = %q", got)
	}
	if len(lines[0].Runs) != 1 || lines[0].Runs[0].Color.R != 0x11 {
		t.Fatalf("first line color = %#v", lines[0].Runs)
	}
	if got := itemInfoLinePlainText(lines[1]); got != "" {
		t.Fatalf("blank line = %q", got)
	}
}

func TestBookWindowPaginationAndTitle(t *testing.T) {
	window := BookWindow{title: "The Book of Ymir"}
	for i := 0; i < bookWindowLinesPerPage+1; i++ {
		window.lines = append(window.lines, parseItemInfoTextLine(fmt.Sprintf("line %d", i)))
	}
	window.totalPages = 2
	if got := len(window.pageLines()); got != bookWindowLinesPerPage {
		t.Fatalf("first page lines = %d", got)
	}
	if got := window.windowTitle(); got != "The Book of Ymir  (1/2)" {
		t.Fatalf("title = %q", got)
	}
	window.page = 1
	if got := len(window.pageLines()); got != 1 {
		t.Fatalf("second page lines = %d", got)
	}
}

func TestEmptyBookHasReadableFallback(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "data", "book", "7277.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	window := BookWindow{}
	if err := window.Open(Context{Resources: &res.Manager{Root: root}, ScreenW: 800, ScreenH: 600}, 7277, "Empty Book"); err != nil {
		t.Fatal(err)
	}
	if len(window.lines) != 1 || itemInfoLinePlainText(window.lines[0]) != "This book is empty." {
		t.Fatalf("empty book lines = %#v", window.lines)
	}
}

func TestWrapBookTextLinesSplitsLongUnbrokenText(t *testing.T) {
	lines := wrapBookTextLines([]string{strings.Repeat("x", bookWindowLineRunes+3)})
	if len(lines) != 2 {
		t.Fatalf("wrapped lines = %d, want 2", len(lines))
	}
	if got := runeLen(itemInfoLinePlainText(lines[0])); got != bookWindowLineRunes {
		t.Fatalf("first line runes = %d, want %d", got, bookWindowLineRunes)
	}
}
