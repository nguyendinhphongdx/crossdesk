#!/bin/sh
# Builds "Share Device.app" as a universal (arm64 + x86_64) binary and zips it.
# Must run on macOS (the capture/injection code uses cgo + CoreGraphics).
#   VERSION=v1.2.3 scripts/build-macos.sh   ->  dist/ShareDevice-macOS-universal.zip
set -eu
VERSION="${VERSION:-dev}"
OUT=dist
APP="$OUT/Share Device.app"
LDFLAGS="-s -w -X main.version=$VERSION"
mkdir -p "$OUT/mac"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="clang -arch arm64" \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/mac/share-device-arm64" ./cmd/share-device
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/mac/share-device-amd64" ./cmd/share-device

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
lipo -create -output "$APP/Contents/MacOS/share-device" "$OUT/mac/share-device-arm64" "$OUT/mac/share-device-amd64"
cp packaging/macos/Info.plist "$APP/Contents/Info.plist"
PLAIN_VERSION=$(echo "$VERSION" | sed 's/^v//')
case "$PLAIN_VERSION" in [0-9]*) ;; *) PLAIN_VERSION=0.0.0 ;; esac
plutil -replace CFBundleShortVersionString -string "$PLAIN_VERSION" "$APP/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$PLAIN_VERSION" "$APP/Contents/Info.plist"
codesign --force --deep --sign - "$APP"

rm -f "$OUT/ShareDevice-macOS-universal.zip"
(cd "$OUT" && ditto -c -k --keepParent "Share Device.app" ShareDevice-macOS-universal.zip)
lipo -info "$APP/Contents/MacOS/share-device"
echo "built $OUT/ShareDevice-macOS-universal.zip"
