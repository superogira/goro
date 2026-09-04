// Package buildinfo carries version metadata injected at build time and
// shown as a small badge in the bottom-right corner of the game window.
package buildinfo

// Version and BuildTime are set via ldflags so every artifact reports what
// it actually is:
//
//	go build -ldflags "-X github.com/kivutar/goro/buildinfo.Version=v85 \
//	  -X 'github.com/kivutar/goro/buildinfo.BuildTime=2026-09-04 18:00'"
//
// Uninjected builds fall back to "dev".
var (
	Version   = "dev"
	BuildTime = ""
)

// Label formats the corner badge text.
func Label() string {
	if BuildTime != "" {
		return Version + " · " + BuildTime
	}
	return Version
}
