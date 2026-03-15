# luadata

A Go library for parsing Lua data files into JSON.

Parses `.lua` files containing variable assignments (e.g., `myVar = { ... }`) and converts them to structured JSON where each top-level variable becomes a JSON key. This is useful for working with game addon data files like World of Warcraft SavedVariables.

## Supported Lua syntax

- Strings (double-quoted, with escape sequences)
- Integers and floats (including scientific notation, negatives)
- Booleans (`true`/`false`) and `nil`
- Tables (`{}`) with bracket-keyed entries (`["key"]=val`, `[1]=val`)
- Implicit array indexing (values without keys get sequential numeric indices)
- Nested tables (arbitrary depth)
- Line comments (`--`) and block comments (`--[[ ]]`)

## Installation

```
go get github.com/mmobeus/luadata
```

## Usage

### Go library

```go
import "github.com/mmobeus/luadata"

// Parse from file
data, err := luadata.ParseFile("saved.lua")

// Parse from string
data, err := luadata.ParseText("input", luaString)

// Quick convert to JSON
jsonBytes, err := luadata.ToJSON(luaBytes)

// Access typed values
name := data.GetString("playerName")
level := data.GetInt("playerLevel")
settings := data.GetTable("userSettings")

// Safe access with ok pattern
if hp, ok := data.MaybeGetInt("health"); ok {
    fmt.Println(hp)
}
```

Available accessors on `KeyValuePairs`:

| Method                                   | Return type      |
|------------------------------------------|------------------|
| `GetString(key)` / `MaybeGetString(key)` | `string`         |
| `GetInt(key)` / `MaybeGetInt(key)`       | `int64`          |
| `GetFloat(key)` / `MaybeGetFloat(key)`   | `float64`        |
| `GetBool(key)` / `MaybeGetBool(key)`     | `bool`           |
| `GetTable(key)` / `MaybeGetTable(key)`   | `KeyValuePairs`  |
| `Len()`                                  | `int`            |
| `Pairs()`                                | `[]KeyValuePair` |

### CLI

```bash
# Build
make build

# Convert a file to JSON
bin/cli/luadata tojson config.lua

# Read from stdin
cat config.lua | bin/cli/luadata tojson -
```

### WebAssembly

The converter is also available as a WebAssembly module. Build it with:

```bash
GOOS=js GOARCH=wasm go build -o luadata.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
```

Or use the Makefile:

```bash
make build-wasm  # outputs to bin/web/
```

Load and call it from JavaScript:

```html
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("luadata.wasm"), go.importObject).then((result) => {
    go.run(result.instance);

    // convertLuaDataToJson is now available globally
    const output = convertLuaDataToJson('playerName = "Thrall"');
    if (output.error) {
        console.error(output.error);
    } else {
        console.log(JSON.parse(output.result));
    }
});
</script>
```

The function `convertLuaDataToJson(input)` takes a Lua data string and returns `{ result: string }` on success or `{ error: string }` on failure.

> **Note:** `wasm_exec.js` must come from the same Go version used to compile the `.wasm` file.

#### Web UI

A ready-made web interface is included:

```bash
make serve
# Opens at http://localhost:8080
```

Paste Lua data on the left, get JSON on the right.

## Raw values

In addition to variable assignments, luadata can parse files containing a single raw Lua value (e.g., just a table, string, number, boolean, or `nil`). When a raw value is detected, the result contains a single top-level key `@root` (which cannot collide with valid Lua identifiers).

```go
data, err := luadata.ParseText("input", `{["a"]=1,["b"]=2}`)
// data has one entry with key "@root" containing the table

data, err = luadata.ParseText("input", `"hello"`)
// data.GetString("@root") == "hello"
```

CLI example:

```bash
echo '{"a","b","c"}' | bin/cli/luadata tojson -
# {"@root":{"1":"a","2":"b","3":"c"}}
```

Raw value mode is exclusive — only one value per input is allowed, and trailing content after the value produces an error.

## Example

Input (`saved.lua`):

```lua
-- Character data
playerName = "Thrall"
playerLevel = 60
settings = {
    ["showHelm"] = true,
    ["ui"] = {
        ["scale"] = 1.25,
        ["panels"] = {"map", "inventory", "chat"},
    },
}
```

Output:

```json
{
  "playerLevel": 60,
  "playerName": "Thrall",
  "settings": {
    "showHelm": true,
    "ui": {
      "panels": {
        "1": "map",
        "2": "inventory",
        "3": "chat"
      },
      "scale": 1.25
    }
  }
}
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

| Command        | Description                       |
|----------------|-----------------------------------|
| `make test`    | Run tests                         |
| `make lint`    | Run golangci-lint (includes fmt)  |
| `make fmt`     | Format code with gofumpt          |
| `make check`   | Run build + test + lint           |
| `make setup`   | Install dev tools                 |

### Releasing

To push a new patch release (the most common case), run:

```
make release
```

This finds the latest semver tag, bumps the patch version (e.g. `v0.1.0` → `v0.1.1`), and asks for confirmation before creating the git tag, pushing it, and creating a GitHub release.

For other version bumps, set `BUMP`:

| Command                    | Description                          |
|----------------------------|--------------------------------------|
| `make release`             | Patch bump (default)                 |
| `make release BUMP=minor`  | Minor bump, resets patch to 0        |
| `make release BUMP=major`  | Major bump, resets minor and patch   |
| `make release BUMP=manual` | Prompt for an exact version string   |

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) for creating GitHub releases.

## Building

| Command           | Description                            |
|-------------------|----------------------------------------|
| `make build`      | Build CLI binary to `bin/cli/luadata`  |
| `make build-wasm` | Build WebAssembly module to `bin/web/` |
| `make serve`      | Build WASM and serve web UI on `:8080` |
| `make clean`      | Remove `bin/` directory                |

## Testing

```
go test ./...
```

## Zero dependencies

Uses only the Go standard library.

## License

[MIT](LICENSE)
