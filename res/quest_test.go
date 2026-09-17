package res

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/korean"
)

func TestParseQuestMetadata(t *testing.T) {
	data := "\ufeff// comment with # delimiters\r\n1001#Acolyte#SG_FEEL#QUE_NOIMAGE#\r\nTalk to ^000077Father^000000.\r\nThen return.#Find Father.#\r\n1002#Other####Summary only.#\ninvalid#skip#####\n1003#truncated#"
	entries := parseQuestMetadata([]byte(data))
	if len(entries) != 2 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[1001] != (QuestMetadata{Title: "Acolyte", Icon: "SG_FEEL", Image: "QUE_NOIMAGE", Description: "Talk to ^000077Father^000000.\nThen return.", Summary: "Find Father."}) {
		t.Fatalf("quest = %+v", entries[1001])
	}
	if entries[1002].Summary != "Summary only." || entries[1002].Description != "" {
		t.Fatalf("empty fields shifted: %+v", entries[1002])
	}
	encoded, err := korean.EUCKR.NewEncoder().Bytes([]byte("1#퀘스트###설명##"))
	if err != nil {
		t.Fatal(err)
	}
	if got := parseQuestMetadata(encoded)[1].Title; got != "퀘스트" {
		t.Fatalf("Korean title = %q", got)
	}
}

func TestQuestMetadataLoadsOnce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "questid2display.txt")
	if err := os.WriteFile(path, []byte("1001#Acolyte#####"), 0600); err != nil {
		t.Fatal(err)
	}
	m := &Manager{Root: root}
	if entry, ok := m.QuestMetadata(1001); !ok || entry.Title != "Acolyte" {
		t.Fatalf("quest = %+v, %v", entry, ok)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.QuestMetadata(1001); !ok {
		t.Fatal("metadata was not cached")
	}
	if _, ok := m.QuestMetadata(9000); ok {
		t.Fatal("unknown quest was found")
	}
	if _, ok := (*Manager)(nil).QuestMetadata(1); ok {
		t.Fatal("nil resources found a quest")
	}
}

func TestQuestMetadataRealArchiveWhenConfigured(t *testing.T) {
	m := realDataManager(t)
	meta, ok := m.QuestMetadata(1001)
	if !ok || meta.Title == "" || meta.Description == "" {
		t.Fatalf("Acolyte quest = %+v, %v", meta, ok)
	}
	for _, name := range []string{meta.Icon, meta.Image} {
		img, source, err := LoadImage(m, ItemIconTextureCandidates(name))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%s: %v", source, img.Bounds())
	}
}
