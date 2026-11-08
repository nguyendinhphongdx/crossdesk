#!/bin/sh
# Builds "Crossdesk.app" as a universal (arm64 + x86_64) binary and zips it.
# Must run on macOS (the capture/injection code uses cgo + CoreGraphics).
#   VERSION=v1.2.3 scripts/build-macos.sh   ->  dist/Crossdesk-macOS-universal.zip
set -eu
VERSION="${VERSION:-dev}"
OUT=dist
# ".noindex" keeps Spotlight/Launchpad from listing the build output as an
# installed copy of the app.
APP="$OUT/build.noindex/Crossdesk.app"
LDFLAGS="-s -w -X main.version=$VERSION"
mkdir -p "$OUT/mac" "$OUT/build.noindex"

CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC="clang -arch arm64" \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/mac/crossdesk-arm64" ./cmd/crossdesk
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC="clang -arch x86_64" \
  go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/mac/crossdesk-amd64" ./cmd/crossdesk

rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
lipo -create -output "$APP/Contents/MacOS/crossdesk" "$OUT/mac/crossdesk-arm64" "$OUT/mac/crossdesk-amd64"
cp packaging/macos/Info.plist "$APP/Contents/Info.plist"
mkdir -p "$APP/Contents/Resources"
cp packaging/macos/Crossdesk.icns "$APP/Contents/Resources/Crossdesk.icns"
PLAIN_VERSION=$(echo "$VERSION" | sed 's/^v//')
case "$PLAIN_VERSION" in [0-9]*) ;; *) PLAIN_VERSION=0.0.0 ;; esac
plutil -replace CFBundleShortVersionString -string "$PLAIN_VERSION" "$APP/Contents/Info.plist"
plutil -replace CFBundleVersion -string "$PLAIN_VERSION" "$APP/Contents/Info.plist"
# Ad-hoc signatures normally identify the app by the hash of this exact
# build, so macOS forgets the Accessibility grant after every update. Pin the
# designated requirement to the bundle ID so the grant survives updates.
codesign --force --deep --sign - \
  --requirements '=designated => identifier "local.crossdesk.app"' "$APP"

rm -f "$OUT/Crossdesk-macOS-universal.zip"
ditto -c -k --keepParent "$APP" "$OUT/Crossdesk-macOS-universal.zip"
lipo -info "$APP/Contents/MacOS/crossdesk"
echo "built $OUT/Crossdesk-macOS-universal.zip"
