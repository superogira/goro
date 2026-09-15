#!/bin/bash
# goro (Ragnarok Online) launcher for Anbernic RG35XX Plus on StockOS mod.
# Install: copy GorORG35/ (folder) and GorORG35.sh into Roms/APPS, then reboot.
progdir=$(cd "$(dirname "$0")" && pwd)
exec >"$progdir/GorORG35-logfile.txt" 2>&1
cd "$progdir/GorORG35"
export GOGPU_PLATFORM=fbdev
# Software renderer writes every frame to the framebuffer; skip vsync waits.
export GOGPU_LOG_LEVEL=warn
exec ./goro -config goro.ini -data-dir ./data
