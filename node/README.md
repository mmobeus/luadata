# @mmobeus/luadata

Native Node.js addon for parsing Lua data files and converting to JSON. Powered by Rust via N-API.

## Install

```
npm install @mmobeus/luadata
```

## Usage

```javascript
import { convertLuaToJson, convertLuaFileToJson } from "@mmobeus/luadata";

// Convert Lua data to a JSON string
const json = convertLuaToJson('playerName = "Thrall"');

// Parse into an object
const data = JSON.parse(convertLuaToJson(luaString));

// Convert a file
const json = convertLuaFileToJson("config.lua");

// With options
const json = convertLuaToJson(luaString, {
    emptyTable: "array",
    arrayMode: "sparse",
    arrayMaxGap: 10,
    stringTransform: { maxLen: 1024, mode: "truncate" },
});
```

Functions are synchronous and call the native Rust parser directly — no initialization step required. TypeScript type definitions are included.

## Options

All functions accept an optional options object with four groups:

- **Schema** (`schema`, `unknownFields`) — provide a JSON Schema string to guide type decisions, overriding heuristics
- **String transform** — limit string length during parsing (`truncate`, `empty`, `redact`, `replace`)
- **Array detection** — control how integer-keyed Lua tables map to JSON arrays (`sparse`, `index-only`, `none`)
- **Empty tables** — choose how empty Lua tables render in JSON (`null`, `omit`, `array`, `object`)

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
