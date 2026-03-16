# luadata

A Go library that parses Lua data files into JSON. Useful for working with game addon data files like World of Warcraft SavedVariables.

## Install

```
go get github.com/mmobeus/luadata
```

## Examples

### Lua to JSON

```go
input := []byte(`
playerName = "Thrall"
playerLevel = 60
settings = {
    ["showHelm"] = true,
    ["ui"] = {
        ["scale"] = 1.25,
        ["panels"] = {"map", "inventory", "chat"},
    },
}
`)

jsonReader, err := luadata.ToJSON(input)
```

Output:

```json
{
  "playerName": "Thrall",
  "playerLevel": 60,
  "settings": {
    "showHelm": true,
    "ui": {
      "scale": 1.25,
      "panels": ["map", "inventory", "chat"]
    }
  }
}
```

### Typed access

```go
data, err := luadata.ParseFile("saved.lua")

name := data.GetString("playerName")
level := data.GetInt("playerLevel")
settings := data.GetTable("settings")

if hp, ok := data.MaybeGetInt("health"); ok {
    fmt.Println(hp)
}
```


## Saved variable format

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

This is a valid Lua file, with a list of assignments to inline data values. These are parsed into a map-like structure, where the keys are the variable names, and the values are json equivalents of the Lua values.

## Raw values

In addition to variable assignments, luadata can parse a single raw Lua value (a table, string, number, boolean, or `nil`). When a raw value is detected, the resulting map-like structure contains a single key `@root`, with the parsed lua value as the value.

```go
data, err := luadata.ParseText("input", `{["a"]=1,["b"]=2}`)
// data has one entry with key "@root" containing the table
```

```bash
echo '{"a","b","c"}' | bin/cli/luadata tojson -
# {"@root":["a","b","c"]}
```

## Options

All parse and convert functions accept functional options.

### String transform

Use `WithStringTransform` to limit string length during parsing. This is indended for use cases where the source data has some values that are larger than the caller expects to manage, and do NOT want the large values to (for example) be rendered in the JSON output.

When a string exceeds `MaxLen`, the transform is applied to both `Source` and `Raw` — the parser treats the result as if the transformed value was the original. The `Transformed` flag on `Value` is set to `true` so callers can detect this if needed.

```go
data, err := luadata.ParseText("input", luaString,
    luadata.WithStringTransform(luadata.StringTransform{
        MaxLen: 1024,
        Mode:   luadata.StringTransformTruncate,
    }),
)
```

Available modes:

| Mode                       | Behavior                              |
|----------------------------|---------------------------------------|
| `StringTransformTruncate`  | Truncate to `MaxLen` bytes            |
| `StringTransformEmpty`     | Replace with `""`                     |
| `StringTransformRedact`    | Replace with `"[redacted]"`           |
| `StringTransformReplace`   | Replace with custom `Replacement` string |

Strings at or under `MaxLen` are not modified.

```go
// Replace long strings with a custom message
jsonReader, err := luadata.ToJSON(input,
    luadata.WithStringTransform(luadata.StringTransform{
        MaxLen:      2048,
        Mode:        luadata.StringTransformReplace,
        Replacement: "[removed]",
    }),
)
```

### Array detection

Lua tables with integer keys are conceptually arrays, but Lua has no distinct array type — arrays are just tables with sequential integer keys. This creates an ambiguity when converting to JSON, where arrays and objects are distinct types.

In Lua (and WoW SavedVariables in particular), array data can appear in two forms:

- **Implicit index syntax**: `{"apple", "banana", "cherry"}` — elements have no explicit keys
- **Explicit integer key syntax**: `{[1] = "apple", [2] = "banana", [3] = "cherry"}` — each element has a `[n] =` prefix

WoW addons may switch between these forms over time. An array initially saved with implicit syntax may later be re-saved with explicit `[n]=` keys after entries are added or removed. Sparse arrays (with gaps) like `{[1] = "apple", [3] = "cherry"}` are also common when elements are deleted.

By default, both implicit index tables and tables with explicit integer keys render as JSON arrays, as long as the gap between consecutive keys does not exceed 20 (`ArrayModeSparse{MaxGap: 20}`). Missing indices are filled with `null`. This produces the most natural JSON for array-like data.

`WithArrayDetection` controls this behavior using one of three modes:

```go
// ArrayModeSparse (default): treat Int-key tables as arrays within a gap threshold.
// The default is ArrayModeSparse{MaxGap: 20} when no option is specified.
data, err := luadata.ParseText("input", luaString,
    luadata.WithArrayDetection(luadata.ArrayModeSparse{MaxGap: 0}), // contiguous only
)

// ArrayModeIndexOnly: only implicit index tables ({"a","b"}) render as arrays.
// Explicit integer keys always produce objects.
data, err := luadata.ParseText("input", luaString,
    luadata.WithArrayDetection(luadata.ArrayModeIndexOnly{}),
)

// ArrayModeNone: no array rendering at all. Every table becomes a JSON object,
// including implicit index tables.
jsonReader, err := luadata.ToJSON(input,
    luadata.WithArrayDetection(luadata.ArrayModeNone{}),
)
```

| Mode | `{[1]="a",[2]="b"}` | `{[1]="a",[3]="c"}` | `{"a","b"}` |
|---|---|---|---|
| `ArrayModeSparse{MaxGap: 20}` (default) | `["a","b"]` | `["a",null,"c"]` | `["a","b"]` |
| `ArrayModeSparse{MaxGap: 0}` | `["a","b"]` | `{"1":"a","3":"c"}` | `["a","b"]` |
| `ArrayModeIndexOnly{}` | `{"1":"a","2":"b"}` | `{"1":"a","3":"c"}` | `["a","b"]` |
| `ArrayModeNone{}` | `{"1":"a","2":"b"}` | `{"1":"a","3":"c"}` | `{"1":"a","2":"b"}` |

Gaps are measured from index 0, so Lua's 1-based arrays (starting at `[1]`) have a gap of 0 from the start. A table starting at `[2]` has a leading gap of 1.

## CLI

```bash
make build

bin/cli/luadata tojson config.lua

cat config.lua | bin/cli/luadata tojson -
```

## API reference

Parse into `KeyValuePairs`:

| Function                              | Description                        |
|---------------------------------------|------------------------------------|
| `ParseFile(path, ...Option)`          | Parse a `.lua` file from disk      |
| `ParseText(name, text, ...Option)`    | Parse Lua data from a string       |
| `ParseReader(name, reader, ...Option)`| Parse Lua data from an `io.Reader` |

Convert to JSON (`io.Reader`):

| Function                              | Description                     |
|---------------------------------------|---------------------------------|
| `FileToJSON(path, ...Option)`         | File to JSON                    |
| `TextToJSON(name, text, ...Option)`   | String to JSON                  |
| `ReaderToJSON(name, reader, ...Option)`| `io.Reader` to JSON            |
| `ToJSON(luaBytes, ...Option)`         | `[]byte` to JSON                |

Accessors on `KeyValuePairs`:

| Method                                     | Return type      |
|--------------------------------------------|------------------|
| `GetString(key)` / `MaybeGetString(key)`   | `string`         |
| `GetInt(key)` / `MaybeGetInt(key)`         | `int64`          |
| `GetFloat(key)` / `MaybeGetFloat(key)`     | `float64`        |
| `GetBool(key)` / `MaybeGetBool(key)`       | `bool`           |
| `GetTable(key)` / `MaybeGetTable(key)`     | `KeyValuePairs`  |
| `Len()`                                    | `int`            |
| `Pairs()`                                  | `[]KeyValuePair` |

## WebAssembly

The converter is available as a WebAssembly module:

```bash
make build-wasm  # outputs to bin/web/
```

Load from JavaScript:

```html
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("luadata.wasm"), go.importObject).then((result) => {
    go.run(result.instance);

    const output = convertLuaDataToJson('playerName = "Thrall"');
    if (output.error) {
        console.error(output.error);
    } else {
        console.log(JSON.parse(output.result));
    }
});
</script>
```

> **Note:** `wasm_exec.js` must come from the same Go version used to compile the `.wasm` file.

A ready-made web interface is also included:

```bash
make serve
# Opens at http://localhost:8080
```

## Development

### Setup

Install development tools (requires Go):

```
make setup
```

This installs:
- [golangci-lint](https://golangci-lint.run/) v2.11.3 — linter aggregator
- [gofumpt](https://github.com/mvdan/gofumpt) v0.9.0 — stricter gofmt

### Running checks locally

```
make check
```

Runs build, tests, and lint — the same checks as the GitHub Actions workflow.

| Command        | Description                      |
|----------------|----------------------------------|
| `make test`    | Run tests                        |
| `make lint`    | Run golangci-lint (includes fmt) |
| `make fmt`     | Format code with gofumpt         |
| `make check`   | Run build + test + lint          |
| `make setup`   | Install dev tools                |

### Building

| Command            | Description                            |
|--------------------|----------------------------------------|
| `make build`       | Build CLI binary to `bin/cli/luadata`  |
| `make build-wasm`  | Build WebAssembly module to `bin/web/` |
| `make serve`       | Build WASM and serve web UI on `:8080` |
| `make clean`       | Remove `bin/` directory                |

### Releasing

To push a new patch release (the most common case), run:

```
make release
```

This finds the latest semver tag, bumps the patch version (e.g. `v0.1.0` → `v0.1.1`), and asks for confirmation before creating the git tag, pushing it, and creating a GitHub release.

For other version bumps, set `BUMP`:

| Command                     | Description                        |
|-----------------------------|------------------------------------|
| `make release`              | Patch bump (default)               |
| `make release BUMP=minor`   | Minor bump, resets patch to 0      |
| `make release BUMP=major`   | Major bump, resets minor and patch |
| `make release BUMP=manual`  | Prompt for an exact version string |

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) for creating GitHub releases.
