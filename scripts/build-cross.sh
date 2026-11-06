#!/bin/sh
# Cross-compiles Windows and Linux builds (pure Go, no cgo) from any OS.
#   VERSION=v1.2.3 scripts/build-cross.sh
set -eu
VERSION="${VERSION:-dev}"
OUT=dist
LDFLAGS="-s -w -X main.version=$VERSION"
mkdir -p "$OUT"

for arch in amd64 arm64; do
  name=$([ "$arch" = amd64 ] && echo x64 || echo arm64)
  CGO_ENABLED=0 GOOS=windows GOARCH=$arch \
    go build -trimpath -ldflags "$LDFLAGS -H windowsgui" -o "$OUT/Crossdesk-Windows-$name.exe" ./cmd/crossdesk

  dir="$OUT/Crossdesk-Linux-$name"
  rm -rf "$dir"
  mkdir -p "$dir"
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch \
    go build -trimpath -ldflags "$LDFLAGS" -o "$dir/crossdesk" ./cmd/crossdesk
  cp packaging/linux/install.sh packaging/linux/crossdesk-setup \
     packaging/linux/70-crossdesk.rules packaging/linux/crossdesk.desktop packaging/linux/crossdesk.png "$dir/"
  chmod +x "$dir/install.sh" "$dir/crossdesk-setup" "$dir/crossdesk"
  tar -C "$OUT" -czf "$OUT/Crossdesk-Linux-$name.tar.gz" "Crossdesk-Linux-$name"
  rm -rf "$dir"
done
ls -la "$OUT"
