package audio

import (
	"path/filepath"
	"strings"
)

// Path candidate helpers are intentionally untagged: gameplay code prefetches
// sound paths on every platform, while the decoder itself only exists in
// nofakecgo builds.

func normalizeSFXPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.ReplaceAll(path, "/", "\\")
	path = strings.TrimPrefix(path, ".\\")
	path = strings.TrimPrefix(path, "data\\")
	if path == "" {
		return ""
	}
	if filepath.Ext(path) == "" {
		path += ".wav"
	}
	return path
}

func sfxPathCandidates(path string) []string {
	normalized := normalizeSFXPath(path)
	if normalized == "" {
		return nil
	}
	slash := strings.ReplaceAll(normalized, "\\", "/")
	lower := strings.ToLower(normalized)
	// Lead with data\wav\ — the canonical kRO location and where the web pack
	// stores sounds — so lookups hit the archive before any HTTP probe.
	var candidates []string
	if strings.HasPrefix(lower, "wav\\") {
		candidates = append(candidates,
			"data\\"+normalized,
			"data/"+slash,
			normalized,
			slash,
		)
	} else {
		candidates = append(candidates,
			"data\\wav\\"+normalized,
			"data/wav/"+slash,
			normalized,
			slash,
			"wav\\"+normalized,
			"wav/"+slash,
		)
	}
	return uniquePathCandidates(candidates)
}

func uniquePathCandidates(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}

// SFXPathCandidates exposes the wav location candidates for a sound name so
// other packages can prefetch exactly the files playback will read.
func SFXPathCandidates(path string) []string {
	return sfxPathCandidates(path)
}
