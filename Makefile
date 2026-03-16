GOLANGCI_LINT_VERSION := v2.11.3
GOFUMPT_VERSION := v0.9.0
BUMP ?= patch

.PHONY: build build-wasm serve clean test lint fmt fmt-check check setup release validate validate-testdata

build:
	go build -o bin/cli/luadata ./cmd/luadata

build-wasm:
	GOOS=js GOARCH=wasm go build -o bin/web/luadata.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" bin/web/
	cp web/index.html bin/web/

serve: build-wasm
	@echo "Serving at http://localhost:8080"
	cd bin/web && python3 -m http.server 8080

clean:
	rm -rf bin

test:
	go test ./...

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
