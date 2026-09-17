package res

import (
	"image"
	"strconv"
	"strings"
)

// WorldMapEntry locates a field/town on worldmap.bmp in source-image pixels.
// Region groups neighboring maps on the same image, not separate continents.
type WorldMapEntry struct {
	MapName string
	Region  int
	Bounds  image.Rectangle
}

// WorldMapEntries returns the cached, read-only legacy mapPosTable.txt table.
func (m *Manager) WorldMapEntries() []WorldMapEntry {
	if m == nil {
		return nil
	}
	if !m.worldMapEntriesLoaded {
		m.worldMapEntriesLoaded = true
		if _, data, ok := m.ReadFirst([]string{"data/mapPosTable.txt", "mapPosTable.txt"}); ok {
			m.worldMapEntries = parseWorldMapEntries(data)
		}
	}
	return m.worldMapEntries
}

func parseWorldMapEntries(data []byte) []WorldMapEntry {
	var entries []WorldMapEntry
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimPrefix(string(data), "\ufeff"), "\n") {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Split(strings.TrimSpace(line), "#")
		// Skip the region-count header ("12@"), comments, and incomplete rows.
		if len(fields) < 6 {
			continue
		}
		var values [5]int
		valid := true
		for i, column := range []int{0, 2, 3, 4, 5} {
			value, err := strconv.Atoi(strings.TrimSpace(fields[column]))
			if err != nil || value < 0 {
				valid = false
				break
			}
			values[i] = value
		}
		name := strings.ToLower(strings.TrimSpace(fields[1]))
		name = strings.TrimSuffix(name, ".rsw")
		if !valid || name == "" || strings.ContainsAny(name, `/\`) || seen[name] || values[3] <= values[1] || values[4] <= values[2] {
			continue
		}
		entries = append(entries, WorldMapEntry{MapName: name, Region: values[0], Bounds: image.Rect(values[1], values[2], values[3], values[4])})
		seen[name] = true
	}
	return entries
}
