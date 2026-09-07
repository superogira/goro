//go:build nofakecgo

package audio

import (
	"os"
	"os/exec"
	"testing"

	"github.com/kivutar/goro/res"
)

func TestParseMP3NameTable(t *testing.T) {
	table := ParseMP3NameTable([]byte("prontera.rsw#bgm\\08.mp3#\r\nDATA\\geffen.rsw#09.mp3#\ninvalid\n"))
	if got := table["prontera.rsw"]; got != "bgm\\08.mp3" {
		t.Fatalf("prontera bgm = %q", got)
	}
	if got := table["geffen.rsw"]; got != "bgm\\09.mp3" {
		t.Fatalf("geffen bgm = %q", got)
	}
}

func TestCanonicalMapName(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"prontera", "prontera.rsw"},
		{"prontera.gat", "prontera.rsw"},
		{"data\\prontera.rsw", "prontera.rsw"},
		{"data/prontera.rsw", "prontera.rsw"},
	} {
		if got := canonicalMapName(tc.in); got != tc.want {
			t.Fatalf("canonicalMapName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAudioPathCandidatesPreferBGMDirectory(t *testing.T) {
	got := audioPathCandidates("bgm\\01.mp3")
	want := "BGM\\01.mp3"
	for _, candidate := range got {
		if candidate == want {
			return
		}
	}
	t.Fatalf("candidates %v did not include %q", got, want)
}

func TestSFXPathCandidatesUseWavDataDirectories(t *testing.T) {
	got := sfxPathCandidates("_enemy_hit_normal1.wav")
	for _, want := range []string{"_enemy_hit_normal1.wav", "wav\\_enemy_hit_normal1.wav", "data\\wav\\_enemy_hit_normal1.wav"} {
		if !containsString(got, want) {
			t.Fatalf("candidates %v did not include %q", got, want)
		}
	}
}

func TestSFXPathCandidatesLeadWithWavDataDirectory(t *testing.T) {
	got := sfxPathCandidates("lunatic_die.wav")
	if len(got) < 2 || got[0] != `data\wav\lunatic_die.wav` || got[1] != "data/wav/lunatic_die.wav" {
		t.Fatalf("candidates %v must lead with data\\wav\\ spellings", got)
	}
	got = sfxPathCandidates(`wav\se_prtbird_02.wav`)
	if len(got) < 2 || got[0] != `data\wav\se_prtbird_02.wav` || got[1] != "data/wav/se_prtbird_02.wav" {
		t.Fatalf("candidates %v must lead with data\\wav\\ spellings", got)
	}
}

func TestSFXPathCandidatesAppendWavExtension(t *testing.T) {
	got := sfxPathCandidates("effect\\attack")
	for _, want := range []string{"effect\\attack.wav", "wav\\effect\\attack.wav", "data\\wav\\effect\\attack.wav"} {
		if !containsString(got, want) {
			t.Fatalf("candidates %v did not include %q", got, want)
		}
	}
}

func TestOutputSampleRateForLowRateMP3UsesDefaultRate(t *testing.T) {
	if got := outputSampleRateForSource(22050); got != defaultSampleRate {
		t.Fatalf("output sample rate for 22050 = %d, want %d", got, defaultSampleRate)
	}
}

func TestResamplePCM16StereoUpsamplePreservesConstantSignal(t *testing.T) {
	var src []byte
	for range 64 {
		src = appendPCMFrame(src, 10000, -12000)
	}

	got, err := resamplePCM16Stereo(src, 22050, 44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(src)*2 {
		t.Fatalf("resampled len = %d, want %d", len(got), len(src)*2)
	}
	for frame := 16; frame < len(got)/4-16; frame++ {
		left, right := readPCM16(got, frame, 0), readPCM16(got, frame, 1)
		if absInt16Diff(left, 10000) > 1 || absInt16Diff(right, -12000) > 1 {
			t.Fatalf("frame %d = %d,%d want near 10000,-12000", frame, left, right)
		}
	}
}

func TestResamplePCM16StereoUpsampleDoesNotRingIntoClipping(t *testing.T) {
	var src []byte
	for frame := 0; frame < 64; frame++ {
		if frame%2 == 0 {
			src = appendPCMFrame(src, 30000, -30000)
		} else {
			src = appendPCMFrame(src, -30000, 30000)
		}
	}

	got, err := resamplePCM16Stereo(src, 22050, 44100)
	if err != nil {
		t.Fatal(err)
	}
	for frame := 0; frame < len(got)/4; frame++ {
		for channel := 0; channel < 2; channel++ {
			value := readPCM16(got, frame, channel)
			if value > 30000 || value < -30000 {
				t.Fatalf("frame %d channel %d = %d, want no overshoot past source range", frame, channel, value)
			}
		}
	}
}

func TestResamplePCM16StereoUpsampleInterpolatesHalfFrames(t *testing.T) {
	src := appendPCMFrame(nil, 10000, -10000)
	src = appendPCMFrame(src, 20000, -20000)

	got, err := resamplePCM16Stereo(src, 22050, 44100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(src)*2 {
		t.Fatalf("resampled len = %d, want %d", len(got), len(src)*2)
	}
	if left := readPCM16(got, 1, 0); left != 15000 {
		t.Fatalf("interpolated left = %d, want 15000", left)
	}
	if right := readPCM16(got, 1, 1); right != -15000 {
		t.Fatalf("interpolated right = %d, want -15000", right)
	}
}

func TestMonoPCM16ToStereo(t *testing.T) {
	src := []byte{0x34, 0x12, 0x00, 0x80}
	got, err := monoPCM16ToStereo(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("stereo len = %d, want 8", len(got))
	}
	if readPCM16(got, 0, 0) != 0x1234 || readPCM16(got, 0, 1) != 0x1234 {
		t.Fatalf("frame 0 = %d,%d", readPCM16(got, 0, 0), readPCM16(got, 0, 1))
	}
	if readPCM16(got, 1, 0) != -32768 || readPCM16(got, 1, 1) != -32768 {
		t.Fatalf("frame 1 = %d,%d", readPCM16(got, 1, 0), readPCM16(got, 1, 1))
	}
}

func TestDebugCompareRealBGMWithFFmpeg(t *testing.T) {
	if os.Getenv("GORO_DEBUG_BGM_FFMPEG") != "1" {
		t.Skip("set GORO_DEBUG_BGM_FFMPEG=1")
	}
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	data, source, err := readAudioFile(manager, "bgm\\01.mp3")
	if err != nil {
		t.Fatal(err)
	}
	goPCM, rate, _, err := decodeNativePCM(data)
	if err != nil {
		t.Fatal(err)
	}
	in := t.TempDir() + "/bgm.mp3"
	if err := os.WriteFile(in, data, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("ffmpeg", "-v", "error", "-i", in, "-f", "s16le", "-acodec", "pcm_s16le", "-ac", "2", "-ar", "22050", "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	logPCMComparison(t, source, goPCM, rate, out)
	for _, offsetSamples := range []int{-2304, -1152, -576, 0, 576, 1152, 2304} {
		mean := meanAbsPCM16DiffWithOffset(goPCM, out, offsetSamples)
		t.Logf("offset_samples=%d mean_diff=%.2f", offsetSamples, mean)
	}
}

func logPCMComparison(t *testing.T, label string, pcm []byte, rate int, reference []byte) {
	t.Helper()
	limit := min(len(pcm), len(reference))
	mismatch := 0
	maxDiff := 0
	sumDiff := int64(0)
	for i := 0; i+1 < limit; i += 2 {
		a := int(int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8))
		b := int(int16(uint16(reference[i]) | uint16(reference[i+1])<<8))
		diff := a - b
		if diff < 0 {
			diff = -diff
		}
		if diff != 0 {
			mismatch++
		}
		if diff > maxDiff {
			maxDiff = diff
		}
		sumDiff += int64(diff)
	}
	samples := limit / 2
	t.Logf("%s rate=%d bytes=%d ffmpeg_bytes=%d samples=%d mismatch=%d max_diff=%d mean_diff=%.2f", label, rate, len(pcm), len(reference), samples, mismatch, maxDiff, float64(sumDiff)/float64(samples))
}

func appendPCMFrame(pcm []byte, left, right int16) []byte {
	frame := make([]byte, 4)
	writePCM16(frame, 0, 0, left)
	writePCM16(frame, 0, 1, right)
	return append(pcm, frame...)
}

func absInt16Diff(a, b int16) int {
	diff := int(a) - int(b)
	if diff < 0 {
		return -diff
	}
	return diff
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func meanAbsPCM16DiffWithOffset(a, b []byte, offsetSamples int) float64 {
	aStart, bStart := 0, 0
	if offsetSamples > 0 {
		aStart = offsetSamples * 2
	} else if offsetSamples < 0 {
		bStart = -offsetSamples * 2
	}
	limit := min(len(a)-aStart, len(b)-bStart)
	if limit <= 1 {
		return 0
	}
	limit -= limit % 2
	sum := int64(0)
	samples := 0
	for i := 0; i+1 < limit; i += 2 {
		ai := aStart + i
		bi := bStart + i
		av := int(int16(uint16(a[ai]) | uint16(a[ai+1])<<8))
		bv := int(int16(uint16(b[bi]) | uint16(b[bi+1])<<8))
		diff := av - bv
		if diff < 0 {
			diff = -diff
		}
		sum += int64(diff)
		samples++
	}
	return float64(sum) / float64(samples)
}
