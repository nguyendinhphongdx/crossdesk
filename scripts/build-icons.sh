#!/bin/sh
# Regenerates every raster icon from assets/brand/*.svg (macOS: needs Google Chrome).
#   scripts/build-icons.sh
set -eu
CH="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
ROOT=$(cd "$(dirname "$0")/.." && pwd)
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

render() { # svg out px
  printf '<!doctype html><style>html,body{margin:0;background:transparent}img{display:block;width:%spx;height:%spx}</style><img src="file://%s">' "$3" "$3" "$1" > "$TMP/r.html"
  "$CH" --headless=new --disable-gpu --hide-scrollbars --default-background-color=00000000 \
    --force-device-scale-factor=1 --window-size="$3,$3" --screenshot="$2" "file://$TMP/r.html" >/dev/null 2>&1
}

B="$ROOT/assets/brand"
render "$B/crossdesk-app-icon.svg" "$TMP/app.png" 1024
render "$B/crossdesk-mark.svg" "$TMP/mark.png" 1024
render "$B/crossdesk-glyph.svg" "$ROOT/internal/ui/assets/menubar@2x.png" 36

# macOS .icns
mkdir -p "$TMP/Crossdesk.iconset"
for s in 16 32 128 256 512; do
  sips -z $s $s "$TMP/app.png" --out "$TMP/Crossdesk.iconset/icon_${s}x${s}.png" >/dev/null
  d=$((s * 2)); sips -z $d $d "$TMP/app.png" --out "$TMP/Crossdesk.iconset/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$TMP/Crossdesk.iconset" -o "$ROOT/packaging/macos/Crossdesk.icns"

# Linux, README, Windows source PNG
sips -z 256 256 "$TMP/app.png" --out "$ROOT/packaging/linux/crossdesk.png" >/dev/null
sips -z 256 256 "$TMP/mark.png" --out "$B/crossdesk-mark-256.png" >/dev/null
sips -z 256 256 "$TMP/mark.png" --out "$ROOT/packaging/windows/crossdesk-256.png" >/dev/null

# Windows .ico (PNG entries)
for s in 16 24 32 48 64 128 256; do sips -z $s $s "$TMP/mark.png" --out "$TMP/ico-$s.png" >/dev/null; done
python3 - "$TMP" "$ROOT/packaging/windows/crossdesk.ico" <<'PY'
import struct, sys
d, out = sys.argv[1], sys.argv[2]
sizes = [16, 24, 32, 48, 64, 128, 256]
imgs = [open(f"{d}/ico-{s}.png", "rb").read() for s in sizes]
buf = struct.pack("<HHH", 0, 1, len(sizes)); off = 6 + 16 * len(sizes)
for s, b in zip(sizes, imgs):
    buf += struct.pack("<BBBBHHII", s % 256, s % 256, 0, 0, 1, 32, len(b), off); off += len(b)
open(out, "wb").write(buf + b"".join(imgs))
PY
cp "$ROOT/packaging/windows/crossdesk.ico" "$ROOT/internal/ui/assets/crossdesk.ico"
cp "$B/crossdesk-mark.svg" "$ROOT/site/public/favicon.svg"
echo "icons regenerated (run go-winres to refresh the Windows .syso; see README)"
