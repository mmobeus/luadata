// luadata.js — reusable wrapper around the luadata WASM module.
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

let ready = false;

export function init(wasmPath = "luadata.wasm") {
    return new Promise((resolve, reject) => {
        const go = new Go();
        WebAssembly.instantiateStreaming(fetch(wasmPath), go.importObject).then((result) => {
            go.run(result.instance);
            ready = true;
            resolve();
        }).catch(reject);
    });
}

export function convert(input, opts) {
    if (!ready) throw new Error("luadata WASM not initialized — call init() first");

    const hasOpts = opts && Object.keys(opts).length > 0;
    const res = hasOpts ? convertLuaDataToJson(input, opts) : convertLuaDataToJson(input);
    if (res.error) throw new Error(res.error);
    return res.result;
}
