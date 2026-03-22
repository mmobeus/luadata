use serde::Serialize;
use wasm_bindgen::prelude::*;

use luadata::options::{
    ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode, UnknownFieldMode,
};

#[derive(Serialize)]
struct WasmResult {
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

/// Convert Lua data to JSON.
///
/// Takes a Lua data string and an optional options object.
/// Returns a JS object with either `result` (JSON string) or `error` (message).
#[wasm_bindgen(js_name = "convertLuaDataToJson")]
pub fn convert_lua_data_to_json(input: &str, opts: JsValue) -> JsValue {
    let config = match parse_js_options(&opts) {
        Ok(c) => c,
        Err(e) => {
            return to_js_value(&WasmResult {
                result: None,
                error: Some(e),
            });
        }
    };

    match luadata::text_to_json("input", input, config) {
        Ok(json) => to_js_value(&WasmResult {
            result: Some(json),
            error: None,
        }),
        Err(e) => to_js_value(&WasmResult {
            result: None,
            error: Some(e),
        }),
    }
}

fn to_js_value<T: Serialize>(val: &T) -> JsValue {
    let serializer = serde_wasm_bindgen::Serializer::new().serialize_maps_as_objects(true);
    val.serialize(&serializer).unwrap_or(JsValue::NULL)
}

fn parse_js_options(opts: &JsValue) -> Result<ParseConfig, String> {
    let mut config = ParseConfig::new();

    if opts.is_undefined() || opts.is_null() {
        return Ok(config);
    }

    // Read options from JS object using js_sys
    if let Some(et) = get_string_field(opts, "emptyTable") {
        config.empty_table_mode = match et.as_str() {
            "null" => EmptyTableMode::Null,
            "omit" => EmptyTableMode::Omit,
            "array" => EmptyTableMode::Array,
            "object" => EmptyTableMode::Object,
            _ => return Err(format!("unknown emptyTable value: {:?}", et)),
        };
    }

    if let Some(am) = get_string_field(opts, "arrayMode") {
        let max_gap = get_number_field(opts, "arrayMaxGap").unwrap_or(20.0) as usize;
        config.array_mode = Some(match am.as_str() {
            "none" => ArrayMode::None,
            "index-only" => ArrayMode::IndexOnly,
            "sparse" => ArrayMode::Sparse { max_gap },
            _ => return Err(format!("unknown arrayMode value: {:?}", am)),
        });
    }

    if let Some(uf) = get_string_field(opts, "unknownFields") {
        config.unknown_field_mode = match uf.as_str() {
            "ignore" => UnknownFieldMode::Ignore,
            "include" => UnknownFieldMode::Include,
            "fail" => UnknownFieldMode::Fail,
            _ => return Err(format!("unknown unknownFields value: {:?}", uf)),
        };
    }

    if let Some(schema_str) = get_string_field(opts, "schema") {
        config.schema =
            Some(luadata::parse_schema(&schema_str).map_err(|e| format!("schema error: {}", e))?);
    }

    if let Some(st) = get_object_field(opts, "stringTransform") {
        let max_len = get_number_field(&st, "maxLen")
            .ok_or("stringTransform.maxLen must be a positive number")?
            as usize;

        if max_len == 0 {
            return Err("stringTransform.maxLen must be a positive number".to_string());
        }

        let mode_str = get_string_field(&st, "mode").unwrap_or("truncate".to_string());
        let mode = match mode_str.as_str() {
            "truncate" => StringTransformMode::Truncate,
            "empty" => StringTransformMode::Empty,
            "redact" => StringTransformMode::Redact,
            "replace" => StringTransformMode::Replace,
            _ => {
                return Err(format!(
                    "unknown stringTransform.mode value: {:?}",
                    mode_str
                ));
            }
        };

        let replacement = get_string_field(&st, "replacement").unwrap_or_default();

        config.string_transform = Some(StringTransform {
            max_len,
            mode,
            replacement,
        });
    }

    Ok(config)
}

fn get_string_field(obj: &JsValue, key: &str) -> Option<String> {
    let val = js_sys::Reflect::get(obj, &JsValue::from_str(key)).ok()?;
    if val.is_undefined() || val.is_null() {
        None
    } else {
        val.as_string()
    }
}

fn get_number_field(obj: &JsValue, key: &str) -> Option<f64> {
    let val = js_sys::Reflect::get(obj, &JsValue::from_str(key)).ok()?;
    if val.is_undefined() || val.is_null() {
        None
    } else {
        val.as_f64()
    }
}

fn get_object_field(obj: &JsValue, key: &str) -> Option<JsValue> {
    let val = js_sys::Reflect::get(obj, &JsValue::from_str(key)).ok()?;
    if val.is_undefined() || val.is_null() {
        None
    } else {
        Some(val)
    }
}
