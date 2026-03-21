// @mmobeus/luadata-wasm — Parse Lua data files and convert to JSON.
//
// Usage:
//   import { init, convert } from "@mmobeus/luadata-wasm";
//   await init();
//   const json = convert('playerName = "Thrall"');

import wasmInit, { convertLuaDataToJson } from "./wasm/luadata_wasm.js";

let initialized = false;

/**
 * Initialize the WASM module. Must be called once before using convert().
 */
export async function init() {
    if (initialized) return;
    await wasmInit();
    initialized = true;
}

/**
 * Convert a Lua data string to a JSON string.
 *
 * @param {string} input - Lua data source text
 * @param {ConvertOptions} [opts] - Conversion options
 * @returns {string} JSON string
 * @throws {Error} If the WASM module is not initialized or parsing fails
 */
export function convert(input, opts) {
    if (!initialized) throw new Error("luadata WASM not initialized — call init() first");

    const hasOpts = opts && Object.keys(opts).length > 0;
    const res = hasOpts
        ? convertLuaDataToJson(input, opts)
        : convertLuaDataToJson(input);

    if (res.error) throw new Error(res.error);
    return res.result;
}
