# luadata (Go)

Go wrapper for luadata — parse Lua data files (such as World of Warcraft SavedVariables) and convert to JSON. Uses the Rust core library via a C shared library loaded at runtime.

## Install

```
go get github.com/mmobeus/luadata/go
```

## Usage

```go
import luadata "github.com/mmobeus/luadata/go"

// From a string
reader, err := luadata.TextToJSON("input", luaString)

// From a file
reader, err := luadata.FileToJSON("config.lua")

// From bytes
reader, err := luadata.ToJSON(luaBytes)

// From an io.Reader
reader, err := luadata.ReaderToJSON("input", r)

// With options
reader, err := luadata.TextToJSON("input", luaString,
    luadata.WithArrayMode("sparse", 10),
    luadata.WithEmptyTableMode("array"),
    luadata.WithStringTransform(1024, "truncate"),
)
```

All functions return an `io.Reader` containing JSON.

## Options

All functions accept functional options:

- **`WithSchema(schemaJSON)`** — provide a JSON Schema string to guide type decisions, overriding heuristics
- **`WithUnknownFieldMode(mode)`** — how to handle fields not in the schema (`ignore`, `include`, `fail`)
- **`WithStringTransform(maxLen, mode, [replacement])`** — limit string length during parsing (`truncate`, `empty`, `redact`, `replace`)
- **`WithArrayMode(mode, [maxGap])`** — control how integer-keyed Lua tables map to JSON arrays (`sparse`, `index-only`, `none`)
- **`WithEmptyTableMode(mode)`** — choose how empty Lua tables render in JSON (`null`, `omit`, `array`, `object`)

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
