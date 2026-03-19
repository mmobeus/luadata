use luadata::options::*;
use luadata::text_to_json;

fn json(input: &str, config: ParseConfig) -> String {
    text_to_json("input", input, config).unwrap()
}

fn json_default(input: &str) -> String {
    json(input, ParseConfig::new())
}

// ========== Simple Values ==========

#[test]
fn test_simple_string() {
    assert_eq!(json_default(r#"foo="bar""#), r#"{"foo":"bar"}"#);
}

#[test]
fn test_simple_int() {
    assert_eq!(json_default("foo=42"), r#"{"foo":42}"#);
}

#[test]
fn test_simple_float() {
    assert_eq!(json_default("foo=3.14"), r#"{"foo":3.14}"#);
}

#[test]
fn test_simple_bool_true() {
    assert_eq!(json_default("foo=true"), r#"{"foo":true}"#);
}

#[test]
fn test_simple_bool_false() {
    assert_eq!(json_default("foo=false"), r#"{"foo":false}"#);
}

#[test]
fn test_simple_nil() {
    assert_eq!(json_default("foo=nil"), r#"{"foo":null}"#);
}

// ========== Multiple Variables ==========

#[test]
fn test_multiple_variables() {
    assert_eq!(json_default("a=1\nb=2\nc=3\n"), r#"{"a":1,"b":2,"c":3}"#);
}

// ========== Nested Tables ==========

#[test]
fn test_nested_table() {
    assert_eq!(
        json_default(r#"foo={[1]="a",[2]="b"}"#),
        r#"{"foo":["a","b"]}"#
    );
}

#[test]
fn test_table_with_bracketed_keys() {
    assert_eq!(
        json_default(r#"data={["name"]="test",["count"]=5}"#),
        r#"{"data":{"name":"test","count":5}}"#
    );
}

#[test]
fn test_implicit_index_keys() {
    assert_eq!(
        json_default(r#"arr={"a","b","c"}"#),
        r#"{"arr":["a","b","c"]}"#
    );
}

#[test]
fn test_deeply_nested_table() {
    assert_eq!(
        json_default(r#"root={["child"]={["grandchild"]={["value"]=42}}}"#),
        r#"{"root":{"child":{"grandchild":{"value":42}}}}"#
    );
}

#[test]
fn test_multiline_table() {
    let input = "data={\n[\"a\"]=1,\n[\"b\"]=2,\n[\"c\"]=3,\n}\n";
    assert_eq!(json_default(input), r#"{"data":{"a":1,"b":2,"c":3}}"#);
}

// ========== Whitespace Variations ==========

#[test]
fn test_spaces_around_equals() {
    assert_eq!(json_default("foo = 42"), r#"{"foo":42}"#);
}

#[test]
fn test_tabs_around_equals() {
    assert_eq!(json_default("foo\t=\t42"), r#"{"foo":42}"#);
}

#[test]
fn test_leading_whitespace() {
    assert_eq!(json_default("  foo=42"), r#"{"foo":42}"#);
}

#[test]
fn test_trailing_newline() {
    assert_eq!(json_default("foo=42\n"), r#"{"foo":42}"#);
}

#[test]
fn test_multiple_trailing_newlines() {
    assert_eq!(json_default("foo=42\n\n\n"), r#"{"foo":42}"#);
}

// ========== Comments ==========

#[test]
fn test_line_comment_before() {
    assert_eq!(json_default("-- comment\nfoo=1"), r#"{"foo":1}"#);
}

#[test]
fn test_line_comment_after() {
    assert_eq!(json_default("foo=1\n-- comment\n"), r#"{"foo":1}"#);
}

#[test]
fn test_inline_comment_between_vars() {
    assert_eq!(json_default("a=1\n-- middle\nb=2"), r#"{"a":1,"b":2}"#);
}

// ========== Empty Input ==========

#[test]
fn test_empty_input() {
    // Empty input should produce empty object
    let result = text_to_json("input", "", ParseConfig::new()).unwrap();
    // serde_json serializes empty Map as {}
    assert_eq!(result, "null");
}

#[test]
fn test_whitespace_only_input() {
    let result = text_to_json("input", "   \n\t\n  ", ParseConfig::new()).unwrap();
    assert_eq!(result, "null");
}

// ========== Errors ==========

#[test]
fn test_error_missing_value() {
    assert!(text_to_json("input", "foo=", ParseConfig::new()).is_err());
}

#[test]
fn test_error_missing_equals() {
    assert!(text_to_json("input", "foo 42", ParseConfig::new()).is_err());
}

#[test]
fn test_error_unterminated_string() {
    assert!(text_to_json("input", r#"foo="bar"#, ParseConfig::new()).is_err());
}

#[test]
fn test_error_unterminated_table() {
    assert!(text_to_json("input", "foo={", ParseConfig::new()).is_err());
}

// ========== Escape Sequences ==========

#[test]
fn test_escaped_quote_mid_string() {
    assert_eq!(
        json_default(r#"foo="hello\"world""#),
        r#"{"foo":"hello\"world"}"#
    );
}

#[test]
fn test_double_backslash_before_closing_quote() {
    assert_eq!(json_default(r#"foo="hello\\""#), r#"{"foo":"hello\\"}"#);
}

#[test]
fn test_triple_backslash_before_quote() {
    assert_eq!(
        json_default(r#"foo="hello\\\"world""#),
        r#"{"foo":"hello\\\"world"}"#
    );
}

#[test]
fn test_backslash_n_is_newline() {
    assert_eq!(
        json_default(r#"foo="hello\nworld""#),
        "{\"foo\":\"hello\\nworld\"}"
    );
}

#[test]
fn test_backslash_t_is_tab() {
    assert_eq!(
        json_default(r#"foo="hello\tworld""#),
        "{\"foo\":\"hello\\tworld\"}"
    );
}

#[test]
fn test_four_backslashes() {
    assert_eq!(json_default(r#"foo="hello\\\\""#), r#"{"foo":"hello\\\\"}"#);
}

#[test]
fn test_escaped_quote_at_start() {
    assert_eq!(json_default(r#"foo="\"hello""#), r#"{"foo":"\"hello"}"#);
}

#[test]
fn test_multiple_escaped_quotes() {
    assert_eq!(
        json_default(r#"foo="say \"hi\" ok""#),
        r#"{"foo":"say \"hi\" ok"}"#
    );
}

// ========== Raw Values ==========

#[test]
fn test_raw_table() {
    assert_eq!(
        json_default(r#"{["a"]=1,["b"]=2}"#),
        r#"{"@root":{"a":1,"b":2}}"#
    );
}

#[test]
fn test_raw_string() {
    assert_eq!(json_default(r#""hello""#), r#"{"@root":"hello"}"#);
}

#[test]
fn test_raw_int() {
    assert_eq!(json_default("42"), r#"{"@root":42}"#);
}

#[test]
fn test_raw_negative_int() {
    assert_eq!(json_default("-7"), r#"{"@root":-7}"#);
}

#[test]
fn test_raw_float() {
    assert_eq!(json_default("3.14"), r#"{"@root":3.14}"#);
}

#[test]
fn test_raw_bool_true() {
    assert_eq!(json_default("true"), r#"{"@root":true}"#);
}

#[test]
fn test_raw_bool_false() {
    assert_eq!(json_default("false"), r#"{"@root":false}"#);
}

#[test]
fn test_raw_nil() {
    assert_eq!(json_default("nil"), r#"{"@root":null}"#);
}

#[test]
fn test_raw_empty_table() {
    assert_eq!(json_default("{}"), r#"{"@root":null}"#);
}

#[test]
fn test_raw_array() {
    assert_eq!(
        json_default(r#"{"a","b","c"}"#),
        r#"{"@root":["a","b","c"]}"#
    );
}

#[test]
fn test_raw_with_leading_comment() {
    assert_eq!(json_default("-- comment\n42"), r#"{"@root":42}"#);
}

#[test]
fn test_raw_with_surrounding_whitespace() {
    assert_eq!(json_default("  42  "), r#"{"@root":42}"#);
}

// ========== Raw Value Errors ==========

#[test]
fn test_raw_error_trailing_content_after_int() {
    assert!(text_to_json("input", "42 foo", ParseConfig::new()).is_err());
}

#[test]
fn test_raw_error_trailing_content_after_table() {
    assert!(text_to_json("input", "{} foo=1", ParseConfig::new()).is_err());
}

// ========== Raw Value No Regression ==========

#[test]
fn test_identifier_starting_with_keyword() {
    assert_eq!(json_default("true_val=true"), r#"{"true_val":true}"#);
}

// ========== Negative Numbers ==========

#[test]
fn test_negative_number_top_level() {
    assert_eq!(json_default("foo=-7"), r#"{"foo":-7}"#);
}

#[test]
fn test_negative_number_in_table() {
    assert_eq!(
        json_default(r#"data={["val"]=-7}"#),
        r#"{"data":{"val":-7}}"#
    );
}

// ========== Table No Trailing Comma ==========

#[test]
fn test_table_no_trailing_comma() {
    assert_eq!(
        json_default(r#"foo={["a"]=1,["b"]=2}"#),
        r#"{"foo":{"a":1,"b":2}}"#
    );
}

// ========== Array Rendering ==========

#[test]
fn test_simple_implicit_array() {
    assert_eq!(
        json_default(r#"data={"foo","bar","baz"}"#),
        r#"{"data":["foo","bar","baz"]}"#
    );
}

#[test]
fn test_nested_implicit_array() {
    let input = "data={\n[\"list\"]={\"a\",\"b\",\"c\"},\n}";
    assert_eq!(json_default(input), r#"{"data":{"list":["a","b","c"]}}"#);
}

#[test]
fn test_explicit_integer_keys_sparse_array_default() {
    assert_eq!(
        json_default(r#"data={[1]="a",[3]="c"}"#),
        r#"{"data":["a",null,"c"]}"#
    );
}

#[test]
fn test_mixed_keys_render_as_map() {
    let input = "data={\n\"foo\",\n[\"name\"]=\"bar\",\n}";
    assert_eq!(json_default(input), r#"{"data":{"1":"foo","name":"bar"}}"#);
}

#[test]
fn test_preserves_insertion_order() {
    assert_eq!(
        json_default(r#"data={["z"]=1,["a"]=2,["m"]=3}"#),
        r#"{"data":{"z":1,"a":2,"m":3}}"#
    );
}

#[test]
fn test_top_level_key_order_preserved() {
    assert_eq!(json_default("z=1\na=2\nm=3"), r#"{"z":1,"a":2,"m":3}"#);
}

#[test]
fn test_nested_table_order_preserved() {
    assert_eq!(
        json_default(r#"outer={["inner"]={["z"]=1,["a"]=2}}"#),
        r#"{"outer":{"inner":{"z":1,"a":2}}}"#
    );
}

// ========== Array Detection Modes ==========

fn json_with_array_mode(input: &str, mode: ArrayMode) -> String {
    let mut config = ParseConfig::new();
    config.array_mode = Some(mode);
    text_to_json("input", input, config).unwrap()
}

#[test]
fn test_sparse_contiguous_int_keys() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[1]="a",[2]="b",[3]="c"}"#,
            ArrayMode::Sparse { max_gap: 0 }
        ),
        r#"{"data":["a","b","c"]}"#
    );
}

#[test]
fn test_sparse_within_gap() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[1]="a",[3]="c"}"#,
            ArrayMode::Sparse { max_gap: 1 }
        ),
        r#"{"data":["a",null,"c"]}"#
    );
}

#[test]
fn test_sparse_exceeds_gap() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[1]="a",[5]="e"}"#,
            ArrayMode::Sparse { max_gap: 1 }
        ),
        r#"{"data":{"1":"a","5":"e"}}"#
    );
}

#[test]
fn test_mixed_int_and_string_keys_stays_object() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[1]="a",["name"]="b"}"#,
            ArrayMode::Sparse { max_gap: 10 }
        ),
        r#"{"data":{"1":"a","name":"b"}}"#
    );
}

#[test]
fn test_nested_tables_propagate_option() {
    assert_eq!(
        json_with_array_mode(
            r#"data={["items"]={[1]="x",[2]="y"}}"#,
            ArrayMode::Sparse { max_gap: 0 }
        ),
        r#"{"data":{"items":["x","y"]}}"#
    );
}

#[test]
fn test_index_only_renders_int_keys_as_object() {
    assert_eq!(
        json_with_array_mode(r#"data={[1]="a",[2]="b"}"#, ArrayMode::IndexOnly),
        r#"{"data":{"1":"a","2":"b"}}"#
    );
}

#[test]
fn test_index_only_renders_implicit_index_as_array() {
    assert_eq!(
        json_with_array_mode(r#"data={"a","b","c"}"#, ArrayMode::IndexOnly),
        r#"{"data":["a","b","c"]}"#
    );
}

#[test]
fn test_none_mode_renders_everything_as_object() {
    assert_eq!(
        json_with_array_mode(r#"data={"a","b","c"}"#, ArrayMode::None),
        r#"{"data":{"1":"a","2":"b","3":"c"}}"#
    );
}

#[test]
fn test_none_mode_renders_int_keys_as_object() {
    assert_eq!(
        json_with_array_mode(r#"data={[1]="a",[2]="b"}"#, ArrayMode::None),
        r#"{"data":{"1":"a","2":"b"}}"#
    );
}

#[test]
fn test_implicit_index_unaffected_by_sparse_option() {
    assert_eq!(
        json_with_array_mode(r#"data={"a","b","c"}"#, ArrayMode::Sparse { max_gap: 0 }),
        r#"{"data":["a","b","c"]}"#
    );
}

#[test]
fn test_keys_not_starting_at_1_gap_0() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[2]="a",[3]="b"}"#,
            ArrayMode::Sparse { max_gap: 0 }
        ),
        r#"{"data":{"2":"a","3":"b"}}"#
    );
}

#[test]
fn test_keys_not_starting_at_1_gap_ok() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[2]="a",[3]="b"}"#,
            ArrayMode::Sparse { max_gap: 1 }
        ),
        r#"{"data":[null,"a","b"]}"#
    );
}

#[test]
fn test_single_element_array() {
    assert_eq!(
        json_with_array_mode(r#"data={[1]="only"}"#, ArrayMode::Sparse { max_gap: 0 }),
        r#"{"data":["only"]}"#
    );
}

#[test]
fn test_int_keys_with_nested_table_values() {
    assert_eq!(
        json_with_array_mode(
            r#"data={[1]={["name"]="a"},[2]={["name"]="b"}}"#,
            ArrayMode::Sparse { max_gap: 0 }
        ),
        r#"{"data":[{"name":"a"},{"name":"b"}]}"#
    );
}

// ========== Empty Table Modes ==========

fn json_with_empty_mode(input: &str, mode: EmptyTableMode) -> String {
    let mut config = ParseConfig::new();
    config.empty_table_mode = mode;
    text_to_json("input", input, config).unwrap()
}

#[test]
fn test_empty_table_default_null() {
    assert_eq!(json_default("foo={}"), r#"{"foo":null}"#);
}

#[test]
fn test_empty_table_null_inline() {
    assert_eq!(
        json_with_empty_mode("foo={}", EmptyTableMode::Null),
        r#"{"foo":null}"#
    );
}

#[test]
fn test_empty_table_null_whitespace() {
    assert_eq!(
        json_with_empty_mode("foo={\n}", EmptyTableMode::Null),
        r#"{"foo":null}"#
    );
}

#[test]
fn test_empty_table_omit_removes_key() {
    assert_eq!(
        json_with_empty_mode("foo={}\nbar=1", EmptyTableMode::Omit),
        r#"{"bar":1}"#
    );
}

#[test]
fn test_empty_table_omit_whitespace() {
    assert_eq!(
        json_with_empty_mode("foo={\n}\nbar=1", EmptyTableMode::Omit),
        r#"{"bar":1}"#
    );
}

#[test]
fn test_empty_table_omit_all_empty() {
    assert_eq!(
        json_with_empty_mode("foo={}", EmptyTableMode::Omit),
        r#"{}"#
    );
}

#[test]
fn test_empty_table_array_inline() {
    assert_eq!(
        json_with_empty_mode("foo={}", EmptyTableMode::Array),
        r#"{"foo":[]}"#
    );
}

#[test]
fn test_empty_table_array_whitespace() {
    assert_eq!(
        json_with_empty_mode("foo={\n}", EmptyTableMode::Array),
        r#"{"foo":[]}"#
    );
}

#[test]
fn test_empty_table_object_inline() {
    assert_eq!(
        json_with_empty_mode("foo={}", EmptyTableMode::Object),
        r#"{"foo":{}}"#
    );
}

#[test]
fn test_empty_table_object_whitespace() {
    assert_eq!(
        json_with_empty_mode("foo={\n}", EmptyTableMode::Object),
        r#"{"foo":{}}"#
    );
}

// Nested empty tables
#[test]
fn test_nested_empty_null() {
    assert_eq!(
        json_with_empty_mode(r#"data={["a"]={},["b"]=1}"#, EmptyTableMode::Null),
        r#"{"data":{"a":null,"b":1}}"#
    );
}

#[test]
fn test_nested_empty_omit() {
    assert_eq!(
        json_with_empty_mode(r#"data={["a"]={},["b"]=1}"#, EmptyTableMode::Omit),
        r#"{"data":{"b":1}}"#
    );
}

#[test]
fn test_nested_empty_array() {
    assert_eq!(
        json_with_empty_mode(r#"data={["a"]={},["b"]=1}"#, EmptyTableMode::Array),
        r#"{"data":{"a":[],"b":1}}"#
    );
}

#[test]
fn test_nested_empty_object() {
    assert_eq!(
        json_with_empty_mode(r#"data={["a"]={},["b"]=1}"#, EmptyTableMode::Object),
        r#"{"data":{"a":{},"b":1}}"#
    );
}

// Empty tables in arrays
#[test]
fn test_array_element_empty_null() {
    assert_eq!(
        json_with_empty_mode("data={{},1,2}", EmptyTableMode::Null),
        r#"{"data":[null,1,2]}"#
    );
}

#[test]
fn test_array_element_empty_omit() {
    assert_eq!(
        json_with_empty_mode("data={{},1,2}", EmptyTableMode::Omit),
        r#"{"data":[1,2]}"#
    );
}

#[test]
fn test_array_element_empty_array() {
    assert_eq!(
        json_with_empty_mode("data={{},1,2}", EmptyTableMode::Array),
        r#"{"data":[[],1,2]}"#
    );
}

#[test]
fn test_array_element_empty_object() {
    assert_eq!(
        json_with_empty_mode("data={{},1,2}", EmptyTableMode::Object),
        r#"{"data":[{},1,2]}"#
    );
}

// Raw empty table
#[test]
fn test_raw_empty_table_omit_falls_back_to_null() {
    assert_eq!(
        json_with_empty_mode("{}", EmptyTableMode::Omit),
        r#"{"@root":null}"#
    );
}

#[test]
fn test_raw_empty_table_null() {
    assert_eq!(
        json_with_empty_mode("{}", EmptyTableMode::Null),
        r#"{"@root":null}"#
    );
}

#[test]
fn test_raw_empty_table_array() {
    assert_eq!(
        json_with_empty_mode("{}", EmptyTableMode::Array),
        r#"{"@root":[]}"#
    );
}

#[test]
fn test_raw_empty_table_object() {
    assert_eq!(
        json_with_empty_mode("{}", EmptyTableMode::Object),
        r#"{"@root":{}}"#
    );
}

// ========== String Transform ==========

#[test]
fn test_string_transform_truncate() {
    let long_str = "x".repeat(100);
    let input = format!(r#"long="{}""#, long_str);
    let mut config = ParseConfig::new();
    config.string_transform = Some(StringTransform {
        max_len: 10,
        mode: StringTransformMode::Truncate,
        replacement: String::new(),
    });
    let result = text_to_json("input", &input, config).unwrap();
    assert_eq!(result, r#"{"long":"xxxxxxxxxx"}"#);
}

#[test]
fn test_string_transform_empty() {
    let long_str = "x".repeat(100);
    let input = format!(r#"long="{}""#, long_str);
    let mut config = ParseConfig::new();
    config.string_transform = Some(StringTransform {
        max_len: 10,
        mode: StringTransformMode::Empty,
        replacement: String::new(),
    });
    let result = text_to_json("input", &input, config).unwrap();
    assert_eq!(result, r#"{"long":""}"#);
}

#[test]
fn test_string_transform_redact() {
    let long_str = "x".repeat(100);
    let input = format!(r#"long="{}""#, long_str);
    let mut config = ParseConfig::new();
    config.string_transform = Some(StringTransform {
        max_len: 10,
        mode: StringTransformMode::Redact,
        replacement: String::new(),
    });
    let result = text_to_json("input", &input, config).unwrap();
    assert_eq!(result, r#"{"long":"[redacted]"}"#);
}

#[test]
fn test_string_transform_replace() {
    let long_str = "x".repeat(100);
    let input = format!(r#"long="{}""#, long_str);
    let mut config = ParseConfig::new();
    config.string_transform = Some(StringTransform {
        max_len: 10,
        mode: StringTransformMode::Replace,
        replacement: "[too long]".to_string(),
    });
    let result = text_to_json("input", &input, config).unwrap();
    assert_eq!(result, r#"{"long":"[too long]"}"#);
}

#[test]
fn test_string_transform_under_limit_unchanged() {
    let mut config = ParseConfig::new();
    config.string_transform = Some(StringTransform {
        max_len: 10,
        mode: StringTransformMode::Redact,
        replacement: String::new(),
    });
    let result = text_to_json("input", r#"short="short""#, config).unwrap();
    assert_eq!(result, r#"{"short":"short"}"#);
}

// ========== Non-UTF-8 byte handling ==========

#[test]
fn test_non_utf8_bytes_preserved_losslessly() {
    // Simulate a Lua string containing raw binary data (like Questie's objPtrs).
    // Byte 0x9E is not valid UTF-8 on its own.
    let input: &[u8] = b"data=\"hello\x9eworld\"";
    let result = luadata::to_json(input, ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    let s = v["data"].as_str().unwrap();
    // Byte 0x9E should be preserved as U+009E (Latin-1 code point), not U+FFFD.
    assert!(s.contains('\u{009e}'), "expected U+009E, got: {:?}", s);
    assert!(
        !s.contains('\u{fffd}'),
        "should not contain replacement character"
    );
    assert_eq!(s, "hello\u{009e}world");
}

#[test]
fn test_valid_utf8_unchanged() {
    // Valid UTF-8 multi-byte sequences must pass through unchanged.
    let input = "data=\"\u{4e16}\u{754c}\""; // 世界
    let result = text_to_json("input", input, ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["data"], "世界");
}

#[test]
fn test_mixed_utf8_and_binary() {
    // Mix of valid UTF-8 and raw binary bytes. Because the string contains
    // 0x9E (invalid UTF-8), the whole string falls back to Latin-1 byte mapping.
    // So 0xC3 0xA9 (UTF-8 for é) becomes two Latin-1 chars: Ã (U+00C3) © (U+00A9).
    let mut input = Vec::from(&b"data=\"valid\xc3\xa9"[..]);
    input.extend_from_slice(b"\x9e");
    input.extend_from_slice(b"end\"");
    let result = luadata::to_json(&input, ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    let s = v["data"].as_str().unwrap();
    assert!(s.starts_with("valid\u{00c3}\u{00a9}"));
    assert!(s.contains('\u{009e}'));
    assert!(s.ends_with("end"));
}

#[test]
fn test_valid_utf8_string_via_bytes() {
    // A file with a valid UTF-8 string: player name "Fröst".
    // The whole string is valid UTF-8, so it decodes as UTF-8.
    let input = b"name=\"Fr\xc3\xb6st\"";
    let result = luadata::to_json(input, ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["name"], "Fröst");
}

#[test]
fn test_binary_blob_bytes_preserved() {
    // Binary blob: bytes 0xC4 0xB6 appear but the string also has 0x9E,
    // making it invalid UTF-8. All bytes map to Latin-1 code points.
    // 0xC4 → U+00C4 (Ä), 0xB6 → U+00B6 (¶), NOT U+0136 (Ķ).
    let input = b"data=\"\x02\x9e\xc4\xb6\x02\"";
    let result = luadata::to_json(input, ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    let s = v["data"].as_str().unwrap();
    let chars: Vec<u32> = s.chars().map(|c| c as u32).collect();
    assert_eq!(chars, vec![0x02, 0x9e, 0xc4, 0xb6, 0x02]);
}

// ========== File-based tests ==========

#[test]
fn test_file_simple() {
    let result = luadata::file_to_json("testdata/valid/simple.lua", ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["name"], "hello");
    assert_eq!(v["count"], 42);
    assert_eq!(v["enabled"], true);
}

#[test]
fn test_file_array() {
    let result = luadata::file_to_json("testdata/valid/array.lua", ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["items"], serde_json::json!(["apple", "banana", "cherry"]));
}

#[test]
fn test_file_nested() {
    let result = luadata::file_to_json("testdata/valid/nested.lua", ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["config"]["host"], "localhost");
    assert_eq!(v["config"]["port"], 8080);
    assert_eq!(v["config"]["options"]["verbose"], true);
    assert_eq!(v["config"]["options"]["retries"], 3);
}

#[test]
fn test_file_comments() {
    let result = luadata::file_to_json("testdata/valid/comments.lua", ParseConfig::new()).unwrap();
    let v: serde_json::Value = serde_json::from_str(&result).unwrap();
    assert_eq!(v["name"], "test");
    assert_eq!(v["data"]["a"], 1);
    assert_eq!(v["data"]["b"], 2);
}
