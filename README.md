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

## CLI

```bash
make build

bin/cli/luadata tojson config.lua

cat config.lua | bin/cli/luadata tojson -
```

## API reference

Parse into `KeyValuePairs`:

| Function                     | Description                        |
|------------------------------|------------------------------------|
| `ParseFile(path)`            | Parse a `.lua` file from disk      |
| `ParseText(name, text)`      | Parse Lua data from a string       |
| `ParseReader(name, reader)`  | Parse Lua data from an `io.Reader` |

Convert to JSON (`io.Reader`):

| Function                     | Description                     |
|------------------------------|---------------------------------|
| `FileToJSON(path)`           | File to JSON                    |
| `TextToJSON(name, text)`     | String to JSON                  |
| `ReaderToJSON(name, reader)` | `io.Reader` to JSON             |
| `ToJSON(luaBytes)`           | `[]byte` to JSON                |

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
