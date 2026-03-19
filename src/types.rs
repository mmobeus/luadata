/// The type of a key in a Lua key-value pair.
#[derive(Debug, Clone, PartialEq)]
pub enum KeyType {
    /// A Lua identifier (e.g., `x = ...`)
    Identifier,
    /// An implicit array index (e.g., `{"a","b"}`)
    Index,
    /// A string key (e.g., `["key"] = ...`)
    String,
    /// An integer key (e.g., `[1] = ...`)
    Int,
    /// A boolean key (e.g., `[true] = ...`)
    Bool,
    /// A float key (e.g., `[1.5] = ...`)
    Float,
}

/// The type of a Lua value.
#[derive(Debug, Clone, PartialEq)]
pub enum ValueType {
    Table,
    String,
    Int,
    Float,
    Bool,
    /// Empty table `{}` — ambiguously array or object.
    Empty,
    Nil,
}

/// A parsed Lua key.
#[derive(Debug, Clone)]
pub struct Key {
    pub key_type: KeyType,
    pub source: std::string::String,
    pub raw: RawKey,
}

/// The raw (typed) representation of a key.
#[derive(Debug, Clone)]
pub enum RawKey {
    String(std::string::String),
    Int(i64),
    Float(f64),
    Bool(bool),
}

impl RawKey {
    /// Returns the string representation used as JSON object key.
    pub fn as_json_key(&self) -> std::string::String {
        match self {
            RawKey::String(s) => s.clone(),
            RawKey::Int(i) => i.to_string(),
            RawKey::Float(f) => f.to_string(),
            RawKey::Bool(b) => b.to_string(),
        }
    }
}

/// A parsed Lua value.
#[derive(Debug, Clone)]
pub struct Value {
    pub value_type: ValueType,
    pub source: std::string::String,
    pub raw: RawValue,
    pub transformed: bool,
}

/// The raw (typed) representation of a value.
#[derive(Debug, Clone)]
pub enum RawValue {
    Table(KeyValuePairs),
    String(std::string::String),
    Int(i64),
    Float(f64),
    Bool(bool),
    Nil,
    Empty,
}

/// A single key-value pair from Lua data.
#[derive(Debug, Clone)]
pub struct KeyValuePair {
    pub key: Key,
    pub value: Value,
}

/// An ordered collection of key-value pairs from parsed Lua data.
#[derive(Debug, Clone)]
pub struct KeyValuePairs {
    pub ordered_pairs: Vec<KeyValuePair>,
    pub num_values: usize,
}

impl KeyValuePairs {
    pub fn new() -> Self {
        KeyValuePairs {
            ordered_pairs: Vec::new(),
            num_values: 0,
        }
    }

    pub fn len(&self) -> usize {
        self.ordered_pairs.len()
    }

    pub fn is_empty(&self) -> bool {
        self.ordered_pairs.is_empty()
    }
}

impl Default for KeyValuePairs {
    fn default() -> Self {
        Self::new()
    }
}

/// Checks if a value represents an empty Lua table.
pub fn is_empty_table(v: &Value) -> bool {
    match v.value_type {
        ValueType::Empty => true,
        ValueType::Table => {
            if let RawValue::Table(ref kvps) = v.raw {
                kvps.is_empty()
            } else {
                false
            }
        }
        _ => false,
    }
}

// JsonValue re-exported for downstream crates that need the raw JSON type.
pub use serde_json::Value as JsonValue;
