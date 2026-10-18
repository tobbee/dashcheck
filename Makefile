.PHONY: all build test wasm serve lint clean

GOROOT := $(shell go env GOROOT)
PORT ?= 8080

all: test build wasm

build:
	go build -o out/dashcheck ./cmd/dashcheck

test:
	go test ./...

wasm:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o web/dashcheck.wasm ./cmd/dashcheck-wasm
	cp $(GOROOT)/lib/wasm/wasm_exec.js web/

serve: wasm
	cd web && python3 -m http.server $(PORT) --bind 127.0.0.1

lint:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./fetch ./cmd/dashcheck-wasm

clean:
	rm -rf out web/dashcheck.wasm web/wasm_exec.js
