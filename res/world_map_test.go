package res

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorldMapEntries(t *testing.T) {
	data := "\ufeff// region#map#coordinates\r\n12@\r\n0#HUGEL.RSW#871#0#927#57#\r\n8#prontera.rsw#812#587#870#643# // town\n" +
		"8#prontera.rsw#1#2#3#4#\n1#bad.rsw#0#0#oops#10#\n0#flat.rsw#2#3#2#4#\n0#backward.rsw#4#4#2#2#\n" +
		"0#negative.rsw#-1#0#2#2#\n0#../escape.rsw#0#0#2#2#\n0#short#0#0#\n"
	entries := parseWorldMapEntries([]byte(data))
	if len(entries) != 2 || entries[0].MapName != "hugel" || entries[1].Region != 8 || entries[1].Bounds != image.Rect(812, 587, 870, 643) {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestWorldMapEntriesCached(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "mapPosTable.txt")
	if err := os.WriteFile(path, []byte("0#prontera.rsw#1#2#3#4#"), 0600); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Root: root}
	entries := m.WorldMapEntries()
	if len(entries) != 1 {
		t.Fatal("map table not loaded")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if &m.WorldMapEntries()[0] != &entries[0] {
		t.Fatal("map table was not cached")
	}
	if (*Manager)(nil).WorldMapEntries() != nil {
		t.Fatal("nil manager returned maps")
	}
}

func TestWorldMapRealArchiveWhenConfigured(t *testing.T) {
	m := realDataManager(t)
	entries := m.WorldMapEntries()
	if len(entries) < 100 {
		t.Fatalf("only %d world maps", len(entries))
	}
	img, _, err := LoadImage(m, []string{"data/texture/유저인터페이스/worldmap.bmp"})
	if err != nil {
		t.Fatal(err)
	}
	foundProntera := false
	for _, entry := range entries {
		if !entry.Bounds.In(img.Bounds()) {
			t.Errorf("%s outside map: %v", entry.MapName, entry.Bounds)
		}
		if entry.MapName == "prontera" {
			foundProntera = true
		}
	}
	if !foundProntera {
		t.Fatal("Prontera missing")
	}
	t.Logf("%d maps on %v artwork", len(entries), img.Bounds().Size())
}
