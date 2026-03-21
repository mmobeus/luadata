use napi::Error;
use napi_derive::napi;

use luadata::options::{
    ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode,
};

#[napi(object)]
pub struct StringTransformOptions {
    pub max_len: u32,
    pub mode: Option<String>,
    pub replacement: Option<String>,
}

#[napi(object)]
pub struct ConvertOptions {
    pub empty_table: Option<String>,
    pub array_mode: Option<String>,
    pub array_max_gap: Option<u32>,
    pub string_transform: Option<StringTransformOptions>,
}

/// Convert a Lua data string to a JSON string.
#[napi]
pub fn convert_lua_to_json(input: String, opts: Option<ConvertOptions>) -> napi::Result<String> {
    let config = build_config(opts)?;
    luadata::text_to_json("input", &input, config).map_err(Error::from_reason)
}

/// Convert a Lua data file to a JSON string.
#[napi]
pub fn convert_lua_file_to_json(
    path: String,
    opts: Option<ConvertOptions>,
) -> napi::Result<String> {
    let config = build_config(opts)?;
    luadata::file_to_json(&path, config).map_err(Error::from_reason)
}

fn build_config(opts: Option<ConvertOptions>) -> napi::Result<ParseConfig> {
    let mut config = ParseConfig::new();

    let opts = match opts {
        Some(o) => o,
        None => return Ok(config),
    };

    if let Some(et) = opts.empty_table {
        config.empty_table_mode = match et.as_str() {
            "null" => EmptyTableMode::Null,
            "omit" => EmptyTableMode::Omit,
            "array" => EmptyTableMode::Array,
            "object" => EmptyTableMode::Object,
            _ => {
                return Err(Error::from_reason(format!(
                    "unknown emptyTable value: {et:?}"
                )));
            }
        };
    }

    if let Some(am) = opts.array_mode {
        config.array_mode = Some(match am.as_str() {
            "none" => ArrayMode::None,
            "index-only" => ArrayMode::IndexOnly,
            "sparse" => ArrayMode::Sparse {
                max_gap: opts.array_max_gap.unwrap_or(20) as usize,
            },
            _ => {
                return Err(Error::from_reason(format!(
                    "unknown arrayMode value: {am:?}"
                )));
            }
        });
    }

    if let Some(st) = opts.string_transform {
        if st.max_len == 0 {
            return Err(Error::from_reason(
                "stringTransform.maxLen must be a positive number",
            ));
        }

        let mode = match st.mode.as_deref().unwrap_or("truncate") {
            "truncate" => StringTransformMode::Truncate,
            "empty" => StringTransformMode::Empty,
            "redact" => StringTransformMode::Redact,
            "replace" => StringTransformMode::Replace,
            m => {
                return Err(Error::from_reason(format!(
                    "unknown stringTransform.mode value: {m:?}"
                )));
            }
        };

        config.string_transform = Some(StringTransform {
            max_len: st.max_len as usize,
            mode,
            replacement: st.replacement.unwrap_or_default(),
        });
    }

    Ok(config)
}
