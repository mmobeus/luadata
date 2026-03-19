"""Tests for the luadata Python wrapper."""

import json
import os
from pathlib import Path

import pytest

from luadata import LuaDataError, lua_to_dict, lua_to_json

TESTDATA = os.path.join(os.path.dirname(__file__), "..", "..", "testdata")


class TestLuaToJson:
    def test_simple_assignment(self):
        result = lua_to_json('x = 42')
        assert json.loads(result) == {"x": 42}

    def test_returns_string(self):
        result = lua_to_json('x = 1')
        assert isinstance(result, str)

    def test_from_path(self):
        result = lua_to_json(Path(os.path.join(TESTDATA, "valid", "simple.lua")))
        assert json.loads(result) == {"name": "hello", "count": 42, "enabled": True}

    def test_from_pathlib(self):
        result = lua_to_json(Path(TESTDATA) / "valid" / "simple.lua")
        assert json.loads(result) == {"name": "hello", "count": 42, "enabled": True}


class TestLuaToDict:
    def test_simple_assignment(self):
        result = lua_to_dict('x = 42')
        assert result == {"x": 42}

    def test_returns_dict(self):
        result = lua_to_dict('x = 1')
        assert isinstance(result, dict)

    def test_multiple_assignments(self):
        result = lua_to_dict('name = "hello"\ncount = 42\nenabled = true')
        assert result == {"name": "hello", "count": 42, "enabled": True}

    def test_implicit_array(self):
        result = lua_to_dict('items = {"apple", "banana", "cherry"}')
        assert result == {"items": ["apple", "banana", "cherry"]}

    def test_nested_table(self):
        lua = 'config = {["host"] = "localhost", ["port"] = 8080}'
        result = lua_to_dict(lua)
        assert result == {"config": {"host": "localhost", "port": 8080}}

    def test_nil_value(self):
        result = lua_to_dict("x = nil")
        assert result == {"x": None}

    def test_float_value(self):
        result = lua_to_dict("x = 3.14")
        assert result["x"] == pytest.approx(3.14)

    def test_negative_number(self):
        result = lua_to_dict("x = -10")
        assert result == {"x": -10}

    def test_empty_input(self):
        result = lua_to_dict("")
        assert result is None

    def test_from_path(self):
        result = lua_to_dict(Path(os.path.join(TESTDATA, "valid", "simple.lua")))
        assert result == {"name": "hello", "count": 42, "enabled": True}

    def test_from_pathlib(self):
        result = lua_to_dict(Path(TESTDATA) / "valid" / "array.lua")
        assert result == {"items": ["apple", "banana", "cherry"]}

    def test_nested_file(self):
        result = lua_to_dict(Path(TESTDATA) / "valid" / "nested.lua")
        assert result["config"]["host"] == "localhost"
        assert result["config"]["port"] == 8080
        assert result["config"]["options"]["verbose"] is True

    def test_file_not_found(self):
        with pytest.raises(FileNotFoundError):
            lua_to_dict(Path("/nonexistent/file.lua"))


class TestEmptyTableOption:
    def test_null(self):
        result = lua_to_dict("x = {}", empty_table="null")
        assert result == {"x": None}

    def test_omit(self):
        result = lua_to_dict("x = {}\ny = 1", empty_table="omit")
        assert result == {"y": 1}

    def test_array(self):
        result = lua_to_dict("x = {}", empty_table="array")
        assert result == {"x": []}

    def test_object(self):
        result = lua_to_dict("x = {}", empty_table="object")
        assert result == {"x": {}}

    def test_json_variant(self):
        result = lua_to_json("x = {}", empty_table="array")
        assert json.loads(result) == {"x": []}


class TestArrayModeOption:
    def test_sparse(self):
        result = lua_to_dict('x = {"a", "b", "c"}', array_mode="sparse")
        assert result == {"x": ["a", "b", "c"]}

    def test_none(self):
        result = lua_to_dict('x = {"a", "b", "c"}', array_mode="none")
        assert result == {"x": {"1": "a", "2": "b", "3": "c"}}

    def test_index_only(self):
        result = lua_to_dict('x = {"a", "b", "c"}', array_mode="index-only")
        assert result == {"x": ["a", "b", "c"]}

    def test_sparse_with_max_gap(self):
        result = lua_to_dict(
            'x = {[1] = "a", [5] = "b"}',
            array_mode="sparse",
            array_max_gap=10,
        )
        assert isinstance(result["x"], list)

    def test_sparse_gap_exceeded(self):
        result = lua_to_dict(
            'x = {[1] = "a", [100] = "b"}',
            array_mode="sparse",
            array_max_gap=2,
        )
        assert isinstance(result["x"], dict)


class TestStringTransformOption:
    def test_truncate(self):
        result = lua_to_dict('x = "hello world"', string_max_len=5, string_mode="truncate")
        assert result == {"x": "hello"}

    def test_empty(self):
        result = lua_to_dict('x = "hello world"', string_max_len=5, string_mode="empty")
        assert result == {"x": ""}

    def test_redact(self):
        result = lua_to_dict('x = "hello world"', string_max_len=5, string_mode="redact")
        assert result == {"x": "[redacted]"}

    def test_replace(self):
        result = lua_to_dict(
            'x = "hello world"',
            string_max_len=5,
            string_mode="replace",
            string_replacement="***",
        )
        assert result == {"x": "***"}

    def test_short_string_not_transformed(self):
        result = lua_to_dict('x = "hi"', string_max_len=5, string_mode="truncate")
        assert result == {"x": "hi"}


class TestErrors:
    def test_parse_error_raises(self):
        with pytest.raises(LuaDataError):
            lua_to_dict("{{{invalid}}}")

    def test_invalid_empty_table_option(self):
        with pytest.raises(LuaDataError, match="unknown empty_table"):
            lua_to_dict("x = {}", empty_table="bad")

    def test_invalid_array_mode_option(self):
        with pytest.raises(LuaDataError, match="unknown array_mode"):
            lua_to_dict("x = {}", array_mode="bad")

    def test_invalid_string_mode_option(self):
        with pytest.raises(LuaDataError, match="unknown string_transform.mode"):
            lua_to_dict('x = "hello"', string_max_len=3, string_mode="bad")

    def test_json_variant_raises(self):
        with pytest.raises(LuaDataError):
            lua_to_json("{{{invalid}}}")
