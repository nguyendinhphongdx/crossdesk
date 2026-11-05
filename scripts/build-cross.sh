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
    go build -trimpath -ldflags "$LDFLAGS -H windowsgui" -o "$OUT/ShareDevice-Windows-$name.exe" ./cmd/share-device

  dir="$OUT/ShareDevice-Linux-$name"
  rm -rf "$dir"
  mkdir -p "$dir"
  CGO_ENABLED=0 GOOS=linux GOARCH=$arch \
    go build -trimpath -ldflags "$LDFLAGS" -o "$dir/share-device" ./cmd/share-device
  cp packaging/linux/install.sh packaging/linux/share-device-setup \
     packaging/linux/70-share-device.rules packaging/linux/share-device.desktop "$dir/"
  chmod +x "$dir/install.sh" "$dir/share-device-setup" "$dir/share-device"
  tar -C "$OUT" -czf "$OUT/ShareDevice-Linux-$name.tar.gz" "ShareDevice-Linux-$name"
  rm -rf "$dir"
done
ls -la "$OUT"
