# Agents

## Building

- **Always use Make targets** (`make build`, `make build-wasm`, etc.) instead of running `go build` directly.
- **Never** run bare `go build ./cmd/luadata` or `go build ./cmd/wasm` — these produce stray binaries (`luadata`, `wasm`) in the project root. Use the Make targets which output to `bin/cli/` and `bin/web/` respectively.
- **Never** run `GOOS=js GOARCH=wasm go build ./cmd/wasm` without `-o` — same problem, stray `wasm` binary in root.
- All build outputs go under `bin/` (`bin/cli/` for the CLI, `bin/web/` for wasm + static assets).
- The `web/` source directory contains source files (`index.html`, `luadata.js`, `app.js`). `make build-wasm` copies them to `bin/web/` along with the compiled WASM and runtime.
- To verify compilation only (no output binary), use `go build ./...` for native and `GOOS=js GOARCH=wasm go vet ./cmd/wasm` for the wasm target.
