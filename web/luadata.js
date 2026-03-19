// luadata.js — reusable wrapper around the luadata Rust WASM module.
//
// Usage:
//   import { init, convert } from "./luadata.js";
//   await init();                        // loads WASM, resolves when ready
//   const json = convert(luaString);     // returns parsed JSON string
//   const json = convert(luaString, {    // with options
//     emptyTable: "object",
//     arrayMode: "sparse",
//     arrayMaxGap: 10,
//     stringTransform: { maxLen: 100, mode: "redact" },
//   });

let wasmModule = null;

export async function init(pkgPath = "pkg") {
    const mod = await import(`./${pkgPath}/luadata_wasm.js`);
    await mod.default();
    wasmModule = mod;
}

export function convert(input, opts) {
    if (!wasmModule) throw new Error("luadata WASM not initialized — call init() first");

    const hasOpts = opts && Object.keys(opts).length > 0;
    const res = hasOpts
        ? wasmModule.convertLuaDataToJson(input, opts)
        : wasmModule.convertLuaDataToJson(input);

    if (res.error) throw new Error(res.error);
    return res.result;
}
