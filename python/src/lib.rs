use pyo3::exceptions::PyValueError;
use pyo3::prelude::*;

use ::luadata::options::{
    ArrayMode, EmptyTableMode, ParseConfig, StringTransform, StringTransformMode,
};

/// Convert Lua data to a JSON string.
///
/// Args:
///     text: Lua data as a string.
///     empty_table: How to render empty tables ("null", "omit", "array", "object").
///     array_mode: Array detection mode ("none", "index-only", "sparse").
///     array_max_gap: Max gap for sparse array mode.
///     string_max_len: Max string length before transform.
///     string_mode: Transform mode ("truncate", "empty", "redact", "replace").
///     string_replacement: Custom replacement string (for "replace" mode).
///
/// Returns:
///     JSON string.
///
/// Raises:
///     ValueError: If the Lua input cannot be parsed or options are invalid.
#[pyfunction]
#[pyo3(signature = (
    text,
    *,
    empty_table=None,
    array_mode=None,
    array_max_gap=None,
    string_max_len=None,
    string_mode=None,
    string_replacement=None,
))]
fn lua_to_json(
    text: &str,
    empty_table: Option<&str>,
    array_mode: Option<&str>,
    array_max_gap: Option<usize>,
    string_max_len: Option<usize>,
    string_mode: Option<&str>,
    string_replacement: Option<&str>,
) -> PyResult<String> {
    let config = build_config(
        empty_table,
        array_mode,
        array_max_gap,
        string_max_len,
        string_mode,
        string_replacement,
    )?;

    ::luadata::text_to_json("input", text, config).map_err(PyValueError::new_err)
}

/// Convert Lua data to a Python dict.
///
/// Args:
///     text: Lua data as a string.
///     empty_table: How to render empty tables ("null", "omit", "array", "object").
///     array_mode: Array detection mode ("none", "index-only", "sparse").
///     array_max_gap: Max gap for sparse array mode.
///     string_max_len: Max string length before transform.
///     string_mode: Transform mode ("truncate", "empty", "redact", "replace").
///     string_replacement: Custom replacement string (for "replace" mode).
///
/// Returns:
///     Python dict, or None for empty input.
///
/// Raises:
///     ValueError: If the Lua input cannot be parsed or options are invalid.
#[pyfunction]
#[pyo3(signature = (
    text,
    *,
    empty_table=None,
    array_mode=None,
    array_max_gap=None,
    string_max_len=None,
    string_mode=None,
    string_replacement=None,
))]
#[allow(clippy::too_many_arguments)]
fn lua_to_dict<'py>(
    py: Python<'py>,
    text: &str,
    empty_table: Option<&str>,
    array_mode: Option<&str>,
    array_max_gap: Option<usize>,
    string_max_len: Option<usize>,
    string_mode: Option<&str>,
    string_replacement: Option<&str>,
) -> PyResult<Bound<'py, PyAny>> {
    let json_str = lua_to_json(
        text,
        empty_table,
        array_mode,
        array_max_gap,
        string_max_len,
        string_mode,
        string_replacement,
    )?;

    // Use Python's json.loads to convert JSON string to Python objects
    let json_mod = py.import("json")?;
    let result = json_mod.call_method1("loads", (json_str,))?;
    Ok(result)
}

fn build_config(
    empty_table: Option<&str>,
    array_mode: Option<&str>,
    array_max_gap: Option<usize>,
    string_max_len: Option<usize>,
    string_mode: Option<&str>,
    string_replacement: Option<&str>,
) -> PyResult<ParseConfig> {
    let mut config = ParseConfig::new();

    if let Some(et) = empty_table {
        config.empty_table_mode = match et {
            "null" => EmptyTableMode::Null,
            "omit" => EmptyTableMode::Omit,
            "array" => EmptyTableMode::Array,
            "object" => EmptyTableMode::Object,
            _ => {
                return Err(PyValueError::new_err(format!(
                    "unknown empty_table value: {:?}",
                    et
                )));
            }
        };
    }

    if let Some(am) = array_mode {
        config.array_mode = Some(match am {
            "none" => ArrayMode::None,
            "index-only" => ArrayMode::IndexOnly,
            "sparse" => ArrayMode::Sparse {
                max_gap: array_max_gap.unwrap_or(20),
            },
            _ => {
                return Err(PyValueError::new_err(format!(
                    "unknown array_mode value: {:?}",
                    am
                )));
            }
        });
    }

    if let Some(max_len) = string_max_len {
        if max_len == 0 {
            return Err(PyValueError::new_err(
                "string_max_len must be a positive number",
            ));
        }

        let mode = match string_mode.unwrap_or("truncate") {
            "truncate" => StringTransformMode::Truncate,
            "empty" => StringTransformMode::Empty,
            "redact" => StringTransformMode::Redact,
            "replace" => StringTransformMode::Replace,
            m => {
                return Err(PyValueError::new_err(format!(
                    "unknown string_mode value: {:?}",
                    m
                )));
            }
        };

        config.string_transform = Some(StringTransform {
            max_len,
            mode,
            replacement: string_replacement.unwrap_or("").to_string(),
        });
    }

    Ok(config)
}

/// Native Python module for converting Lua data files to JSON/Python objects.
#[pymodule]
fn luadata(m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add_function(wrap_pyfunction!(lua_to_json, m)?)?;
    m.add_function(wrap_pyfunction!(lua_to_dict, m)?)?;
    Ok(())
}
