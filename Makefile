.PHONY: all mac-app windows test clean

APP := dist/Share Device.app

all: mac-app windows

# macOS app bundle (double-click to start; opens the control page in the browser).
mac-app:
	go build -o dist/share-device ./cmd/share-device
	rm -rf "$(APP)"
	mkdir -p "$(APP)/Contents/MacOS"
	cp packaging/macos/Info.plist "$(APP)/Contents/Info.plist"
	cp dist/share-device "$(APP)/Contents/MacOS/share-device"
	codesign --force --sign - "$(APP)"

# Windows GUI executables (no console window), cross-compiled without cgo.
windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-H windowsgui" -o dist/ShareDevice.exe ./cmd/share-device
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-H windowsgui" -o dist/ShareDevice-arm64.exe ./cmd/share-device

test:
	go test ./...

clean:
	rm -rf dist
