# Agents

## Building

- **Always use Make targets** (`make build`, `make build-clib`, etc.) instead of running `cargo build` or `go build` directly.
- All build outputs go under `bin/` (`bin/cli/` for the CLI, `bin/clib/` for the shared library, `bin/web/` for WASM + static assets).
- The Rust workspace root is `Cargo.toml`. Crates: `luadata` (core lib in `src/`), `clib/`, `cli/`, `python/`, `wasm/`.
- The Go wrapper lives in `go/` with its own `go.mod` (`github.com/mmobeus/luadata/go`).
- For local Go development, run `make build-clib` first, then set `LUADATA_LIB_PATH` to `bin/clib/libluadata.dylib` (or `.so`).
- `go/lib/` is gitignored — shared libraries are embedded only at release time via CI.
- To verify compilation only: `cargo check --workspace` for Rust, `go build ./...` (from `go/` dir) for Go.

## Testing

- `make test-rust` — runs Rust core library tests
- `make test-go` — builds cdylib, then runs Go wrapper tests with it
- `make test` — runs both
