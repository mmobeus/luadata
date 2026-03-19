GOLANGCI_LINT_VERSION := v2.11.3
GOFUMPT_VERSION := v0.9.0
BUMP ?= patch

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SHARED_EXT := .dylib
else ifeq ($(UNAME_S),Linux)
	SHARED_EXT := .so
else
	SHARED_EXT := .dll
endif

.PHONY: build build-clib build-wasm build-docs build-site serve clean test test-python lint fmt fmt-check check setup release validate validate-testdata

build:
	go build -o bin/cli/luadata ./cmd/luadata

build-clib:
	go build -buildmode=c-shared -o bin/clib/libluadata$(SHARED_EXT) ./clib

build-wasm:
	GOOS=js GOARCH=wasm go build -o bin/web/luadata.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" bin/web/
	cp web/index.html web/luadata.js web/app.js bin/web/

build-docs:
	go run ./web/docs/gen -out bin/web/docs

build-site: build-wasm build-docs

serve: build-site
	@echo "Serving at http://localhost:8080"
	@echo "Docs at http://localhost:8080/docs/"
	cd bin/web && python3 -m http.server 8080

clean:
	rm -rf bin

test:
	go test ./...

test-python: build-clib
	cd python && python3 -m pytest tests -v

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

fmt-check:
	@test -z "$$(gofumpt -d .)" || (echo "files need formatting — run 'make fmt'" && gofumpt -d . && exit 1)

check: build test lint fmt-check validate-testdata
	GOOS=js GOARCH=wasm go vet ./cmd/wasm

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

validate: build
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh $(VALIDATE_FLAGS) $(DIR)

validate-testdata: build
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh testdata/valid
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh --expect-fail testdata/invalid

release:
	@bash scripts/release.sh $(BUMP)
