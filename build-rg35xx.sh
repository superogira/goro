#!/bin/sh
# Build the RG35XX (H700, linux/arm64) fbdev package of goro.
# Output: dist/rg35xx/{GorORG35/,GorORG35.sh} — copy both into Roms/APPS.
set -e
cd "$(dirname "$0")"

echo "== cross-compiling goro (linux/arm64) =="
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags nofakecgo -ldflags "-s -w" \
  -o dist/rg35xx/GorORG35/goro .

echo "== copying launcher + config =="
# tr -d '\r' guards against a CRLF checkout (core.autocrlf on Windows):
# a CRLF shebang kills the launcher on the device before it can log.
tr -d '\r' < rg35xx/GorORG35.sh > dist/rg35xx/GorORG35.sh
chmod +x dist/rg35xx/GorORG35.sh
tr -d '\r' < rg35xx/goro.ini > dist/rg35xx/GorORG35/goro.ini
# App-list icon: the StockOS frontend shows the PNG named after the
# launcher script (regenerate with `go run ../cmd/genicon` from rg35xx/).
if [ -f rg35xx/GorORG35.png ]; then
  cp rg35xx/GorORG35.png dist/rg35xx/GorORG35.png
fi

echo "== done =="
echo "package: dist/rg35xx/"
echo "next:    copy GorORG35/ + GorORG35.sh (+ GorORG35.png) to Roms/APPS,"
echo "         add data/, set <address> in data/clientinfo.xml to the rAthena LAN IP"
