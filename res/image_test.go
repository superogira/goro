package res

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestNPCCutinTextureCandidatesUseROIllustrationPath(t *testing.T) {
	got := NPCCutinTextureCandidates("guide")
	want := "data\\texture\\유저인터페이스\\illust\\guide.bmp"
	if len(got) == 0 || got[0] != want {
		t.Fatalf("candidates = %q, want first %q", got, want)
	}
	for _, candidate := range got {
		if strings.HasSuffix(candidate, ".bmp.bmp") {
			t.Fatalf("candidate has duplicate extension: %q", candidate)
		}
	}

	withExtension := NPCCutinTextureCandidates("guide.BMP")
	if len(withExtension) == 0 || withExtension[0] != "data\\texture\\유저인터페이스\\illust\\guide.BMP" {
		t.Fatalf("extended candidates = %q", withExtension)
	}

	untrusted := NPCCutinTextureCandidates("../../guide")
	for _, candidate := range untrusted {
		if strings.Contains(candidate, "..") {
			t.Fatalf("cut-in candidate retained path traversal: %q", candidate)
		}
	}
	if got := NPCCutinTextureCandidates(".."); len(got) != 0 {
		t.Fatalf("parent-directory cut-in candidates = %q, want none", got)
	}
}

func TestApplyROTransparency(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 120, G: 40, B: 200, A: 255})

	out := applyROTransparency(img)
	if got := out.NRGBAAt(0, 0).A; got != 0 {
		t.Fatalf("magenta alpha = %d, want 0", got)
	}
	if got := out.NRGBAAt(1, 0).A; got != 255 {
		t.Fatalf("non-key alpha = %d, want 255", got)
	}
}

func TestApplyROTransparencyClearsTransparentRGB(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 128, B: 0, A: 0})

	out := applyROTransparency(img)
	for x := 0; x < 2; x++ {
		got := out.NRGBAAt(x, 0)
		if got != (color.NRGBA{}) {
			t.Fatalf("transparent pixel %d = %+v, want fully zero", x, got)
		}
	}
}

func TestApplyEffectTransparencyClearsTransparentRGB(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 1, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 120, G: 40, B: 200, A: 255})

	out := ApplyEffectTransparency(img)
	if got := out.NRGBAAt(0, 0); got != (color.NRGBA{}) {
		t.Fatalf("near-black key pixel = %+v, want fully zero", got)
	}
	if got := out.NRGBAAt(1, 0); got != (color.NRGBA{R: 120, G: 40, B: 200, A: 255}) {
		t.Fatalf("non-key pixel = %+v, want original", got)
	}
}

func TestApplyEffectTransparencyCanPreserveBlackKey(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 1, G: 2, B: 1, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, G: 0, B: 255, A: 255})

	out := ApplyEffectTransparencyWithBlackKey(img, false)
	if got := out.NRGBAAt(0, 0); got != (color.NRGBA{R: 1, G: 2, B: 1, A: 255}) {
		t.Fatalf("near-black pixel = %+v, want preserved", got)
	}
	if got := out.NRGBAAt(1, 0); got != (color.NRGBA{}) {
		t.Fatalf("magenta key pixel = %+v, want fully zero", got)
	}
}
