# Rust Port Evaluation

We prototyped a full rewrite of luadata's parser and converter in Rust. This
document captures what we built, what we learned, and why we decided to keep Go
as the single implementation.

## What we built

A complete Rust port in a Cargo workspace with four crates:

- **luadata** (core library) — hand-written lexer, recursive descent parser, and
  JSON converter, mirroring the Go implementation. Used `serde_json` with the
  `preserve_order` feature for insertion-order-preserving JSON output.
- **luadata-cli** — clap-based CLI with `tojson` and `validate` subcommands and
  the same flags as the Go CLI.
- **luadata-python** — PyO3 native Python module exposing `lua_to_json()` and
  `lua_to_dict()` directly, replacing the ctypes + Go clib approach.
- **luadata-wasm** — wasm-bindgen module with the same `convert_lua_data_to_json`
  API as the Go WASM build.

All 29 Go test cases were ported and passing. The Rust CLI produced
byte-identical JSON output against every file in `testdata/`.

## What we learned

### WASM binary size: 165KB vs 3.2MB

The most dramatic difference. Go's WASM binary includes the entire Go runtime —
garbage collector, goroutine scheduler, and the `syscall/js` bridge. That's a
fixed ~2-3MB overhead regardless of program size. Rust compiles to WASM with
essentially zero runtime: no GC, no scheduler, just the application code and a
thin wasm-bindgen glue layer, and clocked in at 165KB.

### serde-wasm-bindgen: maps serialize as JS Map, not Object

When returning `serde_json::Value::Object` from Rust to JavaScript via
`serde_wasm_bindgen::to_value`, the default serializer produces a JS `Map`, not
a plain `Object`. Property access like `res.result` returns `undefined` on a
`Map` — you'd need `res.get("result")`.

The fix is to use the serializer with `serialize_maps_as_objects(true)`:

```rust
let serializer = serde_wasm_bindgen::Serializer::new().serialize_maps_as_objects(true);
value.serialize(&serializer)
```

This was the only behavioral bug in the prototype — everything else matched Go
output on the first try.

### PyO3 vs ctypes + clib

The current Python integration works by:
1. Building a Go shared library with CGO (`-buildmode=c-shared`)
2. Loading it from Python via ctypes
3. Marshalling strings across the FFI boundary
4. Parsing a JSON envelope (`{"result":...}` or `{"error":...}`)

The Rust/PyO3 approach compiled directly to a native Python extension module.
Python functions received and returned native Python types — no JSON envelope, no
manual memory management, no shared library discovery logic. The `maturin` build
tool handled wheel packaging automatically.

### Porting the parser was straightforward

The Go parser uses a hand-written lexer with character-by-character scanning and
a recursive descent parser — a pattern that maps naturally to Rust. Key
adaptations:

- Go's `rune`-based iteration became `Vec<char>` for O(1) indexing
- Go's `interface{}` (`any`) for values became Rust enums (`RawValue`, `RawKey`),
  giving compile-time type safety
- Go's functional options (`WithArrayDetection(mode)`) became a builder pattern
- Error handling used `Result<T, String>` throughout instead of Go's `(T, error)`

## The tradeoff matrix

We evaluated three approaches for the long-term architecture:

| | Go consumers | Python | WASM | Maintenance |
|---|---|---|---|---|
| **Go as source of truth** (current) | Pure Go import | ctypes -> Go clib | Go WASM (3.2MB) | 1 parser |
| **Rust as source of truth** | CGO required | PyO3 (native) | Rust WASM (165KB) | 1 parser |
| **Both implementations** | Pure Go import | PyO3 (native) | Rust WASM (165KB) | 2 parsers, shared test suite |

## Why we chose Go

The primary consumers of this library are Go projects. Making Rust the source of
truth would require those consumers to call a Rust shared library via CGO, which
has significant costs in the Go ecosystem:

- **Cross-compilation breaks.** Pure Go cross-compiles trivially
  (`GOOS=linux GOARCH=amd64 go build`). With CGO, you need a C toolchain —
  cross-compilers, target sysroots — for every target platform.
- **Static binaries require extra work.** Go's default static linking is a major
  deployment advantage (scratch Docker images, Lambda functions, single-binary
  CLI tools). CGO pulls in libc and dynamic linking by default.
- **CI/CD overhead.** Build containers would need a C compiler and the compiled
  Rust shared library available for every target architecture.
- **`CGO_ENABLED=0` is the default** for cross-compilation, so
  `GOOS=linux go build` on a Mac would silently fail to link the library.

These are not minor inconveniences — they fundamentally change Go's deployment
model for any project that depends on luadata.

NOTE: we will continue to look at this. I know that at least one of the use cases, 
the team is already familiar with building with CGO, and none of the above
are all that challenging for them. If we do go the rust route, we'll have a
golang wrapper folks can use, and we'll include a write up on how to build with it.

## The clib as universal FFI boundary

The current architecture already provides a clean solution for non-Go consumers.
The Go clib (`-buildmode=c-shared`) exports a standard C interface:

```c
extern char* LuaDataToJSON(char* input, char* options);
extern void LuaDataFree(char* ptr);
```

Any language with C FFI support can call this — Python does it today via ctypes,
and Rust could do it equally well. Calling a C shared library from Rust is a
first-class, well-supported operation:

```rust
extern "C" {
    fn LuaDataToJSON(input: *const c_char, options: *const c_char) -> *mut c_char;
    fn LuaDataFree(ptr: *mut c_char);
}
```

This is significantly simpler than the reverse direction (Go calling Rust via
CGO). Rust consumers face comparable complexity to what Python already handles.

## What Rust does better

To be clear about what we're leaving on the table:

- **WASM size**: 20x smaller (165KB vs 3.2MB). For a web-hosted converter, this
  is a meaningful difference in load time.
- **Python integration**: PyO3/maturin is a cleaner build story than
  CGO + ctypes + shared library discovery.
- **Potential performance**: Rust's lack of GC and zero-cost abstractions could
  yield faster parsing, though we didn't benchmark — the Go implementation is
  already fast enough for all current use cases.

These are real advantages. For a project where Python or WASM were the primary
consumers, Rust would likely be the right choice. But for this project, the Go
ecosystem cost outweighs these benefits.
