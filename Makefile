# CDImage Makefile

BINARY_NAME=cdimage
GO_FILES=$(shell find . -name "*.go" -type f)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
LDFLAGS=-ldflags="-X main.version=$(VERSION)"

.PHONY: all build clean test install help deps build-cli build-gui

all: build

build: deps
	go build $(LDFLAGS) -o $(BINARY_NAME) .

build-cli: deps
	go build $(LDFLAGS) -tags cli -o $(BINARY_NAME)-cli .

build-gui: deps
	go build $(LDFLAGS) -o $(BINARY_NAME) .

deps:
	go mod download
	go mod tidy

gui-deps:
	@echo "Installing GUI dependencies..."
	@echo "For Ubuntu/Debian: sudo apt-get install libgl1-mesa-dev xorg-dev"
	@echo "For Fedora: sudo dnf install mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel"
	@echo "For Arch: sudo pacman -S libgl libxcursor libxrandr libxinerama libxi"

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

release: clean
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

demo:
	@echo "Running CLI demo..."
	@echo "./$(BINARY_NAME) list-presets"
	@echo "./$(BINARY_NAME) burn -i image.jpg -p verbatim-cd-rw-1"
	@echo "./$(BINARY_NAME) gui"

help:
	@echo "Available targets:"
	@echo "  build     - Build the binary with GUI support"
	@echo "  build-cli - Build CLI-only binary (smaller)"
	@echo "  build-gui - Build with GUI support (default)"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  gui-deps  - Show GUI dependency installation commands"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  install   - Install binary to /usr/local/bin"
	@echo "  uninstall - Remove binary from /usr/local/bin"
	@echo "  release   - Build release binaries for Linux"
	@echo "  demo      - Show usage examples"
	@echo "  help      - Show this help"