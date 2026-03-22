# Guide: Rust as a universal core

This document tells the story of how luadata evolved from a single-language Go utility into a multi-language Rust project, and explains how each piece works under the hood. If you're interested in building a core Rust library and shipping it to multiple language ecosystems, this is a walkthrough of one way to do it.

For usage documentation, see the [README](README.md). For a concise structural reference, see [ARCHITECTURE.md](ARCHITECTURE.md).

## How we got here

luadata started as a pure Go library. It was built for a Go system that needed to parse World of Warcraft SavedVariables files — Lua data files that store addon state. The parser worked well, and it was eventually extracted into its own module so it could be reused.

Once the util was pulled into its own project, I wanted to look into how to call it from python. That lead to using a light wrapper go library that used CGO (so main go repo was still non-CGO). 

After that, I wanted to try WASM. It worked well! 

So, it sort of kept going from there. It became a learning project: how can I reuse this util in different ecosystems. Rust was next. 

I don't think I ever actually did the Rust calling the golang C shared libs. I wanted to try a port to Rust, and then compare that with the golang impl. The Go WASM target produced a ~3.2MB binary (Go's runtime overhead), and Rust's clocked in at ~124KB. Roughly 25x smaller. Testing the 2 cli's showed that parsing was also noticeably faster in the Rust version: 4x faster or completed in 25% of the time it took golang, across ~200 files and 320MB of JSON.

The Rust version was clearly better as a core, but that created a problem: the existing Go system still needed the parser. Go's standard approach to calling C/Rust code is CGO, which requires a C toolchain...and then running it with CGO enabled which has challenges, and is generally avoided when possible...so it isn't great to have a shared util need CGO, causing all things that want to use it (mine included) to have to enable CGO just for it. For my purposes here, [purego](https://github.com/ebitengine/purego) solved this: it loads shared libraries at runtime using assembly trampolines, with `CGO_ENABLED=0`. No CGO needed. 

With the Rust core and Go bridge working, I reworked the python impl, then wasm. And eventually, native node. Each binding became a learning exercise in a different part of the Rust ecosystem — PyO3, napi-rs, wasm-bindgen — and in how to package and publish to each language's registry. The project became less about the parser itself (it's quite simple) and more about the infrastructure: how do you take a core Rust utility and make it available everywhere, with proper packaging, CI/CD, and security?

This guide documents how we do that. It's not the only way, but it's one complete, working example.

## Architecture at a glance

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

All language bindings share the same Rust parser — there is exactly one implementation of the lexer, parser, and JSON converter.

## How each binding works

### Rust core (`src/`)

The core library is a pure Rust crate with no unsafe code. It exposes:

```rust
pub fn text_to_json(name: &str, text: &str, config: ParseConfig) -> Result<String, String>;
pub fn file_to_json(path: &str, config: ParseConfig) -> Result<String, String>;
```

Internally, it's a hand-written lexer (`lexer.rs`) feeding a recursive descent parser (`parser.rs`), with a converter (`converter.rs`) that walks the parse tree and emits JSON. Configuration options (array detection, empty table handling, string transforms) are applied during conversion.

Every other binding ultimately calls into this code.

### C shared library (`clib/`)

The C library is the bridge layer for FFI-based bindings. It exposes two functions:

```c
char* LuaDataToJSON(const char* input, const char* options);
void  LuaDataFree(char* ptr);
```

A key design decision: **options are passed as a JSON string, not as C structs**. This means the C ABI surface is just two functions and two string parameters. Adding new options to the Rust core (like `StringTransform`) requires zero changes to the C interface — the JSON is parsed on the Rust side.

Results use a JSON envelope: `{"result": "..."}` on success, `{"error": "..."}` on failure. Memory ownership follows a simple rule: `LuaDataToJSON` allocates and returns a string via `CString::into_raw()`; the caller must free it with `LuaDataFree`, which reclaims it via `CString::from_raw()`.

Currently only the Go binding uses this layer, but any FFI-capable language could. Originally, this was build from golang C wrapper, and used in the python packaging...funny how it flipped.

### Go (`go/`)

The Go binding is the most architecturally interesting because of the constraint: no CGO. I wasn't sure about purego at first. Then I stubled on to [this post](https://www.yuchanns.xyz/posts/bridging-rust-and-native-go/) which talked about a path to running OpenDAL in golang without CGO. My use case was kept simple, so I didn't need to go all the way to the purego + libbfi solution that is described there, but I may do that later as another experiment to see it in action.

**Runtime FFI with purego.** [purego](https://github.com/ebitengine/purego) uses assembly trampolines to call C functions from pure Go. 

For our impl (just using 'purego' lib), the FFI layer (`go/internal/ffi/`) registers function pointers at init time:

```go
purego.RegisterLibFunc(&pLuaDataToJSON, lib, "LuaDataToJSON")
purego.RegisterLibFunc(&pLuaDataFree, lib, "LuaDataFree")
```

Platform-specific loading happens behind build tags: `purego.Dlopen` on Unix, `syscall.LoadLibrary` on Windows.

**Library distribution via `go:embed`.** This is where it gets interesting. On the `main` branch, `embed_dev.go` provides an empty `EmbeddedLib` byte slice — local development uses `LUADATA_LIB_PATH` to point at a locally-built shared library. On the `release` branch, the CI replaces `embed_dev.go` with platform-specific files like `embed_darwin_arm64.go`:

```go
//go:build darwin && arm64

//go:embed lib/darwin_arm64/libluadata_clib.dylib
var EmbeddedLib []byte
```

When a Go consumer runs `go get github.com/mmobeus/luadata/go@v0.1.13`, Go resolves the module from the release branch. At runtime, the embedded library is extracted to a temp directory and loaded via purego. The consumer never sees a shared library file — it's just a normal Go import.

**GC safety.** When passing Go strings to the C FFI as pointers, `runtime.KeepAlive()` ensures the Go garbage collector doesn't collect the backing memory while Rust is reading it. This is easy to forget and causes intermittent crashes if missed.

**Idiomatic API.** The public API uses Go's functional options pattern:

```go
reader, err := luadata.TextToJSON("input", luaString,
    luadata.WithArrayMode("sparse", 10),
    luadata.WithEmptyTableMode("array"),
)
```

All functions return `io.Reader`, letting callers stream or buffer as needed.

### Python (`python/`)

The Python binding uses [PyO3](https://pyo3.rs/) to compile Rust directly into a native Python extension module (`.so` on Unix, `.pyd` on Windows). There's no C layer involved — PyO3 handles the CPython ABI.

Two functions are exposed: `lua_to_json()` returns a JSON string, and `lua_to_dict()` returns a Python dictionary. A pragmatic shortcut: `lua_to_dict` calls `lua_to_json` internally and then runs Python's `json.loads()` on the result, rather than building Python objects directly from Rust. This keeps the binding code simple at a negligible performance cost.

[maturin](https://www.maturin.rs/) handles building and packaging. It produces platform-specific wheels that pip can install normally.

### Node.js native (`node/`)

The Node.js binding uses [napi-rs](https://napi.rs/) to compile Rust into a `.node` binary addon. Like PyO3, there's no C layer — napi-rs handles the N-API boundary directly.

napi-rs provides automatic struct deserialization: JavaScript objects passed as function arguments are mapped to Rust structs annotated with `#[napi(object)]`. This means the binding code is mostly type definitions and thin wrappers.

**Platform distribution.** napi-rs uses a pattern where the root npm package (`@mmobeus/luadata`) has `optionalDependencies` on platform-specific packages (`@mmobeus/luadata-darwin-arm64`, `@mmobeus/luadata-linux-x64-gnu`, etc.). When a user runs `npm install`, npm installs only the package matching their platform. The root package's `index.js` detects the platform and loads the correct `.node` binary.

### WebAssembly (`wasm/` + `node-wasm/`)

The WASM binding uses [wasm-bindgen](https://rustwasm.github.io/wasm-bindgen/) to compile Rust to WebAssembly with JavaScript interop. The build tool is [wasm-pack](https://rustwasm.github.io/wasm-pack/).

There are two build targets:

- `--target web` produces output for direct browser use (loaded as an ES module)
- `--target bundler` produces output for npm, consumed by bundlers like webpack or vite

The `node-wasm/` directory wraps the bundler output in a clean npm package with an `init()` + `convert()` API. `init()` loads the WASM module (async, must be called once); `convert()` is synchronous after that.

One quirk: wasm-bindgen doesn't auto-deserialize JavaScript objects the way napi-rs does. Option parsing uses `js_sys::Reflect::get()` to manually read fields from the JS options object. The binding also uses an error envelope pattern (returns `{result, error}` instead of throwing) to keep error handling predictable across the WASM boundary.

### CLI (`cli/`)

The CLI is the simplest binding — a Rust binary that calls the core library directly, with [clap](https://docs.rs/clap/) for argument parsing. No FFI, no serialization boundary. It supports `tojson` (convert) and `validate` (check syntax) subcommands.

Distributed via `cargo install luadata_cli` and through a Homebrew tap (`brew install mmobeus/tap/luadata`).

## Packaging and publishing

Each ecosystem has its own packaging conventions, build tools, and registry. Here's what gets published where:

| Ecosystem | Package | Build Tool | Registry | What's Published |
|---|---|---|---|---|
| Rust | `luadata` | cargo | crates.io | Source crate |
| Rust (CLI) | `luadata_cli` | cargo | crates.io | Source crate (binary) |
| Python | `mmobeus-luadata` | maturin | PyPI | Platform wheels (5) + sdist |
| Node.js | `@mmobeus/luadata` | napi-rs | npm | Root package + 5 platform packages |
| WASM | `@mmobeus/luadata-wasm` | wasm-pack | npm | WASM binary + JS/TS wrapper |
| Go | `github.com/mmobeus/luadata/go` | go:embed | pkg.go.dev | Source + embedded shared libs |
| CLI (Homebrew) | `mmobeus/tap/luadata` | cargo | Homebrew | Pre-built macOS binaries |

### The Go release branch

The Go distribution model deserves special mention because it's unusual. Most Go modules are pure source code — `go get` downloads source and compiles it. But luadata's Go wrapper loads a pre-built Rust shared library at runtime, so the library must be distributed alongside the Go source.

The solution is a `release` branch that contains everything on `main` plus the cross-compiled shared libraries embedded in Go source files. The CI:

1. Builds the Rust cdylib for all five platforms
2. Copies each library into `go/internal/ffi/lib/<platform>/`
3. Generates a Go file per platform with `//go:build` constraints and `//go:embed` directives
4. Commits to the `release` branch and tags it (e.g., `v0.1.13` + `go/v0.1.13`)

The dual tag (`v0.1.13` for the root module, `go/v0.1.13` for the Go submodule) is required by Go's module system for subdirectory modules.

### Node.js platform packages

napi-rs uses a multi-package pattern for cross-platform native addons. Instead of one npm package with all platform binaries (which would bloat installs), it publishes:

- `@mmobeus/luadata` — the root package with JS/TS entry points
- `@mmobeus/luadata-darwin-arm64` — macOS Apple Silicon binary
- `@mmobeus/luadata-darwin-x64` — macOS Intel binary
- `@mmobeus/luadata-linux-x64-gnu` — Linux x86_64 binary
- `@mmobeus/luadata-linux-arm64-gnu` — Linux ARM64 binary
- `@mmobeus/luadata-win32-x64-msvc` — Windows x86_64 binary

The root package lists the platform packages as `optionalDependencies`. npm installs only the one matching the user's platform.

## Release flow

Releases are triggered by pushing an RC (release candidate) tag to `main`:

1. **`make release`** computes the next version, tags the commit as `v<version>-rc.<n>`, and pushes the tag
2. **CI builds** the Rust cdylib, CLI binary, and Node.js native addon — each across five platforms
3. **CI tests** run Rust and Go tests against the freshly-built artifacts
4. **Release branch** is prepared: shared libraries embedded, version files updated, final tag created
5. **GitHub Release** is created with CLI tarballs and shared library artifacts
6. **Registries** are published to in parallel: PyPI, npm (WASM + native), crates.io
7. **Homebrew tap** is updated via a cross-repo dispatch event
8. **Version bump PR** is opened against `main`

The RC-tag model means `main` is always the development branch and never contains release artifacts. If a release fails partway through, a new RC tag (e.g., `v0.1.13-rc.2`) can retry without conflicting with the previous attempt.

## Security: trusted publishing

A goal of this project was to avoid storing any registry credentials as GitHub secrets. All three major registries (PyPI, npm, crates.io) support **OIDC trusted publishing**, which eliminates long-lived tokens entirely.

### How OIDC trusted publishing works

Instead of storing API tokens, each registry is configured to trust GitHub Actions as an identity provider:

1. The GitHub Actions workflow requests a short-lived OIDC token from GitHub's token endpoint
2. The token contains claims identifying the repository, workflow, and environment
3. The registry validates the token against a pre-configured trust policy (e.g., "accept publishes from `mmobeus/luadata` workflows in the `pypi` environment")
4. If valid, the registry accepts the publish

The token is short-lived (minutes) and scoped to the specific workflow run. There's nothing to rotate, nothing that can leak, and revoking access is a single toggle in the registry's settings.

### Per-registry setup

**PyPI** uses [`pypa/gh-action-pypi-publish`](https://github.com/pypa/gh-action-pypi-publish), which handles the OIDC exchange automatically. The PyPI project is configured with a "trusted publisher" that matches the GitHub repository and workflow file.

**npm** supports OIDC natively in recent versions. Setting `NODE_AUTH_TOKEN` to an empty string signals the npm CLI to use the OIDC flow instead of a static token. The npm package is configured with provenance settings that trust the GitHub repository.

**crates.io** uses [`rust-lang/crates-io-auth-action`](https://github.com/rust-lang/crates-io-auth-action), which exchanges the GitHub OIDC token for a short-lived crates.io publish token via `CARGO_REGISTRY_TOKEN`.

### The Homebrew exception

Homebrew doesn't have a registry API — updating a tap means pushing a commit to another GitHub repository. OIDC can't help here because the trust boundary is GitHub-to-GitHub, not GitHub-to-external-registry.

The solution is a **GitHub App** (`mmobeus-homebrew-updater`) installed only on the `homebrew-tap` repository with minimal permissions (Contents + Actions read/write). Two org-level secrets (`HOMEBREW_APP_ID`, `HOMEBREW_APP_PRIVATE_KEY`) allow the release workflow to mint a short-lived token scoped to the tap repo. This token sends a `repository_dispatch` event that triggers the formula update.

### Environment protection

Each publish job runs in a named [GitHub environment](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) (`pypi`, `npm`, `crates-io`). Environments can be configured to require manual approval before jobs run, adding a human gate before any package is published.
