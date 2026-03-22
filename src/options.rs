/// Controls how strings exceeding MaxLen are transformed.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum StringTransformMode {
    /// Truncate to MaxLen bytes.
    Truncate,
    /// Replace with "".
    Empty,
    /// Replace with "[redacted]".
    Redact,
    /// Replace with a custom string.
    Replace,
}

/// Configures how long strings are handled during parsing.
#[derive(Debug, Clone)]
pub struct StringTransform {
    pub max_len: usize,
    pub mode: StringTransformMode,
    pub replacement: String, // only used with Replace
}

/// Controls how Lua tables with integer keys are rendered in JSON.
#[derive(Debug, Clone, PartialEq)]
pub enum ArrayMode {
    /// Disable all array rendering. Every table becomes a JSON object.
    None,
    /// Only implicit index tables ({"a","b","c"}) render as arrays.
    IndexOnly,
    /// Allow sparse integer-keyed tables to render as arrays within MaxGap.
    Sparse { max_gap: usize },
}

/// Controls how empty Lua tables ({}) are rendered in JSON.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum EmptyTableMode {
    /// Render as null (default).
    Null,
    /// Omit the key entirely.
    Omit,
    /// Render as [].
    Array,
    /// Render as {}.
    Object,
}

/// Controls how fields not present in the schema are handled.
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum UnknownFieldMode {
    /// Skip fields not in the schema (default).
    Ignore,
    /// Include unknown fields, converting without schema guidance.
    Include,
    /// Return an error when an unknown field is encountered.
    Fail,
}

/// Internal configuration for parsing.
#[derive(Debug, Clone)]
pub struct ParseConfig {
    pub string_transform: Option<StringTransform>,
    pub array_mode: Option<ArrayMode>,
    pub empty_table_mode: EmptyTableMode,
    pub schema: Option<crate::schema::SchemaNode>,
    pub unknown_field_mode: UnknownFieldMode,
}

impl ParseConfig {
    pub fn new() -> Self {
        ParseConfig {
            string_transform: None,
            array_mode: None,
            empty_table_mode: EmptyTableMode::Null,
            schema: None,
            unknown_field_mode: UnknownFieldMode::Ignore,
        }
    }

    pub fn effective_array_mode(&self) -> ArrayMode {
        self.array_mode
            .clone()
            .unwrap_or(ArrayMode::Sparse { max_gap: 20 })
    }

    pub fn effective_empty_table_mode(&self) -> EmptyTableMode {
        self.empty_table_mode
    }

    /// Apply string transform if configured and the string exceeds max_len.
    /// Returns (transformed_string, was_transformed).
    pub fn transform_string(&self, source: &str) -> (String, bool) {
        match &self.string_transform {
            Some(st) if source.len() > st.max_len => match st.mode {
                StringTransformMode::Truncate => (source[..st.max_len].to_string(), true),
                StringTransformMode::Empty => (String::new(), true),
                StringTransformMode::Redact => ("[redacted]".to_string(), true),
                StringTransformMode::Replace => (st.replacement.clone(), true),
            },
            _ => (source.to_string(), false),
        }
    }
}

impl Default for ParseConfig {
    fn default() -> Self {
        Self::new()
    }
}
