use indexmap::IndexMap;
use serde_json::Value as JsonValue;

/// How a string field's bytes should be encoded in JSON output.
#[derive(Debug, Clone, PartialEq)]
pub enum StringFormat {
    /// Current behavior: UTF-8 if valid, Latin-1 code points otherwise.
    Latin1,
    /// Emit as a JSON array of byte values [104, 101, …].
    Bytes,
    /// Emit as a base64-encoded JSON string.
    Base64,
}

/// The subset of JSON Schema types we support.
#[derive(Debug, Clone, PartialEq)]
pub enum SchemaType {
    String {
        format: Option<StringFormat>,
    },
    Integer,
    /// JSON Schema "number" — includes floats.
    Number,
    Boolean,
    Null,
    Object {
        properties: IndexMap<String, SchemaNode>,
        /// Schema for properties not listed in `properties`.
        /// Acts as a `map<string, T>` type when `properties` is empty.
        additional_properties: Option<Box<SchemaNode>>,
    },
    Array {
        items: Option<Box<SchemaNode>>,
    },
}

/// A node in our internal schema tree.
#[derive(Debug, Clone, PartialEq)]
pub struct SchemaNode {
    pub schema_type: SchemaType,
}

/// Parse a JSON Schema string into our internal representation.
///
/// We support the structural subset: `type`, `properties`, `items`, `format`.
/// Validation keywords (minLength, pattern, etc.) are silently ignored.
///
/// Future: when JSON → Lua conversion is added, custom extension keywords
/// (e.g., `x-lua-key-type: "integer"`) can be parsed here to preserve
/// Lua-specific type information for lossless round-trips. These keywords
/// are already silently ignored by the current parser.
pub fn parse_schema(json_str: &str) -> Result<SchemaNode, String> {
    let val: JsonValue =
        serde_json::from_str(json_str).map_err(|e| format!("invalid schema JSON: {}", e))?;
    parse_schema_value(&val)
}

fn parse_schema_value(val: &JsonValue) -> Result<SchemaNode, String> {
    let obj = val
        .as_object()
        .ok_or_else(|| "schema node must be a JSON object".to_string())?;

    let type_str = obj.get("type").and_then(|v| v.as_str());

    // Infer object type if "properties" present but no "type"
    let effective_type = match type_str {
        Some(t) => t,
        None if obj.contains_key("properties") => "object",
        None => return Err("schema node must have a \"type\" field".to_string()),
    };

    let schema_type = match effective_type {
        "string" => {
            let format = obj
                .get("format")
                .and_then(|v| v.as_str())
                .map(|f| match f {
                    "bytes" => Ok(StringFormat::Bytes),
                    "base64" => Ok(StringFormat::Base64),
                    "latin1" => Ok(StringFormat::Latin1),
                    other => Err(format!("unknown string format: {:?}", other)),
                })
                .transpose()?;
            SchemaType::String { format }
        }
        "integer" => SchemaType::Integer,
        "number" => SchemaType::Number,
        "boolean" => SchemaType::Boolean,
        "null" => SchemaType::Null,
        "object" => {
            let mut properties = IndexMap::new();
            if let Some(props) = obj.get("properties").and_then(|v| v.as_object()) {
                for (key, prop_schema) in props {
                    properties.insert(key.clone(), parse_schema_value(prop_schema)?);
                }
            }
            // additionalProperties can be a schema object (map value type)
            // or a boolean (true = allow any, false = disallow). We only
            // parse the object form; booleans are ignored.
            let additional_properties = obj
                .get("additionalProperties")
                .filter(|v| v.is_object())
                .map(|v| parse_schema_value(v).map(Box::new))
                .transpose()?;
            SchemaType::Object {
                properties,
                additional_properties,
            }
        }
        "array" => {
            let items = obj
                .get("items")
                .map(|v| parse_schema_value(v).map(Box::new))
                .transpose()?;
            SchemaType::Array { items }
        }
        other => return Err(format!("unsupported schema type: {:?}", other)),
    };

    Ok(SchemaNode { schema_type })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_simple_string() {
        let schema = parse_schema(r#"{"type": "string"}"#).unwrap();
        assert_eq!(
            schema,
            SchemaNode {
                schema_type: SchemaType::String { format: None }
            }
        );
    }

    #[test]
    fn parse_string_with_format() {
        let schema = parse_schema(r#"{"type": "string", "format": "bytes"}"#).unwrap();
        assert_eq!(
            schema,
            SchemaNode {
                schema_type: SchemaType::String {
                    format: Some(StringFormat::Bytes)
                }
            }
        );
    }

    #[test]
    fn parse_object_with_properties() {
        let schema = parse_schema(
            r#"{"type": "object", "properties": {"name": {"type": "string"}, "count": {"type": "integer"}}}"#,
        )
        .unwrap();
        match &schema.schema_type {
            SchemaType::Object { properties, .. } => {
                assert_eq!(properties.len(), 2);
                assert!(properties.contains_key("name"));
                assert!(properties.contains_key("count"));
            }
            _ => panic!("expected object schema"),
        }
    }

    #[test]
    fn parse_inferred_object() {
        let schema = parse_schema(r#"{"properties": {"x": {"type": "integer"}}}"#).unwrap();
        match &schema.schema_type {
            SchemaType::Object { properties, .. } => {
                assert_eq!(properties.len(), 1);
            }
            _ => panic!("expected inferred object schema"),
        }
    }

    #[test]
    fn parse_array_with_items() {
        let schema = parse_schema(r#"{"type": "array", "items": {"type": "string"}}"#).unwrap();
        match &schema.schema_type {
            SchemaType::Array { items } => {
                assert!(items.is_some());
                assert_eq!(
                    items.as_ref().unwrap().schema_type,
                    SchemaType::String { format: None }
                );
            }
            _ => panic!("expected array schema"),
        }
    }

    #[test]
    fn parse_nested_object_array() {
        let schema = parse_schema(
            r#"{
                "type": "object",
                "properties": {
                    "items": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "id": {"type": "integer"},
                                "name": {"type": "string"}
                            }
                        }
                    }
                }
            }"#,
        )
        .unwrap();
        match &schema.schema_type {
            SchemaType::Object { properties, .. } => {
                let items_schema = properties.get("items").unwrap();
                match &items_schema.schema_type {
                    SchemaType::Array { items } => {
                        let item = items.as_ref().unwrap();
                        match &item.schema_type {
                            SchemaType::Object { properties, .. } => {
                                assert_eq!(properties.len(), 2);
                            }
                            _ => panic!("expected nested object"),
                        }
                    }
                    _ => panic!("expected array"),
                }
            }
            _ => panic!("expected object"),
        }
    }

    #[test]
    fn parse_unknown_format_errors() {
        let result = parse_schema(r#"{"type": "string", "format": "uuid"}"#);
        assert!(result.is_err());
    }

    #[test]
    fn parse_unknown_type_errors() {
        let result = parse_schema(r#"{"type": "tuple"}"#);
        assert!(result.is_err());
    }

    #[test]
    fn parse_missing_type_errors() {
        let result = parse_schema(r#"{"description": "no type here"}"#);
        assert!(result.is_err());
    }

    #[test]
    fn ignores_unknown_keywords() {
        let schema = parse_schema(
            r#"{"type": "string", "minLength": 1, "maxLength": 100, "pattern": ".*"}"#,
        )
        .unwrap();
        assert_eq!(
            schema,
            SchemaNode {
                schema_type: SchemaType::String { format: None }
            }
        );
    }
}
