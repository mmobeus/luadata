# luadata

A Lua data parser with Rust, Go, Python, Node.js, CLI, and WebAssembly interfaces. Useful for working with game addon data files like World of Warcraft SavedVariables.

**[Luadata by Example](https://mmobeus.github.io/luadata/docs/)** — A guided tour of all features with interactive examples.

**[Live Converter](https://mmobeus.github.io/luadata/)** — Try it in your browser. Paste Lua data and get JSON instantly.

### Why so many bindings?

luadata started as a Go utility. The additional language bindings exist because this project doubles as a learning exercise in sharing a core Rust library across multiple languages and ecosystems. See [GUIDE.md](GUIDE.md) for the full story.

## Usage

### Rust — [`luadata`](https://crates.io/crates/luadata)

```
cargo add luadata
```

```rust
let json = luadata::text_to_json("input", r#"playerName = "Thrall""#, ParseConfig::new())?;
```

See [CRATE_README.md](CRATE_README.md) for full docs.

### Go — [`github.com/mmobeus/luadata/go`](go/)

```
go get github.com/mmobeus/luadata/go
```

```go
reader, err := luadata.TextToJSON("input", luaString)
```

All functions return an `io.Reader` containing JSON. See [go/README.md](go/README.md) for full docs.

### Python — [`mmobeus-luadata`](https://pypi.org/project/mmobeus-luadata/)

```
pip install mmobeus-luadata
```

```python
data = lua_to_dict('playerName = "Thrall"')
```

See [python/README.md](python/README.md) for full docs.

### Node.js — [`@mmobeus/luadata`](https://www.npmjs.com/package/@mmobeus/luadata)

```
npm install @mmobeus/luadata
```

```javascript
const json = convertLuaToJson('playerName = "Thrall"');
```

Synchronous, native Rust via N-API. See [node/README.md](node/README.md) for full docs.

### CLI — [`luadata_cli`](https://crates.io/crates/luadata_cli)

```
brew install mmobeus/tap/luadata
```

```bash
luadata tojson config.lua
```

See [cli/README.md](cli/README.md) for full docs.

### WebAssembly — [`@mmobeus/luadata-wasm`](https://www.npmjs.com/package/@mmobeus/luadata-wasm)

For browser projects using a bundler (webpack, vite, etc.):

```
npm install @mmobeus/luadata-wasm
```

```typescript
import { init, convert } from "@mmobeus/luadata-wasm";
await init();
const json = convert('playerName = "Thrall"');
```

See [node-wasm/README.md](node-wasm/README.md) for full docs. For Node.js without a bundler, use `@mmobeus/luadata` instead.

For direct browser usage without a bundler:

```bash
make build-wasm  # outputs to bin/web/
make serve       # opens at http://localhost:8080
```

## Lua data format

The library parses Lua files containing top-level variable assignments. This is a common data persistence technique used by Lua systems (including World of Warcraft addon data). Each assignment is a variable name followed by `=` and a value:

```lua
playerName = "Thrall"
playerLevel = 60
guildRoster = {
    ["Thrall"] = {
        ["level"] = 60,
        ["class"] = "Shaman",
    },
}
```

This is a valid Lua file, with a list of assignments to inline data values. These are parsed into a map-like structure, where the keys are the variable names, and the values are JSON equivalents of the Lua values.

### Raw values

In addition to variable assignments, luadata can parse a single raw Lua value (a table, string, number, boolean, or `nil`). When a raw value is detected, the output contains a single key `@root` with the parsed value:

```bash
echo '{"a","b","c"}' | luadata tojson -
# {"@root":["a","b","c"]}
```

```python
lua_to_dict('{["a"] = 1, ["b"] = 2}')
# {'@root': {'a': 1, 'b': 2}}
```

### Binary strings

Lua strings are raw byte sequences with no inherent encoding. Game addons like
[Questie](https://github.com/Questie/Questie) use this to store compact binary
data (packed coordinates, pointer maps, serialized databases) directly inside
Lua string values. When the game client writes these to disk, the bytes are
written verbatim between quotes.

luadata handles this with a per-string heuristic: if a string's bytes are valid
UTF-8, it decodes them as UTF-8 (so accented player names like "Fröst" render
correctly). If the bytes contain any invalid UTF-8 sequences, each byte is
mapped to its Latin-1 code point, preserving every byte losslessly.

A consumer can recover the original bytes from a binary string value:

```python
raw_bytes = bytes(ord(c) for c in json_value)
```

```javascript
const rawBytes = [...jsonValue].map(c => c.codePointAt(0));
```

```go
rawBytes := []byte(jsonValue)
```

## Options

All parse and convert functions accept options controlling three behaviors: string transform, array detection, and empty table rendering. The defaults are the same across all languages.

### String transform

Limit string length during parsing. When a string exceeds the max length, the transform is applied — the parser treats the result as if the transformed value was the original.

| Mode | Behavior |
|---|---|
| `truncate` | Truncate to max length |
| `empty` | Replace with `""` |
| `redact` | Replace with `"[redacted]"` |
| `replace` | Replace with a custom string |

Strings at or under the max length are not modified.

<details>
<summary>Syntax by language</summary>

**Rust:**
```rust
config.string_transform = Some(StringTransform {
    max_len: 1024,
    mode: StringTransformMode::Truncate,
    replacement: String::new(),
});
```

**Go:**
```go
luadata.WithStringTransform(1024, "truncate")
luadata.WithStringTransform(2048, "replace", "[removed]")
```

**Python:**
```python
lua_to_json(text, string_max_len=1024, string_mode="truncate")
lua_to_json(text, string_max_len=2048, string_mode="replace", string_replacement="[removed]")
```

**CLI:**
```bash
luadata tojson file.lua --string-max-len 1024 --string-mode truncate
luadata tojson file.lua --string-max-len 2048 --string-mode replace --string-replacement "[removed]"
```

**Node.js:**
```javascript
convertLuaToJson(text, { stringTransform: { maxLen: 1024, mode: "truncate" } })
convertLuaToJson(text, { stringTransform: { maxLen: 2048, mode: "replace", replacement: "[removed]" } })
```

**WASM:**
```javascript
convert(text, { stringTransform: { maxLen: 1024, mode: "truncate" } })
```

</details>

### Array detection

Lua tables with integer keys are conceptually arrays, but Lua has no distinct array type — arrays are just tables with sequential integer keys. This creates an ambiguity when converting to JSON, where arrays and objects are distinct types.

In Lua (and WoW SavedVariables in particular), array data can appear in two forms:

- **Implicit index syntax**: `{"apple", "banana", "cherry"}` — elements have no explicit keys
- **Explicit integer key syntax**: `{[1] = "apple", [2] = "banana", [3] = "cherry"}` — each element has a `[n] =` prefix

WoW addons may switch between these forms over time. An array initially saved with implicit syntax may later be re-saved with explicit `[n]=` keys after entries are added or removed. Sparse arrays (with gaps) like `{[1] = "apple", [3] = "cherry"}` are also common when elements are deleted.

By default, both implicit index tables and tables with explicit integer keys render as JSON arrays, as long as the gap between consecutive keys does not exceed 20 (`sparse` mode with max gap 20). Missing indices are filled with `null`. This produces the most natural JSON for array-like data.

| Mode | `{[1]="a",[2]="b"}` | `{[1]="a",[3]="c"}` | `{"a","b"}` |
|---|---|---|---|
| `sparse` (default, max gap 20) | `["a","b"]` | `["a",null,"c"]` | `["a","b"]` |
| `sparse` (max gap 0) | `["a","b"]` | `{"1":"a","3":"c"}` | `["a","b"]` |
| `index-only` | `{"1":"a","2":"b"}` | `{"1":"a","3":"c"}` | `["a","b"]` |
| `none` | `{"1":"a","2":"b"}` | `{"1":"a","3":"c"}` | `{"1":"a","2":"b"}` |

Gaps are measured from index 0, so Lua's 1-based arrays (starting at `[1]`) have a gap of 0 from the start. A table starting at `[2]` has a leading gap of 1.

<details>
<summary>Syntax by language</summary>

**Rust:**
```rust
config.array_mode = Some(ArrayMode::Sparse { max_gap: 0 });
config.array_mode = Some(ArrayMode::IndexOnly);
config.array_mode = Some(ArrayMode::None);
```

**Go:**
```go
luadata.WithArrayMode("sparse", 0)    // contiguous only
luadata.WithArrayMode("index-only")
luadata.WithArrayMode("none")
```

**Python:**
```python
lua_to_json(text, array_mode="sparse", array_max_gap=0)
lua_to_json(text, array_mode="index-only")
lua_to_json(text, array_mode="none")
```

**CLI:**
```bash
luadata tojson file.lua --array-mode sparse --array-max-gap 0
luadata tojson file.lua --array-mode index-only
luadata tojson file.lua --array-mode none
```

**Node.js:**
```javascript
convertLuaToJson(text, { arrayMode: "sparse", arrayMaxGap: 0 })
convertLuaToJson(text, { arrayMode: "index-only" })
convertLuaToJson(text, { arrayMode: "none" })
```

**WASM:**
```javascript
convert(text, { arrayMode: "sparse", arrayMaxGap: 0 })
convert(text, { arrayMode: "index-only" })
convert(text, { arrayMode: "none" })
```

</details>

### Empty tables

Lua has no distinction between an empty array and an empty object — both are simply `{}`. This creates an ambiguity when converting to JSON, where `[]` and `{}` have different meanings. By default, empty tables render as `null`, which avoids making an arbitrary choice between the two JSON types while still making the key visible in the output (unlike omitting it, which could look like a bug).

| Mode | `foo={}` |
|---|---|
| `null` (default) | `{"foo":null}` |
| `omit` | `{}` (key omitted) |
| `array` | `{"foo":[]}` |
| `object` | `{"foo":{}}` |

Both `{}` and whitespace-only tables (like `{\n}`) are treated the same way under all modes. The mode applies everywhere empty tables appear — top-level values, nested table values, and elements inside arrays.

<details>
<summary>Syntax by language</summary>

**Rust:**
```rust
config.empty_table_mode = EmptyTableMode::Array;
```

**Go:**
```go
luadata.WithEmptyTableMode("array")
```

**Python:**
```python
lua_to_json(text, empty_table="array")
```

**CLI:**
```bash
luadata tojson file.lua --empty-table array
```

**Node.js:**
```javascript
convertLuaToJson(text, { emptyTable: "array" })
```

**WASM:**
```javascript
convert(text, { emptyTable: "array" })
```

</details>

## Development

### Prerequisites

- [Rust](https://rustup.rs/) (stable)
- [Go](https://go.dev/) 1.26+
- [gofumpt](https://github.com/mvdan/gofumpt) v0.9.0 (installed by `make setup`)

Optional (for Python development):
- [uv](https://docs.astral.sh/uv/) — Python package manager
- [maturin](https://www.maturin.rs/) — PyO3 build tool

### Setup

```
make setup
```

This installs `gofumpt` and the Rust `clippy`/`rustfmt` components.

### Running checks locally

```
make check
```

Runs build, Rust tests, lint, format check, and testdata validation.

| Command               | Description                               |
|-----------------------|-------------------------------------------|
| `make test`           | Run Rust + Go tests                       |
| `make test-rust`      | Run Rust tests only                       |
| `make test-go`        | Build clib and run Go tests               |
| `make test-python`    | Build Python module and run pytest        |
| `make test-node`      | Build Node.js addon and run tests         |
| `make lint`           | Run clippy                                |
| `make fmt`            | Format Rust and Go code                   |
| `make fmt-check`      | Check formatting without modifying        |
| `make check`          | Build + test + lint + fmt-check + validate|

### Building

| Command               | Description                               |
|-----------------------|-------------------------------------------|
| `make build`          | Build CLI binary to `bin/cli/luadata`     |
| `make build-clib`     | Build C shared library to `bin/clib/`     |
| `make build-clib-go`  | Build clib and copy to Go embed location  |
| `make build-node`     | Build Node.js native addon                |
| `make build-wasm`     | Build WebAssembly module to `bin/web/`    |
| `make build-site`     | Build WASM + docs site                    |
| `make serve`          | Build site and serve on `:8080`           |
| `make clean`          | Remove `bin/` and `target/` directories   |

### Releasing

To push a new patch release (the most common case), run:

```
make release
```

This tags an RC on `main`, which triggers CI to cross-compile the Rust shared library for all platforms, run tests, create the release branch with embedded libraries, and publish the GitHub Release.

For other version bumps, set `BUMP`:

| Command                     | Description                        |
|-----------------------------|------------------------------------|
| `make release`              | Patch bump (default)               |
| `make release BUMP=minor`   | Minor bump, resets patch to 0      |
| `make release BUMP=major`   | Major bump, resets minor and patch |
| `make release BUMP=manual`  | Prompt for an exact version string |

See [ARCHITECTURE.md](ARCHITECTURE.md) for details on the release workflow and Go consumer model.
