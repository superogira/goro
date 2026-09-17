package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/oto/v3"
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

// pipeOutput streams raw PCM into an external player's stdin.
type pipeOutput struct {
	name    string
	rate    int
	command []string // args[0] is the binary; "-" (stdin) is appended at spawn
}

func newPipeOutput(name string, rate int) *pipeOutput {
	switch name {
	case "mpv":
		return &pipeOutput{
			name: name,
			rate: rate,
			command: []string{
				"mpv", "--no-video", "--really-quiet", "--gapless-audio=inf",
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
	args := append(append([]string(nil), p.command...), "-")
	cmd := exec.Command(args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	player := &pipePlayer{done: make(chan struct{})}
	player.volume.Store(1.0)
	if err := cmd.Start(); err != nil {
		return nil
	}
	player.cmd = cmd
	player.stdin = stdin
	player.started.Store(true)
	go func() {
		_, _ = io.Copy(stdin, &gainReader{r: r, volume: &player.volume})
		// EOF (a finished SFX sample) ends the stream; the player process
		// exits on its own once stdin closes.
		_ = stdin.Close()
	}()
	go func() {
		_ = cmd.Wait()
		player.finished.Store(true)
		close(player.done)
	}()
	return player
}

// pipePlayer is an audioPlayer backed by one external process. Raw streams
// cannot pause or seek mid-flight, so Pause acts as a stop (the player is
// discarded by the callers right after) and a later resume restarts.
type pipePlayer struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	volume    atomic.Value // float64
	started   atomic.Bool
	finished  atomic.Bool
	done      chan struct{}
	closeOnce sync.Once
}
func (p *pipePlayer) Play()  { p.started.Store(true) }
func (p *pipePlayer) Pause() { _ = p.Close() }

func (p *pipePlayer) IsPlaying() bool {
	return p != nil && p.started.Load() && !p.finished.Load()
}

func (p *pipePlayer) SetVolume(volume float64) {
	p.volume.Store(clampVolume(volume))
}

func (p *pipePlayer) Close() error {
	if p == nil || p.cmd == nil {
		return nil
	}
	p.closeOnce.Do(func() {
		_ = p.stdin.Close()
		_ = p.cmd.Process.Kill()
	})
	<-p.done
	return nil
}

// gainReader multiplies the s16le samples it passes through by a live
// volume factor; it keeps volume control working for players without a
// runtime volume interface (aplay, and mpv started from a raw stream).
type gainReader struct {
	r      io.Reader
	volume *atomic.Value
}

func (g *gainReader) Read(p []byte) (int, error) {
	n, err := g.r.Read(p)
	v, _ := g.volume.Load().(float64)
	if n > 0 && v != 1.0 {
		for i := 0; i+1 < n; i += 2 {
			sample := int16(binary.LittleEndian.Uint16(p[i : i+2]))
			scaled := int16(float64(sample) * v)
			binary.LittleEndian.PutUint16(p[i:i+2], uint16(scaled))
		}
	}
	return n, err
}

// resolveAudioBackend picks the output backend: an explicit name wins;
// "auto" prefers external players on linux (mpv, then aplay) and falls back
// to the in-process oto driver everywhere else.
func resolveAudioBackend(preferred string) string {
	if preferred != "" && preferred != "auto" {
		return preferred
	}
	if runtime.GOOS == "linux" {
		for _, name := range []string{"mpv", "aplay"} {
			if _, err := exec.LookPath(name); err == nil {
				return name
			}
		}
	}
	return "oto"
}
