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

cd "$progdir/GorORG35"

# --- 1) display diagnostic: paints fb0 in a loop for ~25s (cycling
# red/green/blue/yellow/white every 5s), verifies writes survive
# read-back, and probes the mode-setting ioctls. Watch the screen
# during this phase and check fbtest.log afterwards.
if [ -x ./fbtest ] || [ -f ./fbtest ]; then
  chmod +x ./fbtest 2>/dev/null
  echo "-- running fbtest (~28s, watch the screen for colors) --"
  ./fbtest .
  echo "fbtest exited: $?"
fi

# --- 2) the game itself ---
export GOGPU_PLATFORM=fbdev
export GOGPU_LOG=debug
# gogpu now activates the fb layer itself (FBIOPAN at init) — the thing
# fbtest discovered. Full 640x480 for now: the half-size path caused a
# resize fight between the game and UI surfaces (login drawn top-left,
# colors off), so it stays off until that's fixed.
./goro -config goro.ini -data-dir .
echo "goro exited: $?"
