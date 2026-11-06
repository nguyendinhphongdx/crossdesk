.PHONY: all mac windows-linux test site clean

VERSION ?= dev

all: mac windows-linux

# macOS universal app bundle -> dist/Crossdesk-macOS-universal.zip
mac:
	VERSION=$(VERSION) scripts/build-macos.sh

# Windows .exe + Linux tarballs (cross-compiled, no cgo)
windows-linux:
	VERSION=$(VERSION) scripts/build-cross.sh

test:
	go test ./...

site:
	cd site && npm ci && npm run build

clean:
	rm -rf dist site/dist
