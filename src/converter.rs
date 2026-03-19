use serde_json::{Map, Value as JsonValue};
use std::collections::HashSet;
use std::io::Read;

use crate::options::{ArrayMode, EmptyTableMode, ParseConfig};
use crate::parser::parse_text;
use crate::types::*;

/// Parse Lua data text and return JSON as a string.
pub fn text_to_json(name: &str, text: &str, config: ParseConfig) -> Result<String, String> {
    let parsed = parse_text(name, text, config.clone())?;
    let json_value = convert_kvps_to_json(&parsed, &config);
    serde_json::to_string(&json_value).map_err(|e| format!("JSON serialization error: {}", e))
}

/// Parse Lua data bytes and return JSON as a string.
pub fn to_json(lua: &[u8], config: ParseConfig) -> Result<String, String> {
    let text = std::str::from_utf8(lua).map_err(|e| format!("invalid UTF-8: {}", e))?;
    text_to_json("input", text, config)
}

/// Parse Lua data from an io::Read and return JSON as a string.
pub fn reader_to_json(
    name: &str,
    reader: &mut dyn Read,
    config: ParseConfig,
) -> Result<String, String> {
    let mut text = String::new();
    reader
        .read_to_string(&mut text)
        .map_err(|e| format!("read error: {}", e))?;
    text_to_json(name, &text, config)
}

/// Parse a Lua data file and return JSON as a string.
pub fn file_to_json(path: &str, config: ParseConfig) -> Result<String, String> {
    let text = std::fs::read_to_string(path).map_err(|e| format!("file error: {}", e))?;
    text_to_json(path, &text, config)
}

/// Convert parsed KeyValuePairs to a serde_json::Value.
fn convert_kvps_to_json(kvps: &KeyValuePairs, config: &ParseConfig) -> JsonValue {
    convert_table(&kvps.ordered_pairs, config)
}

/// Convert a table (slice of KeyValuePairs) to a JSON value.
fn convert_table(table: &[KeyValuePair], config: &ParseConfig) -> JsonValue {
    let empty_mode = config.effective_empty_table_mode();

    // Empty table
    if table.is_empty() {
        return match empty_mode {
            EmptyTableMode::Array => JsonValue::Array(vec![]),
            EmptyTableMode::Object => JsonValue::Object(Map::new()),
            _ => JsonValue::Null, // Null and Omit
        };
    }

    let mode = config.effective_array_mode();

    match mode {
        ArrayMode::None => {
            // No array rendering; fall through to object
        }
        ArrayMode::IndexOnly | ArrayMode::Sparse { .. } => {
            // Check if all keys are implicit Index keys
            let all_index =
                !table.is_empty() && table.iter().all(|kv| kv.key.key_type == KeyType::Index);

            if all_index {
                let mut arr: Vec<JsonValue> = Vec::with_capacity(table.len());
                for kv in table {
                    if is_empty_table(&kv.value) && empty_mode == EmptyTableMode::Omit {
                        continue;
                    }
                    arr.push(value_to_json(&kv.value, config));
                }
                return JsonValue::Array(arr);
            }

            // For sparse mode, check integer keys within gap threshold
            if let ArrayMode::Sparse { max_gap } = mode
                && let Some(arr) = try_int_key_array(table, max_gap, config)
            {
                return JsonValue::Array(arr);
            }
        }
    }

    // Object rendering
    // Check for key collisions
    let mut seen = HashSet::with_capacity(table.len());
    let mut has_collision = false;
    for kv in table {
        if !seen.insert(&kv.key.source) {
            has_collision = true;
            break;
        }
    }

    let mut map = Map::new();

    if has_collision {
        map.insert(
            "_wtf_warning".to_string(),
            JsonValue::String("key_collision".to_string()),
        );
    }

    for kv in table {
        let key_str = kv.key.source.clone();
        if is_empty_table(&kv.value)
            && empty_mode == EmptyTableMode::Omit
            && kv.key.source != "@root"
        {
            continue;
        }
        let val = value_to_json(&kv.value, config);
        map.insert(key_str, val);
    }

    JsonValue::Object(map)
}

/// Try to render a table as a JSON array when all keys are Int type
/// and gaps between consecutive keys don't exceed max_gap.
fn try_int_key_array(
    table: &[KeyValuePair],
    max_gap: usize,
    config: &ParseConfig,
) -> Option<Vec<JsonValue>> {
    if table.is_empty() {
        return None;
    }

    let empty_mode = config.effective_empty_table_mode();
    let mut entries: Vec<(i64, &Value)> = Vec::with_capacity(table.len());
    let mut seen = HashSet::with_capacity(table.len());

    for kv in table {
        if kv.key.key_type != KeyType::Int {
            return None;
        }
        if let RawKey::Int(k) = kv.key.raw {
            if k < 1 {
                return None;
            }
            if !seen.insert(k) {
                return None; // duplicate key
            }
            entries.push((k, &kv.value));
        } else {
            return None;
        }
    }

    entries.sort_by_key(|&(k, _)| k);

    // Check gaps
    let mut prev: i64 = 0;
    for &(k, _) in &entries {
        let gap = k - prev - 1;
        if gap > max_gap as i64 {
            return None;
        }
        prev = k;
    }

    // Build array
    let max_key = entries.last().unwrap().0 as usize;
    let mut arr: Vec<JsonValue> = vec![JsonValue::Null; max_key];

    for (k, v) in entries {
        let idx = (k - 1) as usize;
        if is_empty_table(v) {
            if empty_mode == EmptyTableMode::Omit {
                continue; // leave as null
            }
            arr[idx] = empty_table_json(empty_mode);
        } else {
            arr[idx] = value_to_json(v, config);
        }
    }

    Some(arr)
}

/// Convert a single Value to a JSON value.
fn value_to_json(v: &Value, config: &ParseConfig) -> JsonValue {
    if is_empty_table(v) {
        return empty_table_json(config.effective_empty_table_mode());
    }
    match &v.raw {
        RawValue::String(s) => JsonValue::String(s.clone()),
        RawValue::Int(i) => JsonValue::Number(serde_json::Number::from(*i)),
        RawValue::Float(f) => serde_json::Number::from_f64(*f)
            .map(JsonValue::Number)
            .unwrap_or(JsonValue::Null),
        RawValue::Bool(b) => JsonValue::Bool(*b),
        RawValue::Nil => JsonValue::Null,
        RawValue::Empty => empty_table_json(config.effective_empty_table_mode()),
        RawValue::Table(kvps) => convert_table(&kvps.ordered_pairs, config),
    }
}

/// Return the JSON representation of an empty table for the given mode.
fn empty_table_json(mode: EmptyTableMode) -> JsonValue {
    match mode {
        EmptyTableMode::Array => JsonValue::Array(vec![]),
        EmptyTableMode::Object => JsonValue::Object(Map::new()),
        _ => JsonValue::Null,
    }
}
