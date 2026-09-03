package res

import (
	"testing"
)

func TestEffectTextureRealArchiveWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	img, source, err := LoadImage(manager, EffectTextureCandidates("ring_blue"))
	if err != nil {
		t.Fatal(err)
	}
	img = ApplyEffectTransparency(img)
	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Fatalf("invalid %s bounds %v", source, img.Bounds())
	}
	t.Logf("decoded %s bounds=%v", source, img.Bounds())
}

func TestNPCCutinTextureRealArchiveWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	img, source, err := LoadImage(manager, NPCCutinTextureCandidates("kafra_06"))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Empty() {
		t.Fatalf("invalid %s bounds %v", source, img.Bounds())
	}
	t.Logf("decoded %s bounds=%v", source, img.Bounds())
}

func TestWeatherCloudTexturesKeepVisibleAlphaWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	for _, name := range []string{"fog1", "fog2", "fog3", "cloud1", "cloud2", "cloud4"} {
		img, source, err := LoadImageExact(manager, EffectTextureCandidates(name))
		if err != nil {
			t.Fatal(err)
		}
		out := ApplyEffectTransparencyWithBlackKey(img, false)
		visible := 0
		alphaSum := 0
		redSum := 0
		greenSum := 0
		blueSum := 0
		for i := 3; i < len(out.Pix); i += 4 {
			alpha := int(out.Pix[i])
			if alpha == 0 {
				continue
			}
			visible++
			alphaSum += alpha
			redSum += int(out.Pix[i-3])
			greenSum += int(out.Pix[i-2])
			blueSum += int(out.Pix[i-1])
		}
		if visible == 0 {
			t.Fatalf("%s decoded as fully transparent", source)
		}
		t.Logf("%s bounds=%v visible=%d avg_rgb=%.1f,%.1f,%.1f avg_alpha=%.1f", source, out.Bounds(), visible, float64(redSum)/float64(visible), float64(greenSum)/float64(visible), float64(blueSum)/float64(visible), float64(alphaSum)/float64(visible))
	}
}
