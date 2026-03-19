use crate::lexer::{Lexer, is_alpha_numeric};
use crate::options::ParseConfig;
use crate::types::*;

const EOF: char = '\0';

/// Parse Lua data from a string. Returns structured key-value pairs.
pub fn parse_text(name: &str, text: &str, config: ParseConfig) -> Result<KeyValuePairs, String> {
    let mut lex = Lexer::new(text, config);
    let mut kvps = KeyValuePairs::new();

    let mut first_iteration = true;
    loop {
        lex.skip_white_space().map_err(|e| {
            format!(
                "parse failure in {}: line {}, col {}, next {:?}: {}",
                name,
                lex.line,
                lex.col(),
                lex.peek_string(),
                e
            )
        })?;

        if lex.peek() == EOF {
            break;
        }

        if first_iteration {
            first_iteration = false;
            let r = lex.peek();

            let is_raw_value = if !r.is_alphabetic() && r != '_' {
                true
            } else {
                // Could be identifier=value OR a bare keyword (true/false/nil)
                let saved = lex.save();
                let _ = read_lua_identifier(&mut lex);
                lex.skip_space_runes();
                let is_raw = lex.peek() != '=';
                lex.restore(saved);
                is_raw
            };

            if is_raw_value {
                let kvpair = read_raw_value(&mut lex).map_err(|e| {
                    format!(
                        "parse failure in {}: line {}, col {}, next {:?}: {}",
                        name,
                        lex.line,
                        lex.col(),
                        lex.peek_string(),
                        e
                    )
                })?;
                kvps.ordered_pairs.push(kvpair);

                // Expect EOF after raw value
                lex.skip_white_space()
                    .map_err(|e| format!("parse failure in {}: {}", name, e))?;
                if lex.peek() != EOF {
                    return Err(format!(
                        "parse failure in {}: unexpected content after raw value at line {}, col {}",
                        name,
                        lex.line,
                        lex.col()
                    ));
                }
                break;
            }
        }

        let kvpair = read_saved_variable(&mut lex).map_err(|e| {
            format!(
                "parse failure in {}: line {}, col {}, next {:?}: {}",
                name,
                lex.line,
                lex.col(),
                lex.peek_string(),
                e
            )
        })?;
        kvps.ordered_pairs.push(kvpair);
    }

    kvps.num_values = lex.num_values;
    Ok(kvps)
}

fn read_raw_value(lex: &mut Lexer) -> Result<KeyValuePair, String> {
    let value = read_lua_value(lex)?;
    Ok(KeyValuePair {
        key: Key {
            key_type: KeyType::Identifier,
            source: "@root".to_string(),
            raw: RawKey::String("@root".to_string()),
        },
        value,
    })
}

fn read_saved_variable(lex: &mut Lexer) -> Result<KeyValuePair, String> {
    let ident = read_lua_identifier(lex)?;
    lex.skip_white_space()?;

    if lex.next_rune() != '=' {
        return Err("expected '=' after identifier".to_string());
    }

    lex.skip_white_space()?;
    let value = read_lua_value(lex)?;

    Ok(KeyValuePair {
        key: Key {
            key_type: KeyType::Identifier,
            source: ident.clone(),
            raw: RawKey::String(ident),
        },
        value,
    })
}

fn read_lua_identifier(lex: &mut Lexer) -> Result<String, String> {
    if lex.peek().is_ascii_digit() {
        return Err("expected identifier, but got a number".to_string());
    }

    while is_alpha_numeric(lex.next_rune()) {}
    lex.backup();

    let ident = lex.take();
    if ident.is_empty() {
        return Err("expected identifier".to_string());
    }

    Ok(ident)
}

fn read_lua_value(lex: &mut Lexer) -> Result<Value, String> {
    let r = lex.peek();
    let result = match r {
        '{' => read_lua_table(lex),
        '"' => read_quoted_string_value(lex),
        r if r.is_ascii_digit() || r == '-' => read_number_value(lex),
        _ => {
            let value = read_lua_identifier(lex)?;
            match value.as_str() {
                "true" => Ok(Value {
                    value_type: ValueType::Bool,
                    source: "true".to_string(),
                    raw: RawValue::Bool(true),
                    transformed: false,
                }),
                "false" => Ok(Value {
                    value_type: ValueType::Bool,
                    source: "false".to_string(),
                    raw: RawValue::Bool(false),
                    transformed: false,
                }),
                "nil" => Ok(Value {
                    value_type: ValueType::Nil,
                    source: "nil".to_string(),
                    raw: RawValue::Nil,
                    transformed: false,
                }),
                _ => Err(format!("expected to read a value, got {:?}", value)),
            }
        }
    };

    if result.is_ok() {
        lex.num_values += 1;
    }
    result
}

fn read_lua_table(lex: &mut Lexer) -> Result<Value, String> {
    if !lex.accept("{") {
        return Err("expected '{'".to_string());
    }

    if lex.accept("}") {
        lex.ignore();
        return Ok(Value {
            value_type: ValueType::Empty,
            source: String::new(),
            raw: RawValue::Empty,
            transformed: false,
        });
    }

    let mut table_value = KeyValuePairs::new();

    loop {
        lex.skip_white_space()?;

        if lex.accept("}") {
            lex.skip_white_space()?;
            return Ok(Value {
                value_type: ValueType::Table,
                source: String::new(),
                raw: RawValue::Table(table_value),
                transformed: false,
            });
        }

        let r = lex.peek();

        let key = if r == '[' {
            let k = read_lua_table_key(lex)?;
            lex.skip_white_space()?;
            if !lex.accept("=") {
                return Err("expected '='".to_string());
            }
            lex.skip_white_space()?;
            k
        } else {
            let index = table_value.ordered_pairs.len() + 1;
            Key {
                key_type: KeyType::Index,
                source: index.to_string(),
                raw: RawKey::Int(table_value.ordered_pairs.len() as i64),
            }
        };

        let val = read_lua_value(lex)?;
        table_value
            .ordered_pairs
            .push(KeyValuePair { key, value: val });

        lex.skip_white_space()?;

        let r = lex.peek();
        match r {
            ',' => {
                lex.accept(",");
            }
            '}' => {
                // continue — allows no trailing comma
            }
            _ => {
                return Err("expected ',' or '}'".to_string());
            }
        }
    }
}

fn read_lua_table_key(lex: &mut Lexer) -> Result<Key, String> {
    if !lex.accept("[") {
        return Err("expected '['".to_string());
    }

    lex.skip_white_space()?;
    let val = read_lua_value(lex)?;
    lex.skip_white_space()?;

    if !lex.accept("]") {
        return Err("expected ']'".to_string());
    }

    lex.skip_white_space()?;

    match val.value_type {
        ValueType::String => Ok(Key {
            key_type: KeyType::String,
            source: val.source,
            raw: match val.raw {
                RawValue::String(s) => RawKey::String(s),
                _ => unreachable!(),
            },
        }),
        ValueType::Int => Ok(Key {
            key_type: KeyType::Int,
            source: val.source,
            raw: match val.raw {
                RawValue::Int(i) => RawKey::Int(i),
                _ => unreachable!(),
            },
        }),
        ValueType::Bool => Ok(Key {
            key_type: KeyType::Bool,
            source: val.source,
            raw: match val.raw {
                RawValue::Bool(b) => RawKey::Bool(b),
                _ => unreachable!(),
            },
        }),
        ValueType::Float => Ok(Key {
            key_type: KeyType::Float,
            source: val.source,
            raw: match val.raw {
                RawValue::Float(f) => RawKey::Float(f),
                _ => unreachable!(),
            },
        }),
        _ => Err(format!(
            "unsupported value type for key: {:?}",
            val.value_type
        )),
    }
}

fn read_quoted_string_value(lex: &mut Lexer) -> Result<Value, String> {
    if !lex.accept("\"") {
        return Err("expected '\"'".to_string());
    }

    loop {
        match lex.next_rune() {
            '\\' => {
                let curr = lex.pos;
                lex.accept_run("\\");
                let num_escapes = (lex.pos - curr) + 1;

                if !num_escapes.is_multiple_of(2) && lex.peek() == '"' {
                    let _ = lex.next_rune();
                }
            }
            EOF | '\n' => {
                return Err("unterminated quoted string".to_string());
            }
            '"' => {
                let quoted_val = lex.take();
                let decoded = decode_lua_string(&quoted_val);
                let (val, was_transformed) = lex.config.transform_string(&decoded);

                return Ok(Value {
                    value_type: ValueType::String,
                    source: val.clone(),
                    raw: RawValue::String(val),
                    transformed: was_transformed,
                });
            }
            _ => {}
        }
    }
}

/// Decode a Lua quoted string (with surrounding quotes removed internally).
/// Handles escape sequences like \n, \t, \\, \", etc.
fn decode_lua_string(quoted: &str) -> String {
    // Remove surrounding quotes
    if quoted.len() < 2 {
        return String::new();
    }
    let inner = &quoted[1..quoted.len() - 1];

    let mut result = String::with_capacity(inner.len());
    let chars: Vec<char> = inner.chars().collect();
    let mut i = 0;

    while i < chars.len() {
        if chars[i] == '\\' && i + 1 < chars.len() {
            match chars[i + 1] {
                'n' => {
                    result.push('\n');
                    i += 2;
                }
                't' => {
                    result.push('\t');
                    i += 2;
                }
                'r' => {
                    result.push('\r');
                    i += 2;
                }
                '\\' => {
                    result.push('\\');
                    i += 2;
                }
                '"' => {
                    result.push('"');
                    i += 2;
                }
                'a' => {
                    result.push('\x07');
                    i += 2;
                }
                'b' => {
                    result.push('\x08');
                    i += 2;
                }
                'f' => {
                    result.push('\x0C');
                    i += 2;
                }
                'v' => {
                    result.push('\x0B');
                    i += 2;
                }
                _ => {
                    // Unknown escape — keep as-is (Go's strconv.Unquote fallback)
                    result.push(chars[i]);
                    result.push(chars[i + 1]);
                    i += 2;
                }
            }
        } else {
            result.push(chars[i]);
            i += 1;
        }
    }

    result
}

fn read_number_value(lex: &mut Lexer) -> Result<Value, String> {
    lex.accept("-");

    while lex.next_rune().is_ascii_digit() {}
    lex.backup();

    let mut is_int = true;
    if lex.accept(".") {
        is_int = false;
        while lex.next_rune().is_ascii_digit() {}
        lex.backup();
    }

    if lex.accept("eE") {
        is_int = false;
        lex.accept("+-");
        lex.accept_run("0123456789");
    }

    let num_str = lex.take();

    if is_int {
        let val: i64 = num_str.parse().map_err(|e| format!("invalid int: {}", e))?;
        Ok(Value {
            value_type: ValueType::Int,
            source: num_str,
            raw: RawValue::Int(val),
            transformed: false,
        })
    } else {
        let val: f64 = num_str
            .parse()
            .map_err(|e| format!("invalid float: {}", e))?;
        Ok(Value {
            value_type: ValueType::Float,
            source: num_str,
            raw: RawValue::Float(val),
            transformed: false,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_decode_lua_string_basic() {
        assert_eq!(decode_lua_string(r#""hello""#), "hello");
        assert_eq!(decode_lua_string(r#""hello\"world""#), "hello\"world");
        assert_eq!(decode_lua_string(r#""hello\\""#), "hello\\");
        assert_eq!(decode_lua_string(r#""hello\nworld""#), "hello\nworld");
        assert_eq!(decode_lua_string(r#""hello\tworld""#), "hello\tworld");
    }
}
