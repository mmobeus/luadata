use std::ffi::{CStr, CString};
use std::os::raw::c_char;

use serde::{Deserialize, Serialize};

use luadata::options::{
    ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode,
};

/// JSON options structure matching the Go clib interface.
#[derive(Deserialize)]
struct OptionsJSON {
    #[serde(default)]
    empty_table: Option<String>,
    #[serde(default)]
    array_mode: Option<String>,
    #[serde(default)]
    array_max_gap: Option<usize>,
    #[serde(default)]
    string_transform: Option<StringTransformJSON>,
}

#[derive(Deserialize)]
struct StringTransformJSON {
    max_len: usize,
    #[serde(default)]
    mode: Option<String>,
    #[serde(default)]
    replacement: Option<String>,
}

/// Response envelope matching the Go clib interface.
#[derive(Serialize)]
struct ResultJSON {
    #[serde(skip_serializing_if = "Option::is_none")]
    result: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    error: Option<String>,
}

fn parse_options(raw: &str) -> Result<ParseConfig, String> {
    let mut config = ParseConfig::new();

    if raw.is_empty() {
        return Ok(config);
    }

    let oj: OptionsJSON =
        serde_json::from_str(raw).map_err(|e| format!("invalid options JSON: {}", e))?;

    if let Some(ref et) = oj.empty_table {
        config.empty_table_mode = match et.as_str() {
            "null" => EmptyTableMode::Null,
            "omit" => EmptyTableMode::Omit,
            "array" => EmptyTableMode::Array,
            "object" => EmptyTableMode::Object,
            _ => return Err(format!("unknown empty_table value: {:?}", et)),
        };
    }

    if let Some(ref am) = oj.array_mode {
        config.array_mode = Some(match am.as_str() {
            "none" => ArrayMode::None,
            "index-only" => ArrayMode::IndexOnly,
            "sparse" => ArrayMode::Sparse {
                max_gap: oj.array_max_gap.unwrap_or(20),
            },
            _ => return Err(format!("unknown array_mode value: {:?}", am)),
        });
    }

    if let Some(ref st) = oj.string_transform {
        if st.max_len == 0 {
            return Err("string_transform.max_len must be a positive number".to_string());
        }

        let mode_str = st.mode.as_deref().unwrap_or("truncate");
        let mode = match mode_str {
            "truncate" => StringTransformMode::Truncate,
            "empty" => StringTransformMode::Empty,
            "redact" => StringTransformMode::Redact,
            "replace" => StringTransformMode::Replace,
            _ => {
                return Err(format!(
                    "unknown string_transform.mode value: {:?}",
                    mode_str
                ));
            }
        };

        config.string_transform = Some(StringTransform {
            max_len: st.max_len,
            mode,
            replacement: st.replacement.clone().unwrap_or_default(),
        });
    }

    Ok(config)
}

fn marshal_result(r: ResultJSON) -> *mut c_char {
    let json = serde_json::to_string(&r)
        .unwrap_or_else(|_| r#"{"error":"serialization failed"}"#.to_string());
    CString::new(json).unwrap_or_default().into_raw()
}

/// Convert Lua data to JSON.
///
/// # Safety
///
/// Both `input` and `options` must be valid, null-terminated C strings.
/// The returned pointer must be freed with `LuaDataFree`.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn LuaDataToJSON(
    input: *const c_char,
    options: *const c_char,
) -> *mut c_char {
    let input_str = unsafe {
        if input.is_null() {
            ""
        } else {
            match CStr::from_ptr(input).to_str() {
                Ok(s) => s,
                Err(e) => {
                    return marshal_result(ResultJSON {
                        result: None,
                        error: Some(format!("invalid UTF-8 input: {}", e)),
                    });
                }
            }
        }
    };

    let options_str = unsafe {
        if options.is_null() {
            ""
        } else {
            match CStr::from_ptr(options).to_str() {
                Ok(s) => s,
                Err(e) => {
                    return marshal_result(ResultJSON {
                        result: None,
                        error: Some(format!("invalid UTF-8 options: {}", e)),
                    });
                }
            }
        }
    };

    let config = match parse_options(options_str) {
        Ok(c) => c,
        Err(e) => {
            return marshal_result(ResultJSON {
                result: None,
                error: Some(e),
            });
        }
    };

    match luadata::text_to_json("input", input_str, config) {
        Ok(json) => marshal_result(ResultJSON {
            result: Some(json),
            error: None,
        }),
        Err(e) => marshal_result(ResultJSON {
            result: None,
            error: Some(e),
        }),
    }
}

/// Free a string returned by `LuaDataToJSON`.
///
/// # Safety
///
/// `ptr` must be a pointer previously returned by `LuaDataToJSON`, or null.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn LuaDataFree(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe {
            drop(CString::from_raw(ptr));
        }
    }
}
