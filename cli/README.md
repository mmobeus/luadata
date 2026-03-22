# luadata (CLI)

Command-line tool for parsing Lua data files and converting to JSON. Useful for working with game addon data like World of Warcraft SavedVariables.

## Install

### Homebrew

```
brew install mmobeus/tap/luadata
```

### Cargo

```
cargo install luadata_cli
```

## Usage

```bash
# Convert a file
luadata tojson config.lua

# Read from stdin
cat config.lua | luadata tojson -

# Validate without converting
luadata validate config.lua

# With options
luadata tojson config.lua --empty-table array --array-mode sparse --array-max-gap 10
```

## Options

| Flag | Description |
|---|---|
| `--schema <json-or-path>` | JSON Schema (inline JSON or file path) to guide conversion |
| `--unknown-fields <mode>` | Unknown field handling: `ignore`, `include`, `fail` |
| `--empty-table <mode>` | How to render empty tables: `null`, `omit`, `array`, `object` |
| `--array-mode <mode>` | Array detection: `sparse`, `index-only`, `none` |
| `--array-max-gap <n>` | Max gap for sparse array detection (default: 20) |
| `--string-max-len <n>` | Max string length before transform is applied |
| `--string-mode <mode>` | String transform: `truncate`, `empty`, `redact`, `replace` |
| `--string-replacement <s>` | Replacement string (with `--string-mode replace`) |

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
