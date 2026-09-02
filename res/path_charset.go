package res

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
)

// euckrVariant returns the UTF-8 decoding of path when the path carries raw
// EUC-KR (CP949) bytes. GRF-era data files embed paths in that encoding —
// GND texture names, RSW model filenames, RSM texture names — while loose
// file trees extracted for the web build use UTF-8 names. A path that is
// already valid UTF-8, or that does not decode cleanly, has no variant.
func euckrVariant(path string) string {
	if path == "" || utf8.ValidString(path) {
		return ""
	}
	decoded, err := korean.EUCKR.NewDecoder().String(path)
	if err != nil || decoded == "" || decoded == path {
		return ""
	}
	if strings.ContainsRune(decoded, utf8.RuneError) {
		return ""
	}
	return decoded
}

// withCharsetVariants appends the EUC-KR-decoded form of each candidate so
// byte-oriented paths from map files also resolve against UTF-8-named files.
// Candidates keep their relative order; decoded copies follow their source.
func withCharsetVariants(candidates []string) []string {
	out := make([]string, 0, len(candidates)*2)
	for _, candidate := range candidates {
		out = append(out, candidate)
		if variant := euckrVariant(candidate); variant != "" {
			out = append(out, variant)
		}
	}
	return out
}
