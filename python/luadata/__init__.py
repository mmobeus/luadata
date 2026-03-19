"""luadata - Convert Lua data files to JSON."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Literal

from ._binding import call_lua_data_to_json

__all__ = [
    "lua_to_json",
    "lua_to_dict",
    "LuaDataError",
]


class LuaDataError(Exception):
    """Raised when the Go library returns a parse or conversion error."""


def lua_to_json(
    input: str | Path,
    *,
    empty_table: Literal["null", "omit", "array", "object"] | None = None,
    array_mode: Literal["none", "index-only", "sparse"] | None = None,
    array_max_gap: int | None = None,
    string_max_len: int | None = None,
    string_mode: Literal["truncate", "empty", "redact", "replace"] | None = None,
    string_replacement: str | None = None,
) -> str:
    """Convert Lua data to a JSON string.

    Args:
        input: Lua data as a string, or a path to a Lua data file.
        empty_table: How to render empty tables. Default: "null".
        array_mode: Array detection mode. Default: "sparse".
        array_max_gap: Max gap for sparse array mode. Default: 20.
        string_max_len: Max string length before transform is applied.
        string_mode: How to handle strings exceeding max_len. Default: "truncate".
        string_replacement: Custom replacement string (only for mode="replace").

    Returns:
        JSON string of the converted Lua data.

    Raises:
        LuaDataError: If the Lua input cannot be parsed.
    """
    text = _read_input(input)
    return _call(text, **_build_options(
        empty_table=empty_table,
        array_mode=array_mode,
        array_max_gap=array_max_gap,
        string_max_len=string_max_len,
        string_mode=string_mode,
        string_replacement=string_replacement,
    ))


def lua_to_dict(
    input: str | Path,
    *,
    empty_table: Literal["null", "omit", "array", "object"] | None = None,
    array_mode: Literal["none", "index-only", "sparse"] | None = None,
    array_max_gap: int | None = None,
    string_max_len: int | None = None,
    string_mode: Literal["truncate", "empty", "redact", "replace"] | None = None,
    string_replacement: str | None = None,
) -> dict[str, Any] | None:
    """Convert Lua data to a Python dict.

    Args:
        input: Lua data as a string, or a path to a Lua data file.
        empty_table: How to render empty tables. Default: "null".
        array_mode: Array detection mode. Default: "sparse".
        array_max_gap: Max gap for sparse array mode. Default: 20.
        string_max_len: Max string length before transform is applied.
        string_mode: How to handle strings exceeding max_len. Default: "truncate".
        string_replacement: Custom replacement string (only for mode="replace").

    Returns:
        A dict of the converted Lua data, or None for empty input.

    Raises:
        LuaDataError: If the Lua input cannot be parsed.
    """
    result = lua_to_json(input, **{
        k: v for k, v in dict(
            empty_table=empty_table,
            array_mode=array_mode,
            array_max_gap=array_max_gap,
            string_max_len=string_max_len,
            string_mode=string_mode,
            string_replacement=string_replacement,
        ).items() if v is not None
    })
    return json.loads(result)


def _read_input(input: str | Path) -> str:
    """Resolve input to a Lua data string. If it's a Path, read the file."""
    if isinstance(input, Path):
        return input.read_text(encoding="utf-8")
    return input


def _call(text: str, **opts) -> str:
    """Call the Go library and return the JSON result string."""
    options_json = json.dumps(opts) if opts else ""
    envelope = call_lua_data_to_json(text, options_json)
    parsed = json.loads(envelope)

    if "error" in parsed:
        raise LuaDataError(parsed["error"])

    return parsed["result"]


def _build_options(**kwargs) -> dict:
    opts = {}

    if kwargs.get("empty_table") is not None:
        opts["empty_table"] = kwargs["empty_table"]

    if kwargs.get("array_mode") is not None:
        opts["array_mode"] = kwargs["array_mode"]

    if kwargs.get("array_max_gap") is not None:
        opts["array_max_gap"] = kwargs["array_max_gap"]

    st_max_len = kwargs.get("string_max_len")
    if st_max_len is not None:
        st = {"max_len": st_max_len}
        if kwargs.get("string_mode") is not None:
            st["mode"] = kwargs["string_mode"]
        if kwargs.get("string_replacement") is not None:
            st["replacement"] = kwargs["string_replacement"]
        opts["string_transform"] = st

    return opts
