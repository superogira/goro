//go:build !js || !wasm

package res

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// candidatePaths returns the on-disk paths to probe for a normalized
// resource name, in priority order.
func (m *Manager) candidatePaths(normalized string) []string {
	seen := make(map[string]struct{}, 3)
	var out []string
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	add(filepath.Join(m.Root, normalized))
	add(filepath.Join(m.Root, strings.ReplaceAll(normalized, "\\", string(filepath.Separator))))
	add(filepath.Join(m.Root, strings.ReplaceAll(normalized, "/", string(filepath.Separator))))
	return withCharsetVariants(out)
}

// candidateExists reports whether a candidate path refers to a regular file.
func (m *Manager) candidateExists(candidate string) bool {
	stat, err := os.Stat(candidate)
	return err == nil && !stat.IsDir()
}

// readCandidate returns the contents of a candidate path returned by Find.
func (m *Manager) readCandidate(candidate string) ([]byte, error) {
	return os.ReadFile(candidate)
}

// scanArchives loads GRF archives found under Root. The web build serves the
// extracted data directory over HTTP and has no archive support.
func (m *Manager) scanArchives() {
	archivePaths := make([]string, 0)
	seen := make(map[string]struct{})
	if entries, err := os.ReadDir(m.Root); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".grf" && ext != ".gpf" {
				continue
			}
			path := filepath.Join(m.Root, name)
			archivePaths = append(archivePaths, path)
			seen[strings.ToLower(path)] = struct{}{}
		}
	}
	for _, name := range []string{"data.grf", "rdata.grf", "fdata.grf", "event.grf"} {
		path := filepath.Join(m.Root, name)
		if _, ok := seen[strings.ToLower(path)]; ok {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		archivePaths = append(archivePaths, path)
	}
	sortStringsStable(archivePaths)
	for _, path := range archivePaths {
		archive, err := OpenGRF(path)
		if err != nil {
			continue
		}
		m.Archives = append(m.Archives, archive)
	}
}

func sortStringsStable(values []string) {
	sort.SliceStable(values, func(i, j int) bool {
		return archivePriority(values[i]) < archivePriority(values[j])
	})
}
