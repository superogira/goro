#!/bin/bash
# goro (Ragnarik Online) launcher for Anbernic RG35XX Plus on StockOS mod.
# Install: copy GorORG35/ (folder) and GorORG35.sh into Roms/APPS, then reboot.
#
# App folder layout (the app folder itself is the data root):
#   GorORG35/goro        binary
#   GorORG35/goro.ini    config
#   GorORG35/fbtest      tiny display diagnostic (run first)
#   GorORG35/data.grf    packed game data (entries data\...)
#   GorORG35/BGM/        loose BGM (classic RO layout, optional)
#   GorORG35/clientinfo.xml  loose override for the server address (optional)
progdir=$(cd "$(dirname "$0")" && pwd)
# Rotate the previous run's logs instead of appending forever: the debug
# levels used during bring-up grew these to megabytes per play session and
# ate SD card space. Each launch keeps exactly one .old copy.
for f in "$progdir/GorORG35-logfile.txt" "$progdir/GorORG35/goro.log"; do
  [ -f "$f" ] && mv -f "$f" "$f.old"
done
exec >"$progdir/GorORG35-logfile.txt" 2>&1
echo "=== goro launch $(date) ==="
uname -a
free -m 2>/dev/null | head -2

# What is the system launcher doing while an app runs? Its cmdline and
# script (if readable) reveal whether it keeps drawing a loading screen.
for pid in $(pidof launcher.sh 2>/dev/null); do
  echo "-- launcher pid $pid:"
  echo "  cmdline: $(tr '\0' ' ' < /proc/$pid/cmdline 2>/dev/null)"
  echo "  exe: $(readlink /proc/$pid/exe 2>/dev/null)"
  script=$(tr '\0' ' ' < /proc/$pid/cmdline 2>/dev/null | awk '{print $1}')
  if [ -f "$script" ]; then
    echo "  script (first 3000 bytes):"
    head -c 3000 "$script" 2>/dev/null | sed 's/^/    /'
  fi
done

# Display nodes recap
for f in /sys/class/graphics/fb0/virtual_size /sys/class/graphics/fb0/bits_per_pixel; do
  echo "  $f = $(cat "$f" 2>/dev/null)"
done
ls -la /dev/disp /dev/fb0 2>/dev/null

# GPU / SDL runtime: the smooth apps here (clock, ROCreader) render via
# SDL2 with hardware acceleration — record what the device actually has.
echo "-- /usr/lib GPU/SDL libs:"
ls /usr/lib/ 2>/dev/null | grep -iE 'SDL|EGL|GLES|mali|gbm|\.so' | grep -iE 'SDL|EGL|GLES|mali|gbm' | head -20
ls /lib/ 2>/dev/null | grep -iE 'EGL|GLES|mali' | head -10
echo "  board: $(head -1 /mnt/vendor/oem/board.ini 2>/dev/null)"

# Input arbitration: who else holds the evdev nodes? An exclusive
# EVIOCGRAB by a system daemon (gptokeyb-style) silently swallows every
# button event before goro can read it.
echo "-- processes holding /dev/input/event*:"
for fdpath in /proc/[0-9]*/fd/*; do
  target=$(readlink "$fdpath" 2>/dev/null)
  case "$target" in
    /dev/input/event*)
      pid=$(echo "$fdpath" | cut -d/ -f3)
      echo "  $target <- pid $pid ($(cat /proc/$pid/cmdline 2>/dev/null | tr '\0' ' '))"
      ;;
  esac
done
echo "-- running processes:"
ps -ef 2>/dev/null || ls /proc/[0-9]*/cmdline 2>/dev/null | while read c; do
  pid=$(echo "$c" | cut -d/ -f3)
  echo "  $pid: $(tr '\0' ' ' < "$c" 2>/dev/null)"
done

cd "$progdir/GorORG35"

# Log survival: a hard GPU/kernel crash takes the page cache with it,
# leaving a 0-byte logfile after the power cycle. A background syncer
# forces everything written so far onto the SD card every 10 seconds —
# often enough to survive a crash, rarely enough not to stall the game
# with constant SD writes.
(
  while :; do sync; sleep 10; done
) &
syncer=$!

# --- 1) display diagnostic (opt-in): create a marker file "fbtest.enabled"
# next to this script to re-arm it; the panel and present path are proven
# now, so it stays out of the normal boot.
if [ -f ./fbtest.enabled ] && { [ -x ./fbtest ] || [ -f ./fbtest ]; }; then
  chmod +x ./fbtest 2>/dev/null
  echo "-- running fbtest (~9s, watch the screen for colors) --"
  ./fbtest .
  echo "fbtest exited: $?"
fi
sync

# Audio diagnostics: the game's audio context opens silently, so record
# what the kernel actually exposes before blaming the client.
echo "-- sound cards:"
cat /proc/asound/cards 2>/dev/null || echo "  /proc/asound/cards unavailable"
echo "-- /dev/snd:"
ls /dev/snd 2>/dev/null || echo "  /dev/snd missing"
echo "-- mixer state (first 60 lines):"
amixer 2>/dev/null | head -60 || echo "  amixer unavailable"

# DNS check for the game server hostname (from clientinfo.xml): a missing
# DDNS record shows up as "no such host" in the game with no other hint.
host=$(sed -n 's/.*<address>\([^<]*\)<\/address>.*/\1/p' data/clientinfo.xml 2>/dev/null | head -1)
if [ -n "$host" ]; then
  echo "-- resolving clientinfo host '$host':"
  getent hosts "$host" 2>/dev/null || echo "  FAILED: hostname does not resolve (DDNS expired/IP changed?)"
fi

# mpv self-test: a 2s generated tone through mpv+ALSA right at launch —
# an audible beep proves the whole chain works before the game even starts,
# and the rc/timing/stderr in the log show what happens when it doesn't.
# The second test replays the same tone as a RAW stream through the exact
# option set the game uses, so an option mpv rejects dies here, visibly.
if command -v mpv >/dev/null 2>&1; then
  echo "-- mpv version: $(mpv --version 2>&1 | head -1)"
  echo "-- aplay: $(command -v aplay || echo missing)"
  if python3 - >/dev/null 2>&1 <<'PYEOF'
import math, struct, wave
w = wave.open('/tmp/goro-tone.wav', 'w')
w.setnchannels(2); w.setsampwidth(2); w.setframerate(44100)
w.writeframes(b''.join(struct.pack('<hh',
    int(9000*math.sin(i*0.09)), int(9000*math.sin(i*0.09)))
    for i in range(44100*2)))
w.close()
PYEOF
  then
    t0=$(date +%s)
    cat /tmp/goro-tone.wav | timeout 8 mpv --no-video --ao=alsa - >/dev/null 2>/tmp/goro-mpv-test.log
    rc=$?
    t1=$(date +%s)
    echo "-- mpv wav test rc=$rc elapsed=$((t1-t0))s (a beep should have played) stderr:"
    head -c 400 /tmp/goro-mpv-test.log 2>/dev/null | tr '\r' '\n' | tail -3 | sed 's/^/    /'
    # Raw-stream test with the game's exact option set (second beep).
    tail -c +45 /tmp/goro-tone.wav | timeout 8 mpv --no-video --ao=alsa \
      --demuxer=rawaudio --demuxer-rawaudio-format=s16le \
      --demuxer-rawaudio-rate=44100 --demuxer-rawaudio-channels=stereo - \
      >/dev/null 2>/tmp/goro-mpv-raw.log
    rc=$?
    echo "-- mpv raw test rc=$rc (a second beep should have played) stderr:"
    head -c 600 /tmp/goro-mpv-raw.log 2>/dev/null | tr '\r' '\n' | grep -v '^$' | tail -5 | sed 's/^/    /'
  else
    echo "-- python3 unavailable, tone test skipped"
  fi
else
  echo "-- mpv not found in PATH"
fi

# --- 2) the game itself ---
export GOGPU_PLATFORM=fbdev
export GOGPU_LOG=info
# ALSA: prefer the speaker codec (card 0) over the HDMI card (card 2)
# whenever the audio library resolves the "default" device.
export ALSA_CARD="${ALSA_CARD:-0}"
# Watch whether goro actually opens a PCM stream while playing (bounded,
# ~2 minutes of samples, a few hundred bytes).
(
  for i in 1 2 3 4 5 6; do
    sleep 20
    echo "-- pcm stream state (t+$((i*20))s):"
    for c in /proc/asound/card0/pcm0p/sub0/hw_params /proc/asound/card2/pcm0p/sub0/hw_params; do
      echo "  $c:"
      sed 's/^/    /' "$c" 2>/dev/null || echo "    (closed)"
    done
  done
) &
# The app context defines neither HOME nor XDG_CONFIG_HOME; without them
# os.UserConfigDir() fails and goro logs "login ID save failed". Keep the
# user store on the SD card, in a dedicated dir: pointing XDG at the app
# dir itself would resolve the store to <app>/goro/goro.ini, colliding
# with the goro binary (ENOTDIR).
export HOME="${HOME:-$progdir/GorORG35}"
export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$progdir/GorORG35/.config}"
mkdir -p "$XDG_CONFIG_HOME/goro" 2>/dev/null
# GPU experiment: render through EGL/GLES on the Mali driver (libmali).
# If the context comes up, frames go from seconds to GPU speed. Falls
# back to the software path when unset.
export GOGPU_FB_GLES=1
./goro -config goro.ini -data-dir . -graphics-api gles
echo "goro exited: $?"
sync

# If the GPU driver or kernel oopsed, dmesg holds the fingerprints.
echo "-- dmesg tail after goro exit:"
dmesg 2>/dev/null | tail -40
echo "-- still-running goro processes:"
ps -ef 2>/dev/null | grep -v grep | grep goro || echo "  none"

kill $syncer 2>/dev/null
sync
