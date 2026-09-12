//go:build nofakecgo

package audio

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestDecodeWAVPCM8MonoToPCM16Stereo(t *testing.T) {
	got, rate, err := decodeWAVToPCM16Stereo(testWAV(1, 8, 22050, []byte{0, 128, 255}))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 22050 {
		t.Fatalf("sample rate = %d, want 22050", rate)
	}
	if len(got) != 12 {
		t.Fatalf("pcm len = %d, want 12", len(got))
	}
	want := []int16{-32768, 0, 32512}
	for frame, value := range want {
		if left, right := readPCM16(got, frame, 0), readPCM16(got, frame, 1); left != value || right != value {
			t.Fatalf("frame %d = %d,%d, want %d,%d", frame, left, right, value, value)
		}
	}
}

func TestDecodeWAVPCM8StereoToPCM16Stereo(t *testing.T) {
	got, rate, err := decodeWAVToPCM16Stereo(testWAV(2, 8, 11025, []byte{0, 255, 128, 64}))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 11025 {
		t.Fatalf("sample rate = %d, want 11025", rate)
	}
	if len(got) != 8 {
		t.Fatalf("pcm len = %d, want 8", len(got))
	}
	if left, right := readPCM16(got, 0, 0), readPCM16(got, 0, 1); left != -32768 || right != 32512 {
		t.Fatalf("frame 0 = %d,%d, want -32768,32512", left, right)
	}
	if left, right := readPCM16(got, 1, 0), readPCM16(got, 1, 1); left != 0 || right != -16384 {
		t.Fatalf("frame 1 = %d,%d, want 0,-16384", left, right)
	}
}

func TestDecodeWAVPCM8StereoRejectsOddLength(t *testing.T) {
	_, _, err := decodeWAVToPCM16Stereo(testWAV(2, 8, 44100, []byte{0, 128, 255}))
	if err == nil || !strings.Contains(err.Error(), "invalid stereo pcm length 3") {
		t.Fatalf("decode error = %v, want invalid stereo length", err)
	}
}

func testWAV(channels, bitsPerSample uint16, sampleRate uint32, data []byte) []byte {
	blockAlign := channels * bitsPerSample / 8
	byteRate := sampleRate * uint32(blockAlign)
	riffSize := uint32(4 + 8 + 16 + 8 + len(data))
	if len(data)%2 != 0 {
		riffSize++
	}
	wav := []byte("RIFF")
	wav = binary.LittleEndian.AppendUint32(wav, riffSize)
	wav = append(wav, "WAVE"...)
	wav = append(wav, "fmt "...)
	wav = binary.LittleEndian.AppendUint32(wav, 16)
	wav = binary.LittleEndian.AppendUint16(wav, 1)
	wav = binary.LittleEndian.AppendUint16(wav, channels)
	wav = binary.LittleEndian.AppendUint32(wav, sampleRate)
	wav = binary.LittleEndian.AppendUint32(wav, byteRate)
	wav = binary.LittleEndian.AppendUint16(wav, blockAlign)
	wav = binary.LittleEndian.AppendUint16(wav, bitsPerSample)
	wav = append(wav, "data"...)
	wav = binary.LittleEndian.AppendUint32(wav, uint32(len(data)))
	wav = append(wav, data...)
	if len(data)%2 != 0 {
		wav = append(wav, 0)
	}
	return wav
}

func TestSFXPCMCacheLRU(t *testing.T) {
	b := &BGM{}
	pcm6MB := make([]byte, 6<<20)
	// Fill past the 12 MiB limit: A, B fit; C evicts A (least recently used).
	b.storeSFXPCM("a", "src/a", pcm6MB)
	b.storeSFXPCM("b", "src/b", pcm6MB)
	if _, _, ok := b.cachedSFXPCM("a"); !ok {
		t.Fatal("entry a missing before eviction pressure")
	}
	// Touching "a" made "b" the least recently used entry, so storing "c"
	// past the limit evicts "b" — that is the ambient-loop protection this
	// cache exists for: recently played sounds stay hot.
	b.storeSFXPCM("c", "src/c", pcm6MB)
	if _, _, ok := b.cachedSFXPCM("b"); ok {
		t.Fatal("entry b should have been evicted (least recently used)")
	}
	if _, _, ok := b.cachedSFXPCM("a"); !ok {
		t.Fatal("entry a should survive: it was used more recently than b")
	}
	if _, _, ok := b.cachedSFXPCM("c"); !ok {
		t.Fatal("entry c should survive: it was just stored")
	}
	b.sfxCacheMu.Lock()
	bytes := b.sfxCacheBytes
	b.sfxCacheMu.Unlock()
	if bytes != 12<<20 {
		t.Fatalf("cache byte accounting: got %d want %d", bytes, 12<<20)
	}
	// Same-key store replaces without double counting.
	small := make([]byte, 1024)
	b.storeSFXPCM("c", "src/c2", small)
	b.sfxCacheMu.Lock()
	bytes = b.sfxCacheBytes
	src := b.sfxCache["c"].source
	b.sfxCacheMu.Unlock()
	if bytes != (6<<20)+1024 {
		t.Fatalf("replace should rebook bytes: got %d", bytes)
	}
	if src != "src/c2" {
		t.Fatalf("replace should update resolved source: got %s", src)
	}
}
