.PHONY: all build test wasm serve coverage check pre-commit pre-commit-install codespell clean install

CMDS = dashcheck
BINARIES = $(addprefix out/,$(CMDS))
GOROOT := $(shell go env GOROOT)
PORT ?= 8080

all: check build wasm test

build: $(BINARIES)

# Binaries are built as packages, not as main.go files, so that they carry the
# version Go embeds from the git tag and commit (see internal/buildinfo.go).
# They are .PHONY because that version is not a file prerequisite: a binary
# built before a commit or a tag would be kept, still naming the old one. The
# build cache makes the rebuild cheap.
.PHONY: $(BINARIES)
$(BINARIES): out/%:
	go build -o $@ ./cmd/$*

test:
	go test ./...

wasm:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags='-s -w' -o web/dashcheck.wasm ./cmd/dashcheck-wasm
	cp $(GOROOT)/lib/wasm/wasm_exec.js web/

serve: wasm
	python3 tools/serve.py web $(PORT)

coverage:
	go test -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out -o coverage.txt
	@echo "Coverage report: coverage.html"

# The second run lints the packages that are only built for the browser.
check:
	golangci-lint run
	GOOS=js GOARCH=wasm golangci-lint run ./fetch ./cmd/dashcheck-wasm

pre-commit-install: .venv/bin/pre-commit
	.venv/bin/pre-commit install

pre-commit: .venv/bin/pre-commit
	.venv/bin/pre-commit run --all-files

.venv/bin/pre-commit .venv/bin/codespell:
	python3 -m venv .venv
	.venv/bin/pip install pre-commit codespell

# The DASH-IF inventory quotes the upstream messages, typos included, and
# guidlines is in a profile URL that the DASH-IF tool misspells.
codespell: .venv/bin/codespell
	.venv/bin/codespell -S out,references,coverage.html,'*.wasm',wasm_exec.js,dashif-checks.json,dashif-coverage.md \
		-L repid,trun,truns,hask,anc,guidlines

clean:
	rm -rf out/ web/dashcheck.wasm web/wasm_exec.js coverage.out coverage.html coverage.txt .venv/

install:
	go install $(addprefix ./cmd/,$(CMDS))
