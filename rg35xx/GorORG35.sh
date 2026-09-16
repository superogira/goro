#!/bin/bash
# goro (Ragnarik Online) launcher for Anbernic RG35XX Plus on StockOS mod.
# Install: copy GorORG35/ (folder) and GorORG35.sh into Roms/APPS, then reboot.
#
# App folder layout (the app folder itself is the data root):
#   GorORG35/goro        binary
#   GorORG35/goro.ini    config
#   GorORG35/data.grf    packed game data (entries data\...)
#   GorORG35/BGM/        loose BGM (classic RO layout, optional)
#   GorORG35/clientinfo.xml  loose override for the server address (optional)
# or an extracted tree in GorORG35/data/.
#
# Everything the run prints (system context + goro's own log) lands in
# GorORG35-logfile.txt next to this script, and goro also writes
# GorORG35/goro.log through its own file sink.
progdir=$(cd "$(dirname "$0")" && pwd)
exec >>"$progdir/GorORG35-logfile.txt" 2>&1
echo "=== goro launch $(date) ==="
uname -a
free -m 2>/dev/null || true
ls -la "$progdir/GorORG35" 2>/dev/null | head -20

# Framebuffer visibility probe: goro paints each screen buffer of fb0 a
# different solid color for 6 seconds (red = buffer 0, green = 1,
# blue = 2, yellow = 3). The color you see identifies the buffer the
# panel actually scans; NO color at all means fb0 is not the visible
# layer and no fbdev app can draw there.
for f in /sys/class/graphics/fb0/virtual_size /sys/class/graphics/fb0/bits_per_pixel; do
  echo "  $f = $(cat "$f" 2>/dev/null)"
done

cd "$progdir/GorORG35"
export GOGPU_PLATFORM=fbdev
export GOGPU_FB_PROBE=color
# gogpu logs the real framebuffer geometry (device, size, virtual size,
# offsets, bpp, stride)
export GOGPU_LOG=debug
# render at half the panel size; the fb blit upscales it (roughly 4x
# cheaper on the software rasterizer)
export GOGPU_FB_SCALE=2
# Not exec: this shell survives goro and records how it ended —
# exit 137 = SIGKILL (the kernel OOM killer), 139 = segfault.
./goro -config goro.ini -data-dir .
echo "goro exited: $?"
