pub mod converter;
pub mod lexer;
pub mod options;
pub mod parser;
pub mod types;

// Re-export main API
pub use converter::{file_to_json, reader_to_json, text_to_json, to_json};
pub use options::{ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode};
pub use parser::parse_text;
pub use types::{Key, KeyType, KeyValuePair, KeyValuePairs, RawKey, RawValue, Value, ValueType};
