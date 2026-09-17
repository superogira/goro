//go:build nofakecgo

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/kivutar/goro/res"
)

func (b *BGM) SFXVolume() float64 {
	if b == nil {
		return 0
	}
	return b.sfxVolume
}

func (b *BGM) SetSFXVolume(volume float64) {
	if b == nil {
		return
	}
	volume = clampVolume(volume)
	b.sfxVolume = volume
	for _, player := range b.sfxPlayers {
		if player != nil {
			player.SetVolume(volume)
		}
	}
}

func (b *BGM) PlaySFX(path string) (string, error) {
	return b.PlaySFXVolume(path, 1)
}

// sfxCacheLimit caps the decoded-PCM cache. Ambient wavs run a few hundred
// kilobytes each; 12 MiB keeps a map's worth of loops hot without growing
// unbounded across every sound the session ever played.
const sfxCacheLimit = 12 << 20

type sfxCacheEntry struct {
	pcm    []byte
	source string
	used   int64
}

// cachedSFXPCM returns the context-rate PCM for a previously decoded source
// path, refreshing its LRU stamp.
func (b *BGM) cachedSFXPCM(path string) ([]byte, string, bool) {
	b.sfxCacheMu.Lock()
	defer b.sfxCacheMu.Unlock()
	entry, ok := b.sfxCache[path]
	if !ok {
		return nil, "", false
	}
	b.sfxCacheTick++
	entry.used = b.sfxCacheTick
	return entry.pcm, entry.source, true
}

// storeSFXPCM caches decoded PCM, evicting least-recently-used entries when
// the cache would exceed sfxCacheLimit.
func (b *BGM) storeSFXPCM(path, source string, pcm []byte) {
	b.sfxCacheMu.Lock()
	defer b.sfxCacheMu.Unlock()
	if b.sfxCache == nil {
		b.sfxCache = make(map[string]*sfxCacheEntry)
	}
	if old, exists := b.sfxCache[path]; exists {
		b.sfxCacheBytes -= len(old.pcm)
	}
	b.sfxCacheBytes += len(pcm)
	b.sfxCacheTick++
	b.sfxCache[path] = &sfxCacheEntry{pcm: pcm, source: source, used: b.sfxCacheTick}
	for b.sfxCacheBytes > sfxCacheLimit {
		var oldestKey string
		var oldest int64
		first := true
		for key, entry := range b.sfxCache {
			if first || entry.used < oldest {
				oldestKey, oldest, first = key, entry.used, false
			}
		}
		if first {
			break
		}
		b.sfxCacheBytes -= len(b.sfxCache[oldestKey].pcm)
		delete(b.sfxCache, oldestKey)
	}
}

func (b *BGM) PlaySFXVolume(path string, volume float64) (string, error) {
	if b == nil || b.disabled || b.sfxVolume <= 0 || volume <= 0 {
		return "", nil
	}
	path = normalizeSFXPath(path)
	if path == "" {
		return "", nil
	}
	output := b.ensureOutput(defaultSampleRate)
	if output == nil {
		return "", fmt.Errorf("audio context unavailable")
	}
	// Map ambient sounds retrigger every RSW cycle; re-reading and re-decoding
	// the wav on the game goroutine each time is a visible stutter on slower
	// devices. The decoded PCM is cached per source, so steady-state plays
	// cost a map lookup and a player creation.
	if pcm, source, ok := b.cachedSFXPCM(path); ok {
		b.startSFXPlayer(output, pcm, volume)
		return source, nil
	}
	data, source, err := readSFXFile(b.resources, path)
	if err != nil {
		return "", err
	}
	pcm, sourceRate, err := decodeWAVToPCM16Stereo(data)
	if err != nil {
		return source, fmt.Errorf("decode sfx %s: %w", source, err)
	}
	if sourceRate != b.sampleRate {
		pcm, err = resamplePCM16Stereo(pcm, sourceRate, b.sampleRate)
		if err != nil {
			return source, fmt.Errorf("resample sfx %s: %w", source, err)
		}
	}
	b.storeSFXPCM(path, source, pcm)
	b.startSFXPlayer(output, pcm, volume)
	return source, nil
}

func (b *BGM) startSFXPlayer(output audioOutput, pcm []byte, volume float64) {
	player := output.NewPlayer(bytes.NewReader(pcm))
	if player == nil {
		return
	}
	player.SetVolume(b.sfxVolume * clampVolume(volume))
	b.trimSFXPlayers()
	b.sfxPlayers = append(b.sfxPlayers, player)
	player.Play()
}

func (b *BGM) trimSFXPlayers() {
	if len(b.sfxPlayers) == 0 {
		return
	}
	active := b.sfxPlayers[:0]
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		if player.IsPlaying() {
			active = append(active, player)
			continue
		}
	}
	const maxSFXPlayers = 32
	for len(active) > maxSFXPlayers {
		player := active[0]
		active = active[1:]
		if player != nil {
			player.Pause()
		}
	}
	b.sfxPlayers = active
}

func (b *BGM) stopSFX() {
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		_ = player.Close()
	}
	b.sfxPlayers = nil
}

func decodeWAVToPCM16Stereo(data []byte) ([]byte, int, error) {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	pos := 12
	var channels, bitsPerSample uint16
	var sampleRate uint32
	var pcm []byte
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(data) {
			return nil, 0, fmt.Errorf("invalid wav chunk %q size %d", id, size)
		}
		chunk := data[pos : pos+size]
		switch id {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, 0, fmt.Errorf("short fmt chunk")
			}
			format := binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(chunk[14:16])
			if format != 1 {
				return nil, 0, fmt.Errorf("unsupported wav format %d", format)
			}
		case "data":
			pcm = append([]byte(nil), chunk...)
		}
		pos += size
		if pos%2 != 0 {
			pos++
		}
	}
	if sampleRate == 0 || len(pcm) == 0 {
		return nil, 0, fmt.Errorf("missing wav fmt or data")
	}
	switch bitsPerSample {
	case 8:
		var err error
		pcm, err = pcm8ToPCM16Stereo(pcm, channels)
		if err != nil {
			return nil, 0, err
		}
	case 16:
		var err error
		pcm, err = pcm16ToStereo(pcm, channels)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fmt.Errorf("unsupported wav bits per sample %d", bitsPerSample)
	}
	return pcm, int(sampleRate), nil
}

func pcm8ToPCM16Stereo(src []byte, channels uint16) ([]byte, error) {
	switch channels {
	case 1:
		dst := make([]byte, len(src)*4)
		for frame, sample := range src {
			value := unsignedPCM8ToPCM16(sample)
			writePCM16(dst, frame, 0, value)
			writePCM16(dst, frame, 1, value)
		}
		return dst, nil
	case 2:
		if len(src)%2 != 0 {
			return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
		}
		dst := make([]byte, len(src)*2)
		for frame := 0; frame < len(src)/2; frame++ {
			writePCM16(dst, frame, 0, unsignedPCM8ToPCM16(src[frame*2]))
			writePCM16(dst, frame, 1, unsignedPCM8ToPCM16(src[frame*2+1]))
		}
		return dst, nil
	default:
		return nil, fmt.Errorf("unsupported wav channels %d", channels)
	}
}

func pcm16ToStereo(src []byte, channels uint16) ([]byte, error) {
	switch channels {
	case 1:
		return monoPCM16ToStereo(src)
	case 2:
		if len(src)%4 != 0 {
			return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
		}
		return src, nil
	default:
		return nil, fmt.Errorf("unsupported wav channels %d", channels)
	}
}

func monoPCM16ToStereo(src []byte) ([]byte, error) {
	if len(src)%2 != 0 {
		return nil, fmt.Errorf("invalid mono pcm length %d", len(src))
	}
	dst := make([]byte, len(src)*2)
	for frame := 0; frame < len(src)/2; frame++ {
		sample := int16(uint16(src[frame*2]) | uint16(src[frame*2+1])<<8)
		writePCM16(dst, frame, 0, sample)
		writePCM16(dst, frame, 1, sample)
	}
	return dst, nil
}

func unsignedPCM8ToPCM16(sample byte) int16 {
	return int16((int(sample) - 128) << 8)
}

func readSFXFile(manager *res.Manager, path string) ([]byte, string, error) {
	if manager == nil {
		return nil, "", fmt.Errorf("resource manager is nil")
	}
	for _, candidate := range sfxPathCandidates(path) {
		data, err := manager.ReadFileExact(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("sfx not found: %s", path)
}
