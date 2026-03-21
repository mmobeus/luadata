# @mmobeus/luadata-wasm

WebAssembly build of luadata for parsing Lua data files and converting to JSON. Designed for browser projects using a bundler (webpack, vite, etc.).

For Node.js usage without a bundler, use [`@mmobeus/luadata`](https://www.npmjs.com/package/@mmobeus/luadata) instead.

## Install

```
npm install @mmobeus/luadata-wasm
```

## Usage

```typescript
import { init, convert } from "@mmobeus/luadata-wasm";

// Initialize the WASM module (call once before convert)
await init();

// Convert Lua data to a JSON string
const json = convert('playerName = "Thrall"');

// Parse into an object
const data = JSON.parse(convert(luaString));

// With options
const json = convert(luaString, {
    emptyTable: "array",
    arrayMode: "sparse",
    arrayMaxGap: 10,
    stringTransform: { maxLen: 1024, mode: "truncate" },
});
```

`init()` must be called once before `convert()` — it loads the WASM module. TypeScript type definitions are included.

## Options

The `convert` function accepts an optional options object with three groups:

- **String transform** — limit string length during parsing (`truncate`, `empty`, `redact`, `replace`)
- **Array detection** — control how integer-keyed Lua tables map to JSON arrays (`sparse`, `index-only`, `none`)
- **Empty tables** — choose how empty Lua tables render in JSON (`null`, `omit`, `array`, `object`)

See the [full options documentation](https://github.com/mmobeus/luadata#options) for details and examples.

## Links

- [Luadata by Example](https://mmobeus.github.io/luadata/docs/) — guided tour with interactive examples
- [Live Converter](https://mmobeus.github.io/luadata/) — try it in your browser
- [GitHub](https://github.com/mmobeus/luadata)
