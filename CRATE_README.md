# luadata

A Lua data parser that converts Lua data files (such as World of Warcraft SavedVariables) to JSON.

## Install

```
cargo add luadata
```

## Usage

```rust
use luadata::{text_to_json, file_to_json, ParseConfig};

// From a string
let json = text_to_json("input", r#"playerName = "Thrall""#, ParseConfig::new())?;

// From a file
let json = file_to_json("config.lua", ParseConfig::new())?;

// With options
let mut config = ParseConfig::new();
config.array_mode = Some(luadata::options::ArrayMode::IndexOnly);
config.empty_table_mode = luadata::options::EmptyTableMode::Array;
let json = text_to_json("input", lua_string, config)?;
```

## Options

All parse functions accept a `ParseConfig` with four option groups:

- **Schema** — provide a JSON Schema to guide type decisions, overriding heuristics (`schema`, `unknown_field_mode`)
- **String transform** — limit string length during parsing (`truncate`, `empty`, `redact`, `replace`)
- **Array detection** — control how integer-keyed Lua tables map to JSON arrays (`sparse`, `index-only`, `none`)
- **Empty tables** — choose how empty Lua tables render in JSON (`null`, `omit`, `array`, `object`)

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
