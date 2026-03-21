# Architecture

luadata is a Lua data parser written in Rust, with bindings for Go, Python,
WebAssembly, and a standalone CLI. This document describes the architecture and
the reasoning behind it.

## Overview

```
Rust core library (src/)
├── cdylib (clib/)          → C shared library (.so/.dylib/.dll)
│   └── exports: LuaDataToJSON, LuaDataFree
├── PyO3 module (python/)   → native Python extension
├── wasm-bindgen (wasm/)    → WebAssembly module (~124KB)
└── CLI binary (cli/)       → luadata tojson / validate

Go wrapper (go/)
├── Uses purego to load the Rust cdylib at runtime (no CGO)
├── Embeds platform-specific shared libs via go:embed + build tags
└── Exposes TextToJSON, FileToJSON, ToJSON, ReaderToJSON
```

All language bindings share the same Rust parser — there is exactly one
implementation of the lexer, parser, and JSON converter.

## Why Rust as source of truth

Before the rewrite, luadata was a pure Go library. Go served the core use case
well, but adding Python and WebAssembly support exposed limitations:

- **WASM size**: Go's WASM binary was ~3.2MB (runtime overhead). The Rust WASM
  module is ~124KB — roughly 25x smaller.
- **Python integration**: The Go approach required building a CGO shared library
  and loading it via ctypes, with manual JSON-envelope marshalling. Rust/PyO3
  compiles directly to a native Python extension with native types.
- **Single parser**: Maintaining one parser instead of porting logic across
  languages eliminates behavioral drift between platforms.
- **Go compatibility**: The main objection to Rust was that Go consumers would
  need CGO. With [purego](https://github.com/ebitengine/purego), Go loads the
  Rust shared library at runtime with `CGO_ENABLED=0` — no C toolchain required.

## Repository structure

```
luadata/
├── src/                    Rust core library
│   ├── lib.rs              Module exports
│   ├── lexer.rs            Hand-written character-by-character lexer
│   ├── parser.rs           Recursive descent parser
│   ├── converter.rs        Lua → JSON conversion
│   ├── options.rs          ParseConfig, ArrayMode, EmptyTableMode, StringTransform
│   └── types.rs            Key, Value, KeyValuePairs, RawValue, RawKey
├── cli/                    CLI binary (clap)
│   └── src/main.rs         tojson + validate subcommands
├── clib/                   C shared library (cdylib)
│   └── src/lib.rs          LuaDataToJSON / LuaDataFree FFI exports
├── python/                 PyO3 Python module
│   ├── src/lib.rs          lua_to_json / lua_to_dict
│   ├── pyproject.toml      Package: mmobeus-luadata
│   └── tests/              pytest suite
├── wasm/                   wasm-bindgen module
│   └── src/lib.rs          convertLuaDataToJson (JS name)
├── npm/                    npm package (mmobeus-luadata)
│   ├── package.json        Package metadata
│   ├── index.js            ES module wrapper: init() + convert()
│   ├── index.d.ts          TypeScript type definitions
│   └── wasm/               (built at release time, gitignored)
├── go/                     Pure Go wrapper (no CGO)
│   ├── luadata.go          Public API: TextToJSON, FileToJSON, etc.
│   ├── options.go          Functional options: WithArrayMode, etc.
│   ├── luadata_test.go     Go test suite
│   └── internal/ffi/       purego FFI bridge
│       ├── ffi.go          Call(), init, library loading
│       ├── ffi_unix.go     purego.Dlopen (darwin/linux)
│       ├── ffi_windows.go  syscall.LoadLibrary
│       └── embed_dev.go    Empty embed (dev); replaced on release branch
├── web/                    Browser interface
│   ├── index.html          Live converter UI
│   ├── luadata.js          ES module wrapper: init() + convert()
│   ├── app.js              UI logic
│   └── docs/gen/           "Luadata by Example" generator
├── testdata/               Shared test fixtures
│   ├── valid/              .lua files that must parse
│   └── invalid/            .lua files that must fail
├── scripts/
│   ├── release.sh          Tag RC on main
│   ├── prepare-release.sh  Stage libs + generate embeds on release branch
│   └── validate-folder.sh  Validate testdata against CLI
├── Cargo.toml              Workspace root
└── Makefile                Build, test, lint, release targets
```

## Release workflow

Development happens on `main`. The `go/internal/ffi/lib/` directory is
gitignored — local development uses `make build-clib` to populate it, or sets
`LUADATA_LIB_PATH` to point at a locally-built shared library.

### Cutting a release

1. **Tag RC on main**: `make release` (or `make release BUMP=minor`, etc.) tags
   the current commit as `v<version>-rc.1` and pushes the tag.

2. **CI cross-compiles**: The `release.yml` workflow triggers on `v*-rc.*` tags.
   It builds the Rust cdylib for five platforms:
   - `darwin_amd64`, `darwin_arm64`
   - `linux_amd64`, `linux_arm64`
   - `windows_amd64`

3. **CI tests**: Rust tests and Go tests (using the freshly-built Linux library)
   run in the same workflow.

4. **Prepare release branch**: `scripts/prepare-release.sh` copies the
   cross-compiled shared libraries into `go/internal/ffi/lib/<platform>/`,
   generates platform-specific `go:embed` files (replacing `embed_dev.go`), and
   commits everything to the `release` branch with the final version tag.

5. **GitHub Release**: CI creates a GitHub Release with shared library artifacts
   and CLI binary tarballs (macOS Intel + Apple Silicon).

6. **Publish to registries**: After the release succeeds, publish jobs run in
   parallel — all using OIDC trusted publishing (no stored secrets):
   - **PyPI**: Builds platform-specific wheels (same five platforms as clib) plus
     an sdist, then publishes `mmobeus-luadata` via `pypa/gh-action-pypi-publish`.
   - **npm**: Builds the WASM module with wasm-pack, packages it with the JS/TS
     wrapper from `npm/`, and publishes `mmobeus-luadata`.
   - **crates.io**: Publishes the `luadata` core crate via
     `rust-lang/crates-io-auth-action` for OIDC token exchange.

7. **Homebrew**: After the GitHub Release is created, the workflow dispatches a
   `repository_dispatch` event to
   [`mmobeus/homebrew-tap`](https://github.com/mmobeus/homebrew-tap) with the
   version and SHA256 hashes of the CLI tarballs. The tap repo's workflow updates
   the formula and pushes the commit. Users install with
   `brew tap mmobeus/tap && brew install luadata`.

### Homebrew integration

The CLI binary (`luadata`) is built for macOS Intel (`darwin_amd64`) and Apple
Silicon (`darwin_arm64`) in the `build-cli` job, which runs in parallel with
`build-clib`. The binaries are packaged as tarballs and uploaded to the GitHub
Release.

Authentication to the tap repo uses a **GitHub App** (`mmobeus-homebrew-updater`)
rather than a personal access token. The app is installed only on `homebrew-tap`
and has Contents + Actions read/write permissions. Two org-level secrets provide
the credentials:

- `HOMEBREW_APP_ID`: The app's numeric ID
- `HOMEBREW_APP_PRIVATE_KEY`: The app's `.pem` private key

At runtime, [`actions/create-github-app-token`](https://github.com/actions/create-github-app-token)
mints a short-lived token scoped to `homebrew-tap`. This token is used to send
the `repository_dispatch` event that triggers the formula update.

See the [homebrew-tap README](https://github.com/mmobeus/homebrew-tap) for the
full setup details, including how to add formulas for other projects.

### Go consumer model

Go consumers install with:

```
go get github.com/mmobeus/luadata/go@v0.5.0
```

This resolves to the `release` branch commit, which contains the embedded shared
libraries. At runtime, the Go wrapper extracts the platform-appropriate library
to a temp file and loads it via purego. No CGO is involved at any point.

On `main`, `embed_dev.go` provides an empty `EmbeddedLib` byte slice. If
`LUADATA_LIB_PATH` is set, the FFI layer loads the library from that path
instead — this is the local development path.

## Key design decisions

### purego FFI

The Go wrapper uses [purego](https://github.com/ebitengine/purego) to call the
Rust shared library. purego uses assembly trampolines to make C function calls
from pure Go — no CGO, no C compiler, no `CGO_ENABLED=1`. This means:

- `go get` works without a C toolchain
- Cross-compilation works (libraries are pre-built per platform)
- The Go module is a normal Go package from the consumer's perspective

### internal/ffi for platform isolation

Platform-specific code (library loading, embed directives) lives in
`go/internal/ffi/` behind build tags. The public API in `go/luadata.go` is
platform-agnostic and delegates to `ffi.Call()`.

### runtime.KeepAlive for GC safety

When passing Go byte slices to the Rust FFI as C pointers, `runtime.KeepAlive`
ensures the Go garbage collector doesn't collect the backing memory while Rust
is reading it.

### Null-terminated C strings for UTF-8 correctness

The FFI boundary uses null-terminated C strings (`*const c_char` on the Rust
side). The Go wrapper converts Go strings to null-terminated byte slices before
passing them. This avoids length-prefix mismatches and ensures Rust's `CStr`
can safely interpret the data as UTF-8.

### JSON envelope for FFI results

The clib returns results as a JSON envelope: `{"result":"..."}` on success,
`{"error":"..."}` on failure. This keeps the FFI surface minimal (two functions:
`LuaDataToJSON` and `LuaDataFree`) while supporting structured error reporting.
