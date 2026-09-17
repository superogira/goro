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
exec >>"$progdir/GorORG35-logfile.txt" 2>&1
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
# forces everything written so far onto the SD card every 2 seconds.
(
  while :; do sync; sleep 2; done
) &
syncer=$!

# --- 1) display diagnostic: paints fb0 in a loop for ~9s (cycling
# colors), verifies writes survive read-back, and probes the
# mode-setting ioctls. Watch the screen during this phase and check
# fbtest.log afterwards.
if [ -x ./fbtest ] || [ -f ./fbtest ]; then
  chmod +x ./fbtest 2>/dev/null
  echo "-- running fbtest (~9s, watch the screen for colors) --"
  ./fbtest .
  echo "fbtest exited: $?"
fi
sync

# --- 2) the game itself ---
export GOGPU_PLATFORM=fbdev
export GOGPU_LOG=debug
# The app context defines neither HOME nor XDG_CONFIG_HOME; without them
# os.UserConfigDir() fails and goro logs "login ID save failed". Keep the
# save file next to the app so it survives reboots on the SD card.
export HOME="${HOME:-$progdir/GorORG35}"
export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-$progdir/GorORG35}"
# GPU experiment: render through EGL/GLES on the Mali driver (libmali).
# If the context comes up, frames go from seconds to GPU speed. Falls
# back to the software path when unset.
export GOGPU_FB_GLES=1
# diagnostic: floods every frame with magenta before the EGL swap —
# if the panel shows magenta, present reaches the screen
export GOGPU_GLES_DEBUG_CLEAR=1
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
