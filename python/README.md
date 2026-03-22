# mmobeus-luadata

Python bindings for luadata — parse Lua data files (such as World of Warcraft SavedVariables) and convert to JSON or Python dicts. Powered by Rust via PyO3.

## Install

```
pip install mmobeus-luadata
```

## Usage

```python
from luadata import lua_to_json, lua_to_dict

# Get JSON string
json_str = lua_to_json('playerName = "Thrall"')

# Get Python dict
data = lua_to_dict('playerName = "Thrall"')

# With options
data = lua_to_dict(lua_string,
    array_mode="sparse",
    array_max_gap=10,
    empty_table="array",
    string_max_len=1024,
    string_mode="truncate",
)
```

## Options

Both functions accept named arguments for four option groups:

- **Schema** (`schema`, `unknown_fields`) — provide a JSON Schema string to guide type decisions, overriding heuristics
- **String transform** (`string_max_len`, `string_mode`, `string_replacement`) — limit string length during parsing (`truncate`, `empty`, `redact`, `replace`)
- **Array detection** (`array_mode`, `array_max_gap`) — control how integer-keyed Lua tables map to JSON arrays (`sparse`, `index-only`, `none`)
- **Empty tables** (`empty_table`) — choose how empty Lua tables render in JSON (`null`, `omit`, `array`, `object`)

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
