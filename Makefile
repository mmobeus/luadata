GOLANGCI_LINT_VERSION := v2.11.3
GOFUMPT_VERSION := v0.9.0
BUMP ?= patch

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	SHARED_EXT := .dylib
	SHARED_PREFIX := lib
else ifeq ($(UNAME_S),Linux)
	SHARED_EXT := .so
	SHARED_PREFIX := lib
else
	SHARED_EXT := .dll
	SHARED_PREFIX :=
endif

UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
	GO_PLATFORM := $(shell echo $(UNAME_S) | tr A-Z a-z)_amd64
else ifeq ($(UNAME_M),arm64)
	GO_PLATFORM := $(shell echo $(UNAME_S) | tr A-Z a-z)_arm64
else ifeq ($(UNAME_M),aarch64)
	GO_PLATFORM := $(shell echo $(UNAME_S) | tr A-Z a-z)_arm64
endif

.PHONY: build build-clib build-wasm build-npm build-docs build-site serve clean \
	test test-rust test-go test-python test-node \
	lint fmt fmt-check check setup release validate validate-testdata smoke-test \
	build-node

# ── Rust targets ──────────────────────────────────────────────────

build:
	cargo build -p luadata_cli --release
	mkdir -p bin/cli
	cp target/release/luadata bin/cli/

build-clib:
	cargo build -p luadata_clib --release
	mkdir -p bin/clib
	cp target/release/$(SHARED_PREFIX)luadata_clib$(SHARED_EXT) bin/clib/$(SHARED_PREFIX)luadata$(SHARED_EXT)

build-clib-go: build-clib
	mkdir -p go/internal/ffi/lib/$(GO_PLATFORM)
	cp bin/clib/$(SHARED_PREFIX)luadata$(SHARED_EXT) go/internal/ffi/lib/$(GO_PLATFORM)/

SITE_VERSION ?= dev

build-wasm:
	cargo install wasm-pack 2>/dev/null || true
	wasm-pack build wasm --target web --out-dir ../bin/web/pkg
	rm -f bin/web/pkg/.gitignore
	cp web/luadata.js web/app.js bin/web/
	sed 's/__VERSION__/$(SITE_VERSION)/' web/index.html > bin/web/index.html

build-node-wasm:
	cargo install wasm-pack 2>/dev/null || true
	wasm-pack build wasm --target bundler --out-dir ../node-wasm/wasm
	rm -f node-wasm/wasm/package.json node-wasm/wasm/.gitignore

build-docs:
	cd web/docs/gen && go run . -out ../../../bin/web/docs

build-site: build-wasm build-docs

serve: build-site
	@echo "Serving at http://localhost:8080"
	@echo "Docs at http://localhost:8080/docs/"
	cd bin/web && python3 -m http.server 8080

clean:
	rm -rf bin target node-wasm/wasm

# ── Test targets ──────────────────────────────────────────────────

test: test-rust test-go

test-rust:
	cargo test -p luadata

test-go: build-clib
	LUADATA_LIB_PATH=$(CURDIR)/bin/clib/$(SHARED_PREFIX)luadata$(SHARED_EXT) \
		go test -C go -v ./...

test-python:
	cd python && uv run --extra test maturin develop
	cd python && uv run --extra test pytest tests -v

build-node:
	cd node && npm install && npm run build

test-node: build-node
	node node/__test__/index.spec.mjs

# ── Lint / format ─────────────────────────────────────────────────

lint:
	cargo clippy --workspace -- -D warnings

fmt:
	cargo fmt --all
	gofumpt -w go/

fmt-check:
	@cargo fmt --all -- --check
	@test -z "$$(gofumpt -d go/)" || (echo "files need formatting — run 'make fmt'" && gofumpt -d go/ && exit 1)

check: build test-rust lint fmt-check validate-testdata
	cargo check -p luadata_python
	cargo check -p luadata-wasm
	cargo check -p luadata_node

# ── Setup ─────────────────────────────────────────────────────────

setup:
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	rustup component add clippy rustfmt

# ── Validate ──────────────────────────────────────────────────────

validate: build
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh $(VALIDATE_FLAGS) $(DIR)

validate-testdata: build
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh testdata/valid
	@PATH="$(CURDIR)/bin/cli:$(PATH)" bash scripts/validate-folder.sh --expect-fail testdata/invalid

# ── Release ───────────────────────────────────────────────────────

release:
	@bash scripts/release.sh $(BUMP)

smoke-test:
	@bash scripts/smoke-test-release.sh $(VERSION)
