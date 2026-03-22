use base64::Engine;
use serde_json::{Map, Value as JsonValue};
use std::collections::HashSet;
use std::io::Read;

use crate::options::{ArrayMode, EmptyTableMode, ParseConfig, UnknownFieldMode};
use crate::parser::{parse_bytes, parse_text};
use crate::schema::{SchemaNode, SchemaType, StringFormat};
use crate::types::*;

/// Parse Lua data text and return JSON as a string.
pub fn text_to_json(name: &str, text: &str, config: ParseConfig) -> Result<String, String> {
    let parsed = parse_text(name, text, config.clone())?;
    let json_value = convert_kvps_to_json(&parsed, &config)?;
    serde_json::to_string(&json_value).map_err(|e| format!("JSON serialization error: {}", e))
}

/// Parse Lua data bytes and return JSON as a string.
///
/// The lexer operates on raw bytes. Inside string literals, bytes are preserved
/// losslessly: if the string's bytes are valid UTF-8, they decode as UTF-8
/// (so "Fröst" renders correctly); otherwise each byte maps to its Latin-1
/// code point (so binary blobs round-trip perfectly).
pub fn to_json(lua: &[u8], config: ParseConfig) -> Result<String, String> {
    let parsed = parse_bytes("input", lua, config.clone())?;
    let json_value = convert_kvps_to_json(&parsed, &config)?;
    serde_json::to_string(&json_value).map_err(|e| format!("JSON serialization error: {}", e))
}

/// Parse Lua data from an io::Read and return JSON as a string.
pub fn reader_to_json(
    name: &str,
    reader: &mut dyn Read,
    config: ParseConfig,
) -> Result<String, String> {
    let mut buf = Vec::new();
    reader
        .read_to_end(&mut buf)
        .map_err(|e| format!("read error: {}", e))?;
    let parsed = parse_bytes(name, &buf, config.clone())?;
    let json_value = convert_kvps_to_json(&parsed, &config)?;
    serde_json::to_string(&json_value).map_err(|e| format!("JSON serialization error: {}", e))
}

/// Parse a Lua data file and return JSON as a string.
pub fn file_to_json(path: &str, config: ParseConfig) -> Result<String, String> {
    let buf = std::fs::read(path).map_err(|e| format!("file error: {}", e))?;
    let parsed = parse_bytes(path, &buf, config.clone())?;
    let json_value = convert_kvps_to_json(&parsed, &config)?;
    serde_json::to_string(&json_value).map_err(|e| format!("JSON serialization error: {}", e))
}

/// Convert parsed KeyValuePairs to a serde_json::Value.
fn convert_kvps_to_json(kvps: &KeyValuePairs, config: &ParseConfig) -> Result<JsonValue, String> {
    let schema = config.schema.as_ref();
    convert_table(&kvps.ordered_pairs, config, schema)
}

/// Convert a table (slice of KeyValuePairs) to a JSON value.
fn convert_table(
    table: &[KeyValuePair],
    config: &ParseConfig,
    schema: Option<&SchemaNode>,
) -> Result<JsonValue, String> {
    // Schema-driven path: if schema specifies the type, use it directly
    if let Some(schema) = schema {
        return convert_table_with_schema(table, config, schema);
    }

    // Heuristic path: existing behavior
    let empty_mode = config.effective_empty_table_mode();

    if table.is_empty() {
        return Ok(match empty_mode {
            EmptyTableMode::Array => JsonValue::Array(vec![]),
            EmptyTableMode::Object => JsonValue::Object(Map::new()),
            _ => JsonValue::Null,
        });
    }

    let mode = config.effective_array_mode();

    match mode {
        ArrayMode::None => {
            // No array rendering; fall through to object
        }
        ArrayMode::IndexOnly | ArrayMode::Sparse { .. } => {
            let all_index =
                !table.is_empty() && table.iter().all(|kv| kv.key.key_type == KeyType::Index);

            if all_index {
                let mut arr: Vec<JsonValue> = Vec::with_capacity(table.len());
                for kv in table {
                    if is_empty_table(&kv.value) && empty_mode == EmptyTableMode::Omit {
                        continue;
                    }
                    arr.push(value_to_json(&kv.value, config, None)?);
                }
                return Ok(JsonValue::Array(arr));
            }

            if let ArrayMode::Sparse { max_gap } = mode
                && let Some(arr) = try_int_key_array(table, max_gap, config, None)?
            {
                return Ok(JsonValue::Array(arr));
            }
        }
    }

    convert_table_as_object(table, config, None)
}

/// Convert a table using schema guidance.
fn convert_table_with_schema(
    table: &[KeyValuePair],
    config: &ParseConfig,
    schema: &SchemaNode,
) -> Result<JsonValue, String> {
    match &schema.schema_type {
        SchemaType::Array { items } => {
            if table.is_empty() {
                return Ok(JsonValue::Array(vec![]));
            }
            let item_schema = items.as_deref();
            let mut arr: Vec<JsonValue> = Vec::with_capacity(table.len());
            for kv in table {
                arr.push(value_to_json(&kv.value, config, item_schema)?);
            }
            Ok(JsonValue::Array(arr))
        }
        SchemaType::Object {
            properties,
            additional_properties,
        } => {
            if table.is_empty() {
                return Ok(JsonValue::Object(Map::new()));
            }
            convert_table_as_object_with_schema(
                table,
                config,
                properties,
                additional_properties.as_deref(),
            )
        }
        _ => {
            // Schema type doesn't describe a table structure — fall through to heuristics
            convert_table(table, config, None)
        }
    }
}

/// Convert a table as a JSON object, applying schema property lookups for field filtering.
/// `additional_properties` provides the schema for keys not in `properties` (map value type).
fn convert_table_as_object_with_schema(
    table: &[KeyValuePair],
    config: &ParseConfig,
    properties: &indexmap::IndexMap<String, SchemaNode>,
    additional_properties: Option<&SchemaNode>,
) -> Result<JsonValue, String> {
    let empty_mode = config.effective_empty_table_mode();

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
        let key_str = &kv.key.source;

        // Look up in explicit properties first, then fall back to additionalProperties
        let prop_schema = properties.get(key_str).or(additional_properties);

        // Unknown field handling — only apply when the schema constrains keys
        // (has properties or additionalProperties) and no schema matched
        let schema_constrains_keys = !properties.is_empty() || additional_properties.is_some();
        if properties.get(key_str).is_none()
            && additional_properties.is_none()
            && schema_constrains_keys
        {
            match config.unknown_field_mode {
                UnknownFieldMode::Ignore => continue,
                UnknownFieldMode::Fail => {
                    return Err(format!("unknown field: {:?}", key_str));
                }
                UnknownFieldMode::Include => {
                    // Fall through to convert without schema
                }
            }
        }

        if is_empty_table(&kv.value)
            && empty_mode == EmptyTableMode::Omit
            && *key_str != "@root"
            && prop_schema.is_none()
        {
            continue;
        }

        let val = value_to_json(&kv.value, config, prop_schema)?;
        map.insert(key_str.clone(), val);
    }

    Ok(JsonValue::Object(map))
}

/// Convert a table as a JSON object without schema (existing behavior).
fn convert_table_as_object(
    table: &[KeyValuePair],
    config: &ParseConfig,
    schema: Option<&SchemaNode>,
) -> Result<JsonValue, String> {
    // If we have an object schema, delegate to the schema-aware path
    if let Some(schema) = schema
        && let SchemaType::Object {
            properties,
            additional_properties,
        } = &schema.schema_type
    {
        return convert_table_as_object_with_schema(
            table,
            config,
            properties,
            additional_properties.as_deref(),
        );
    }

    let empty_mode = config.effective_empty_table_mode();

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
        let val = value_to_json(&kv.value, config, None)?;
        map.insert(key_str, val);
    }

    Ok(JsonValue::Object(map))
}

/// Try to render a table as a JSON array when all keys are Int type
/// and gaps between consecutive keys don't exceed max_gap.
fn try_int_key_array(
    table: &[KeyValuePair],
    max_gap: usize,
    config: &ParseConfig,
    item_schema: Option<&SchemaNode>,
) -> Result<Option<Vec<JsonValue>>, String> {
    if table.is_empty() {
        return Ok(None);
    }

    let empty_mode = config.effective_empty_table_mode();
    let mut entries: Vec<(i64, &Value)> = Vec::with_capacity(table.len());
    let mut seen = HashSet::with_capacity(table.len());

    for kv in table {
        if kv.key.key_type != KeyType::Int {
            return Ok(None);
        }
        if let RawKey::Int(k) = kv.key.raw {
            if k < 1 {
                return Ok(None);
            }
            if !seen.insert(k) {
                return Ok(None);
            }
            entries.push((k, &kv.value));
        } else {
            return Ok(None);
        }
    }

    entries.sort_by_key(|&(k, _)| k);

    let mut prev: i64 = 0;
    for &(k, _) in &entries {
        let gap = k - prev - 1;
        if gap > max_gap as i64 {
            return Ok(None);
        }
        prev = k;
    }

    let max_key = entries.last().unwrap().0 as usize;
    let mut arr: Vec<JsonValue> = vec![JsonValue::Null; max_key];

    for (k, v) in entries {
        let idx = (k - 1) as usize;
        if is_empty_table(v) {
            if empty_mode == EmptyTableMode::Omit {
                continue;
            }
            arr[idx] = empty_table_json(empty_mode);
        } else {
            arr[idx] = value_to_json(v, config, item_schema)?;
        }
    }

    Ok(Some(arr))
}

/// Convert a single Value to a JSON value, optionally guided by a schema.
fn value_to_json(
    v: &Value,
    config: &ParseConfig,
    schema: Option<&SchemaNode>,
) -> Result<JsonValue, String> {
    if is_empty_table(v) {
        if let Some(schema) = schema {
            return Ok(match &schema.schema_type {
                SchemaType::Array { .. } => JsonValue::Array(vec![]),
                SchemaType::Object { .. } => JsonValue::Object(Map::new()),
                _ => empty_table_json(config.effective_empty_table_mode()),
            });
        }
        return Ok(empty_table_json(config.effective_empty_table_mode()));
    }

    match &v.raw {
        RawValue::String(bytes) => Ok(convert_string(bytes, schema)),
        RawValue::Int(i) => {
            // Safe coercion: int → float when schema says Number
            if let Some(SchemaNode {
                schema_type: SchemaType::Number,
            }) = schema
            {
                Ok(serde_json::Number::from_f64(*i as f64)
                    .map(JsonValue::Number)
                    .unwrap_or(JsonValue::Null))
            } else {
                Ok(JsonValue::Number(serde_json::Number::from(*i)))
            }
        }
        RawValue::Float(f) => Ok(serde_json::Number::from_f64(*f)
            .map(JsonValue::Number)
            .unwrap_or(JsonValue::Null)),
        RawValue::Bool(b) => Ok(JsonValue::Bool(*b)),
        RawValue::Nil => Ok(JsonValue::Null),
        RawValue::Empty => {
            if let Some(schema) = schema {
                Ok(match &schema.schema_type {
                    SchemaType::Array { .. } => JsonValue::Array(vec![]),
                    SchemaType::Object { .. } => JsonValue::Object(Map::new()),
                    _ => empty_table_json(config.effective_empty_table_mode()),
                })
            } else {
                Ok(empty_table_json(config.effective_empty_table_mode()))
            }
        }
        RawValue::Table(kvps) => convert_table(&kvps.ordered_pairs, config, schema),
    }
}

/// Convert raw string bytes to a JSON value, using schema format when available.
///
/// The raw bytes are the decoded content of the Lua string literal (after escape
/// processing). The encoding decision is made here:
/// - No schema or schema says plain string: apply UTF-8/Latin-1 heuristic
///   (valid UTF-8 → decode as UTF-8, otherwise each byte → Latin-1 code point)
/// - `format: "bytes"`: emit raw bytes as a JSON array of integers
/// - `format: "base64"`: emit raw bytes as a base64-encoded string
/// - `format: "latin1"`: force Latin-1 (each byte → code point, no UTF-8 attempt)
fn convert_string(raw_bytes: &[u8], schema: Option<&SchemaNode>) -> JsonValue {
    let format = schema.and_then(|s| match &s.schema_type {
        SchemaType::String { format } => format.as_ref(),
        _ => None,
    });

    match format {
        Some(StringFormat::Bytes) => {
            let vals: Vec<JsonValue> = raw_bytes
                .iter()
                .map(|&b| JsonValue::Number(serde_json::Number::from(b)))
                .collect();
            JsonValue::Array(vals)
        }
        Some(StringFormat::Base64) => {
            let encoded = base64::engine::general_purpose::STANDARD.encode(raw_bytes);
            JsonValue::String(encoded)
        }
        Some(StringFormat::Latin1) => {
            // Force Latin-1: each byte maps to its code point, no UTF-8 decoding
            let s: String = raw_bytes.iter().map(|&b| b as char).collect();
            JsonValue::String(s)
        }
        None => {
            // No schema opinion: apply the UTF-8/Latin-1 heuristic
            let s = crate::lexer::bytes_to_string(raw_bytes);
            JsonValue::String(s)
        }
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
