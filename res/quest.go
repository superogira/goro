package res

import (
	"strconv"
	"strings"
)

// QuestMetadata is the six-column, pre-renewal questid2display.txt entry.
// Objective counts and deadlines come from the server, not this table.
type QuestMetadata struct {
	Title       string
	Icon        string
	Image       string
	Description string
	Summary     string
}

func (m *Manager) QuestMetadata(id uint32) (QuestMetadata, bool) {
	if m == nil {
		return QuestMetadata{}, false
	}
	if !m.questMetadataLoaded {
		m.questMetadataLoaded = true
		if _, data, ok := m.ReadFirst([]string{"data/questid2display.txt", "questid2display.txt"}); ok {
			m.questMetadata = parseQuestMetadata(data)
		}
	}
	entry, ok := m.questMetadata[id]
	return entry, ok
}

func parseQuestMetadata(data []byte) map[uint32]QuestMetadata {
	text := strings.TrimPrefix(decodeGRFName(data), "\ufeff")
	// Comments can contain '#'; remove comment lines before splitting fields.
	var table strings.Builder
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			table.WriteString(line)
			table.WriteByte('\n')
		}
	}
	fields := strings.Split(table.String(), "#")
	entries := make(map[uint32]QuestMetadata)
	for i := 0; i+6 < len(fields); i += 6 {
		id, err := strconv.ParseUint(strings.TrimSpace(fields[i]), 10, 32)
		if err != nil {
			continue
		}
		entries[uint32(id)] = QuestMetadata{
			Title: strings.TrimSpace(fields[i+1]), Icon: strings.TrimSpace(fields[i+2]),
			Image: strings.TrimSpace(fields[i+3]), Description: strings.TrimSpace(fields[i+4]),
			Summary: strings.TrimSpace(fields[i+5]),
		}
	}
	return entries
}
