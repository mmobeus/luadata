# Architecture

luadata is a Lua data parser written in Rust, with bindings for Go, Python, Node.js, WebAssembly, and a standalone CLI. This document is a concise structural reference. For the full story — history, design philosophy, how each binding works under the hood, and lessons learned — see [GUIDE.md](GUIDE.md).

## Overview

```
Rust core library (src/)
├── cdylib (clib/)          → C shared library (.so/.dylib/.dll)
│   └── exports: LuaDataToJSON, LuaDataFree
├── PyO3 module (python/)   → native Python extension
├── napi-rs addon (node/)   → native Node.js addon
├── wasm-bindgen (wasm/)    → WebAssembly module (~124KB)
└── CLI binary (cli/)       → luadata tojson / validate

Go wrapper (go/)
├── Uses purego to load the Rust cdylib at runtime (no CGO)
├── Embeds platform-specific shared libs via go:embed + build tags
└── Exposes TextToJSON, FileToJSON, ToJSON, ReaderToJSON
```

All language bindings share the same Rust parser — there is exactly one
implementation of the lexer, parser, and JSON converter.

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
├── node/                   napi-rs Node.js native addon
│   ├── src/lib.rs          convertLuaToJson / convertLuaFileToJson
│   ├── package.json        Package: @mmobeus/luadata
│   ├── npm/                Platform-specific packages (5 platforms)
│   └── __test__/           Node.js test suite
├── node-wasm/              npm WASM package (@mmobeus/luadata-wasm)
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

For the full narrative on release flow, security model, and publishing details, see [GUIDE.md](GUIDE.md).

Development happens on `main`. Releases are triggered by pushing an RC tag (`v*-rc.*`).

### Quick reference

1. `make release` → tags `v<version>-rc.<n>` on main
2. CI builds cdylib, CLI, and Node.js addon across 5 platforms
3. CI runs Rust + Go tests
4. Release branch prepared with embedded Go libs + version tags
5. GitHub Release created with binaries
6. Parallel publish: PyPI, npm (WASM + native), crates.io — all via OIDC trusted publishing
7. Homebrew tap updated via GitHub App dispatch
8. Version bump PR opened against main

### Publish targets

| Registry | Package | Auth |
|---|---|---|
| crates.io | `luadata`, `luadata_cli` | OIDC (`rust-lang/crates-io-auth-action`) |
| PyPI | `mmobeus-luadata` | OIDC (`pypa/gh-action-pypi-publish`) |
| npm | `@mmobeus/luadata`, `@mmobeus/luadata-wasm` | OIDC (native npm provenance) |
| Homebrew | `mmobeus/tap/luadata` | GitHub App token (scoped to tap repo) |
| pkg.go.dev | `github.com/mmobeus/luadata/go` | None (indexed from release branch) |

### Go consumer model

```
go get github.com/mmobeus/luadata/go@v0.1.13
```

Resolves to the `release` branch, which contains embedded shared libraries. At
runtime, the Go wrapper extracts the platform-appropriate library to a temp file
and loads it via purego. No CGO involved.

On `main`, `embed_dev.go` provides an empty `EmbeddedLib` byte slice — set
`LUADATA_LIB_PATH` to a locally-built library for development.

## Key design decisions

For detailed explanations of each decision, see [GUIDE.md](GUIDE.md).

- **JSON-in, JSON-out C boundary**: The clib accepts and returns JSON strings — no C structs. This keeps the FFI surface at two functions and lets Rust internals evolve without breaking the ABI.
- **purego for Go FFI**: Loads the Rust shared library at runtime using assembly trampolines. No CGO, no C toolchain. `go get` works normally.
- **Embedded shared libs**: The `release` branch embeds platform-specific libraries via `go:embed` + build tags, so Go consumers get everything from `go get`.
- **Platform isolation**: Platform-specific code lives behind build tags in `go/internal/ffi/`. The public API is platform-agnostic.
- **OIDC trusted publishing**: All registry credentials (PyPI, npm, crates.io) use short-lived OIDC tokens. No stored secrets.
