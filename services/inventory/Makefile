# inventory - the inventory's store, its collectors and its HTTP contract.
#
# Plain recipes, one shell per line: macOS ships GNU make 3.81, which has no
# .ONESHELL. /bin/bash is on macOS and Linux alike.
SHELL := /bin/bash

BIN_DIR ?= $(CURDIR)/bin
VERSION ?= $(shell git describe --always --dirty 2>/dev/null || echo dev)
DB      ?= $(CURDIR)/inventory.db
ADDR    ?= 127.0.0.1:8740
XML     ?=

.PHONY: help build build-arm64 test check fmt vet run collect clean

help:
	@echo "inventory - available targets"
	@echo
	@echo "  build        build inventory and inventoryd into $(BIN_DIR)"
	@echo "  build-arm64  cross-compile both for a Raspberry Pi (linux/arm64)"
	@echo "  test         unit tests"
	@echo "  check        fmt check, vet, and the tests"
	@echo "  collect      collect an iTunes library: make collect XML=\"iTunes Library.xml\""
	@echo "  run          run inventoryd on $(ADDR), on DB=$(DB)"

build:
	mkdir -p "$(BIN_DIR)"
	go build -o "$(BIN_DIR)/inventory" ./cmd/inventory
	go build -ldflags "-X main.version=$(VERSION)" -o "$(BIN_DIR)/inventoryd" ./cmd/inventoryd

# No cgo anywhere in the module (ADR-0001), so the Mac builds the Pi's binaries directly.
build-arm64:
	mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "$(BIN_DIR)/inventory-linux-arm64" ./cmd/inventory
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o "$(BIN_DIR)/inventoryd-linux-arm64" ./cmd/inventoryd

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	go vet ./...
	go test ./...

collect:
	@test -n "$(XML)" || { echo "XML= names the library's XML"; exit 2; }
	INV_DB="$(DB)" go run ./cmd/inventory collect itunes "$(XML)"

run:
	INV_DB="$(DB)" INV_ADDR="$(ADDR)" go run -ldflags "-X main.version=$(VERSION)" ./cmd/inventoryd

clean:
	rm -rf "$(BIN_DIR)"
