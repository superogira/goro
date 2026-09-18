package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/kivutar/goro/glog"
)

// audioPlayer abstracts the oto.Player surface used by BGM and SFX so the
// output can also be an external process fed raw PCM over stdin. On the
// rg35xx the in-process driver reports a working context yet never opens a
// kernel PCM stream (silent), while mpv on the same device plays fine — the
// same approach net_radio.sh uses there.
type audioPlayer interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(volume float64)
	Close() error
}

// audioOutput creates players that consume a raw s16le stereo stream at the
// BGM context sample rate.
type audioOutput interface {
	NewPlayer(r io.Reader) audioPlayer
	Name() string
}

// Compile-time proof that oto players satisfy the player contract as-is.
var _ audioPlayer = (*oto.Player)(nil)

type otoOutput struct {
	context *oto.Context
}

func (o otoOutput) Name() string                      { return "oto" }
func (o otoOutput) NewPlayer(r io.Reader) audioPlayer { return o.context.NewPlayer(r) }

// pipeOutput owns ONE external player process for its whole lifetime and
// mixes every BGM/SFS voice in software before the pipe. The speaker device
// on the rg35xx opens exclusively (no dmix), so simultaneous player
// processes can never share it — a single stream must carry the whole mix.
type pipeOutput struct {
	name    string
	rate    int
	command []string

	startOnce sync.Once
	mu        sync.Mutex
	voices    map[*mixerVoice]struct{}
}

func newPipeOutput(name string, rate int) *pipeOutput {
	switch name {
	case "mpv":
		return &pipeOutput{
			name: name,
			rate: rate,
			// Keep this list in sync with the launcher's raw-stream
			// self-test. --gapless-audio=inf killed every spawn with
			// exit status 1: mpv accepts only no|yes|weak there.
			command: []string{
				"mpv", "--no-video", "--really-quiet", "--ao=alsa",
				"--demuxer=rawaudio", "--demuxer-rawaudio-format=s16le",
				fmt.Sprintf("--demuxer-rawaudio-rate=%d", rate),
				"--demuxer-rawaudio-channels=stereo",
			},
		}
	case "aplay":
		return &pipeOutput{
			name: name,
			rate: rate,
			command: []string{
				"aplay", "-q", "-f", "S16_LE", "-r", fmt.Sprintf("%d", rate), "-c", "2",
			},
		}
	}
	return nil
}

func (p *pipeOutput) Name() string { return p.name }

func (p *pipeOutput) NewPlayer(r io.Reader) audioPlayer {
	p.startOnce.Do(func() { p.startMaster() })
	v := &mixerVoice{
		owner:  p,
		reader: r,
		// One mix chunk (framesPerChunk*4 bytes), so every voice read
		// aligns with the pump chunk size.
		scratch: make([]byte, 1024*4),
	}
	v.active.Store(true)
	v.volume.Store(1.0)
	p.mu.Lock()
	p.voices[v] = struct{}{}
	p.mu.Unlock()
	return v
}

// startMaster spawns the single player process and keeps it fed for the
// rest of the session, holding the device open with silence when nothing
// plays (which also makes BGM track changes gapless).
func (p *pipeOutput) startMaster() {
	p.voices = make(map[*mixerVoice]struct{})
	args := append(append([]string(nil), p.command...), "-")
	cmd := exec.Command(args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		glog.Warnf("audio master pipe failed name=%s: %v", p.name, err)
		return
	}
	if err := cmd.Start(); err != nil {
		glog.Warnf("audio master start failed name=%s: %v", p.name, err)
		return
	}
	glog.Infof("audio master stream started name=%s pid=%d rate=%d", p.name, cmd.Process.Pid, p.rate)
	go func() {
		waitErr := cmd.Wait()
		glog.Infof("audio master stream exited name=%s pid=%d err=%v", p.name, cmd.Process.Pid, waitErr)
	}()
	go p.pump(stdin)
}

// pump writes realtime-paced mixed chunks into the player's stdin. Pacing
// matters: an unpaced writer outruns playback and mpv balloons its cache
// (150MB observed on the 1GB device before this fix). 1024 frames (~23ms)
// keeps SFX latency low.
func (p *pipeOutput) pump(stdin io.WriteCloser) {
	const framesPerChunk = 1024
	chunk := make([]byte, framesPerChunk*4)
	chunkDur := time.Duration(framesPerChunk) * time.Second / time.Duration(p.rate)
	ticker := time.NewTicker(chunkDur)
	defer ticker.Stop()
	for range ticker.C {
		p.mix(chunk)
		if _, err := stdin.Write(chunk); err != nil {
			glog.Warnf("audio master write failed name=%s: %v", p.name, err)
			_ = stdin.Close()
			return
		}
	}
}

// mix renders one chunk: the sum of every active voice at its volume,
// clipped to int16. Voices that run dry are finished and removed.
func (p *pipeOutput) mix(chunk []byte) {
	for i := range chunk {
		chunk[i] = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for v := range p.voices {
		if !v.active.Load() || v.finished.Load() {
			continue
		}
		n, err := io.ReadFull(v.reader, v.scratch[:len(chunk)])
		vol, _ := v.volume.Load().(float64)
		for i := 0; i+3 < n; i += 4 {
			left := int32(int16(binary.LittleEndian.Uint16(v.scratch[i : i+2])))
			right := int32(int16(binary.LittleEndian.Uint16(v.scratch[i+2 : i+4])))
			baseL := int32(int16(binary.LittleEndian.Uint16(chunk[i : i+2])))
			baseR := int32(int16(binary.LittleEndian.Uint16(chunk[i+2 : i+4])))
			writeSample16(chunk[i:i+2], baseL+int32(float64(left)*vol))
			writeSample16(chunk[i+2:i+4], baseR+int32(float64(right)*vol))
		}
		if err != nil || n < len(chunk) {
			// One-shot sample drained: drop it from the mix.
			v.finished.Store(true)
			delete(p.voices, v)
		}
	}
}

func writeSample16(dst []byte, v int32) {
	if v > 32767 {
		v = 32767
	} else if v < -32768 {
		v = -32768
	}
	u := uint16(int16(v))
	dst[0], dst[1] = byte(u), byte(u>>8)
}

// mixerVoice is one BGM or SFX source inside the pipeOutput mix.
type mixerVoice struct {
	owner    *pipeOutput
	reader   io.Reader
	scratch  []byte
	active   atomic.Bool
	finished atomic.Bool
	volume   atomic.Value // float64
}

func (v *mixerVoice) Play()  { v.active.Store(true) }
func (v *mixerVoice) Pause() { v.active.Store(false) }

func (v *mixerVoice) IsPlaying() bool {
	return v != nil && v.active.Load() && !v.finished.Load()
}

func (v *mixerVoice) SetVolume(volume float64) {
	v.volume.Store(clampVolume(volume))
}

func (v *mixerVoice) Close() error {
	if v == nil {
		return nil
	}
	v.active.Store(false)
	v.finished.Store(true)
	v.owner.mu.Lock()
	delete(v.owner.voices, v)
	v.owner.mu.Unlock()
	return nil
}

// resolveAudioBackend picks the output backend: an explicit name wins;
// "auto" prefers aplay on linux (its ALSA-only path has the least latency —
// mpv's stream buffering delays SFX behind the action on the rg35xx), then
// mpv, then falls back to the in-process oto driver elsewhere.
func resolveAudioBackend(preferred string) string {
	if preferred != "" && preferred != "auto" {
		return preferred
	}
	if runtime.GOOS == "linux" {
		for _, name := range []string{"aplay", "mpv"} {
			if _, err := exec.LookPath(name); err == nil {
				return name
			}
		}
	}
	return "oto"
}
