# Agents

## Building

- Use `make build` and `make build-wasm` (see Makefile) instead of running `go build` directly.
- **Never** run `GOOS=js GOARCH=wasm go build ./cmd/wasm` without `-o` — it produces a stray `wasm` binary in the current directory. Use `make build-wasm` which outputs to `bin/web/luadata.wasm`.
- All build outputs go under `bin/` (`bin/cli/` for the CLI, `bin/web/` for wasm + static assets). The `web/` source directory contains only `index.html`; generated files are never written there.
- To verify compilation only (no output binary), use `go build ./...` for native and `GOOS=js GOARCH=wasm go vet ./cmd/wasm` for the wasm target.
