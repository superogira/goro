#!/bin/sh
# Build the RG35XX (H700, linux/arm64) fbdev package of goro.
# Output: dist/rg35xx/{GorORG35/,GorORG35.sh} — copy both into Roms/APPS.
set -e
cd "$(dirname "$0")"

echo "== cross-compiling goro (linux/arm64) =="
# The build id feeds the boot-time self update (app.RunSelfUpdate
# compares it against version.txt on the update server).
VERSION="$(git rev-parse --short HEAD)-$(date +%Y%m%d%H%M)"
echo "build version: $VERSION"
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 \
  go build -tags nofakecgo \
  -ldflags "-s -w -X github.com/kivutar/goro/app.buildVersion=$VERSION" \
  -o dist/rg35xx/GorORG35/goro .
# version.txt + checksum for publishing to the update server.
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s\n%s\n' "$VERSION" "$(sha256sum dist/rg35xx/GorORG35/goro | cut -d' ' -f1)" \
    > dist/rg35xx/version.txt
fi

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
