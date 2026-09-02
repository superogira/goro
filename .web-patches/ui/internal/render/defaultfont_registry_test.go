package render

import (
	"os"
	"testing"

	"github.com/gogpu/ui/theme/font"
)

// The plain DrawText path (used by every widget when its canvas lacks
// StyledTextDrawer) must resolve its default font through the global
// FontRegistry, so registering a Thai-capable replacement under the
// default family name takes effect there too.
func TestEnsureDefaultFontsResolvesRegistryReplacement(t *testing.T) {
	data, err := os.ReadFile("testdata/Sarabun-Regular.ttf")
	if err != nil {
		t.Skipf("test font missing: %v", err)
	}
	if err := GlobalFontRegistry().Register(defaultFontFamily, font.Regular, font.Normal, data); err != nil {
		t.Fatalf("register replacement: %v", err)
	}
	ensureDefaultFonts() // idempotent via the package own sync.Once

	want := GlobalFontRegistry().Resolve(defaultFontFamily, font.Regular, font.Normal)
	if want == nil {
		t.Fatal("registry could not resolve default family")
	}
	if defaultRegular != want {
		t.Fatal("plain DrawText default font is not the registry-resolved replacement — tofu fallback for non-Latin scripts")
	}
}
