.PHONY: all mac windows test clean

all: mac windows

mac:
	go build -o dist/share-device ./cmd/share-device

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/share-device.exe ./cmd/share-device
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -o dist/share-device-arm64.exe ./cmd/share-device

test:
	go test ./...

clean:
	rm -rf dist
