/**
 * Options for string length transformation.
 */
export interface StringTransformOptions {
    /** Maximum string length before the transform is applied. */
    maxLen: number;
    /** Transform mode. Default: "truncate". */
    mode?: "truncate" | "empty" | "redact" | "replace";
    /** Replacement string (only used with mode "replace"). */
    replacement?: string;
}

/**
 * Options for the convert function.
 */
export interface ConvertOptions {
    /** How to render empty Lua tables. Default: "null". */
    emptyTable?: "null" | "omit" | "array" | "object";
    /** Array detection mode. Default: "sparse". */
    arrayMode?: "none" | "index-only" | "sparse";
    /** Maximum gap between integer keys for sparse array detection. Default: 20. */
    arrayMaxGap?: number;
    /** String length transformation options. */
    stringTransform?: StringTransformOptions;
}

/**
 * Initialize the WASM module. Must be called once before using convert().
 */
export function init(): Promise<void>;

/**
 * Convert a Lua data string to a JSON string.
 *
 * @param input - Lua data source text
 * @param opts - Conversion options
 * @returns JSON string
 * @throws If the WASM module is not initialized or parsing fails
 */
export function convert(input: string, opts?: ConvertOptions): string;
