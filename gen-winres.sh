#!/usr/bin/env bash
set -euo pipefail

SVG="wiredoor.svg"
ICONS_DIR="build/windows/icons/"

curl -Lo "$SVG" "https://www.wiredoor.net/images/wiredoor.svg"

mkdir -p "$ICONS_DIR"

for s in 16 24 32 48 64 128 256; do
  rsvg-convert -w "$s" -h "$s" "$SVG" -o "$ICONS_DIR/${s}x${s}.png"
done

WINRES_TPL="build/windows/winres/winres.json.tmpl"
WINRES="build/windows/winres/winres.json"

export VERSION
envsubst < "$WINRES_TPL" > "$WINRES"

go install github.com/tc-hib/go-winres@latest
go-winres make --in build/windows/winres/winres.json
