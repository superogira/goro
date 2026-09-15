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
progdir=$(cd "$(dirname "$0")" && pwd)
exec >"$progdir/GorORG35-logfile.txt" 2>&1
cd "$progdir/GorORG35"
export GOGPU_PLATFORM=fbdev
exec ./goro -config goro.ini -data-dir .
