#!/usr/bin/env bash
set -euo pipefail

SVG="wiredoor.svg"
ICONS_DIR="build/windows/icons/"

WINRES_TPL="build/windows/winres/winres.json.tpl"
WINRES="build/windows/winres/winres.json"

curl -Lo "$SVG" "https://www.wiredoor.net/images/wiredoor.svg"

mkdir -p "$ICONS_DIR"

for s in 16 24 32 48 64 128 256; do
  rsvg-convert -w "$s" -h "$s" "$SVG" -o "$ICONS_DIR/${s}x${s}.png"
done

magick \
  "$ICONS_DIR/16x16.png" \
  "$ICONS_DIR/24x24.png" \
  "$ICONS_DIR/32x32.png" \
  "$ICONS_DIR/48x48.png" \
  "$ICONS_DIR/64x64.png" \
  "$ICONS_DIR/128x128.png" \
  "$ICONS_DIR/256x256.png" \
  "$ICONS_DIR/wiredoor.ico"

export VERSION
envsubst < "$WINRES_TPL" > "$WINRES"

go install github.com/tc-hib/go-winres@latest
go-winres make --in build/windows/winres/winres.json
