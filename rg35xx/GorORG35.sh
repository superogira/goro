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
cd "$progdir/GorORG35"
export GOGPU_PLATFORM=fbdev
# Not exec: this shell survives goro and records how it ended —
# exit 137 = SIGKILL (the kernel OOM killer), 139 = segfault.
./goro -config goro.ini -data-dir .
echo "goro exited: $?"
