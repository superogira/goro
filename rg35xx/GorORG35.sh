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

# Audio bring-up notes (solved, kept for reference): the in-process oto
# driver never opens a PCM stream on this firmware, so the game pipes a
# software-mixed s16le stream into one mpv process (--ao=alsa, see
# audio/pipe_output.go). The launch-time beep self-tests and the PCM
# watcher were removed once confirmed working; the log still records the
# audio backend and master stream start from the game itself.

# --- 2) the game itself ---
export GOGPU_PLATFORM=fbdev
export GOGPU_LOG=info
# ALSA: prefer the speaker codec (card 0) over the HDMI card (card 2)
# whenever the audio library resolves the "default" device.
export ALSA_CARD="${ALSA_CARD:-0}"
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
