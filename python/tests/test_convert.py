"""Tests for the luadata Python module."""

import json
import os
from pathlib import Path

import pytest

from luadata import lua_to_dict, lua_to_json

TESTDATA = os.path.join(os.path.dirname(__file__), "..", "..", "testdata")


class TestLuaToJson:
    def test_simple_assignment(self):
        result = lua_to_json('x = 42')
        assert json.loads(result) == {"x": 42}

    def test_returns_string(self):
        result = lua_to_json('x = 1')
        assert isinstance(result, str)

    def test_string_value(self):
        result = lua_to_json('foo = "bar"')
        assert json.loads(result) == {"foo": "bar"}

    def test_bool_values(self):
        result = lua_to_json("a = true\nb = false")
        parsed = json.loads(result)
        assert parsed == {"a": True, "b": False}

    def test_nil_value(self):
        result = lua_to_json("x = nil")
        assert json.loads(result) == {"x": None}

    def test_float_value(self):
        result = lua_to_json("x = 3.14")
        parsed = json.loads(result)
        assert parsed["x"] == pytest.approx(3.14)

    def test_negative_number(self):
        result = lua_to_json("x = -10")
        assert json.loads(result) == {"x": -10}


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

    def test_deeply_nested(self):
        lua = 'root = {["child"] = {["grandchild"] = {["value"] = 42}}}'
        result = lua_to_dict(lua)
        assert result["root"]["child"]["grandchild"]["value"] == 42

    def test_nil_value(self):
        result = lua_to_dict("x = nil")
        assert result == {"x": None}

    def test_float_value(self):
        result = lua_to_dict("x = 3.14")
        assert result["x"] == pytest.approx(3.14)

    def test_negative_number(self):
        result = lua_to_dict("x = -10")
        assert result == {"x": -10}


class TestRawValues:
    def test_raw_table(self):
        result = lua_to_dict('{["a"] = 1, ["b"] = 2}')
        assert result == {"@root": {"a": 1, "b": 2}}

    def test_raw_string(self):
        result = lua_to_dict('"hello"')
        assert result == {"@root": "hello"}

    def test_raw_int(self):
        result = lua_to_dict("42")
        assert result == {"@root": 42}

    def test_raw_bool(self):
        result = lua_to_dict("true")
        assert result == {"@root": True}

    def test_raw_array(self):
        result = lua_to_dict('{"a", "b", "c"}')
        assert result == {"@root": ["a", "b", "c"]}


class TestInsertionOrder:
    def test_top_level_order(self):
        result = lua_to_json("z = 1\na = 2\nm = 3")
        # JSON keys should preserve insertion order
        keys = list(json.loads(result).keys())
        assert keys == ["z", "a", "m"]

    def test_table_key_order(self):
        result = lua_to_json('data = {["z"] = 1, ["a"] = 2, ["m"] = 3}')
        keys = list(json.loads(result)["data"].keys())
        assert keys == ["z", "a", "m"]


class TestComments:
    def test_line_comment_before(self):
        result = lua_to_dict("-- comment\nfoo = 1")
        assert result == {"foo": 1}

    def test_line_comment_after(self):
        result = lua_to_dict("foo = 1\n-- comment\n")
        assert result == {"foo": 1}

    def test_comment_between_vars(self):
        result = lua_to_dict("a = 1\n-- middle\nb = 2")
        assert result == {"a": 1, "b": 2}


class TestEscapeSequences:
    def test_escaped_quote(self):
        result = lua_to_dict(r'foo = "hello\"world"')
        assert result == {"foo": 'hello"world'}

    def test_backslash(self):
        result = lua_to_dict(r'foo = "hello\\"')
        assert result == {"foo": "hello\\"}

    def test_newline(self):
        result = lua_to_dict(r'foo = "hello\nworld"')
        assert result == {"foo": "hello\nworld"}

    def test_tab(self):
        result = lua_to_dict(r'foo = "hello\tworld"')
        assert result == {"foo": "hello\tworld"}


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

    def test_nested_omit(self):
        result = lua_to_dict('data = {["a"] = {}, ["b"] = 1}', empty_table="omit")
        assert result == {"data": {"b": 1}}

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

    def test_index_only_explicit_int_stays_object(self):
        result = lua_to_dict('x = {[1] = "a", [2] = "b"}', array_mode="index-only")
        assert result == {"x": {"1": "a", "2": "b"}}

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


class TestUTF8:
    def test_cjk(self):
        result = lua_to_dict('name = "你好世界"')
        assert result == {"name": "你好世界"}

    def test_emoji(self):
        result = lua_to_dict('icon = "🎮🗡️🛡️"')
        assert result == {"icon": "🎮🗡️🛡️"}

    def test_japanese(self):
        result = lua_to_dict('msg = "こんにちは"')
        assert result == {"msg": "こんにちは"}

    def test_korean(self):
        result = lua_to_dict('greeting = "안녕하세요"')
        assert result == {"greeting": "안녕하세요"}

    def test_cyrillic(self):
        result = lua_to_dict('text = "Привет мир"')
        assert result == {"text": "Привет мир"}

    def test_accented(self):
        result = lua_to_dict('café = "résumé"')
        assert result == {"café": "résumé"}

    def test_utf8_in_table_keys(self):
        result = lua_to_dict('data = {["名前"] = "test"}')
        assert result == {"data": {"名前": "test"}}

    def test_utf8_in_array(self):
        result = lua_to_dict('items = {"剑", "盾", "弓"}')
        assert result == {"items": ["剑", "盾", "弓"]}

    def test_multibyte_with_escape(self):
        result = lua_to_dict(r'msg = "line1\nこんにちは"')
        assert result == {"msg": "line1\nこんにちは"}

    def test_json_output_valid_utf8(self):
        result = lua_to_json('name = "你好世界"')
        parsed = json.loads(result)
        assert parsed["name"] == "你好世界"


class TestErrors:
    def test_parse_error_raises(self):
        with pytest.raises(ValueError):
            lua_to_dict("{{{invalid}}}")

    def test_unterminated_string(self):
        with pytest.raises(ValueError):
            lua_to_dict('foo = "bar')

    def test_missing_value(self):
        with pytest.raises(ValueError):
            lua_to_dict("foo =")

    def test_invalid_empty_table_option(self):
        with pytest.raises(ValueError, match="unknown empty_table"):
            lua_to_dict("x = {}", empty_table="bad")

    def test_invalid_array_mode_option(self):
        with pytest.raises(ValueError, match="unknown array_mode"):
            lua_to_dict("x = {}", array_mode="bad")

    def test_invalid_string_mode_option(self):
        with pytest.raises(ValueError, match="unknown string_mode"):
            lua_to_dict('x = "hello"', string_max_len=3, string_mode="bad")

    def test_json_variant_raises(self):
        with pytest.raises(ValueError):
            lua_to_json("{{{invalid}}}")


class TestFileTestdata:
    def test_simple(self):
        path = Path(TESTDATA) / "valid" / "simple.lua"
        text = path.read_text(encoding="utf-8")
        result = lua_to_dict(text)
        assert result == {"name": "hello", "count": 42, "enabled": True}

    def test_array(self):
        path = Path(TESTDATA) / "valid" / "array.lua"
        text = path.read_text(encoding="utf-8")
        result = lua_to_dict(text)
        assert result == {"items": ["apple", "banana", "cherry"]}

    def test_nested(self):
        path = Path(TESTDATA) / "valid" / "nested.lua"
        text = path.read_text(encoding="utf-8")
        result = lua_to_dict(text)
        assert result["config"]["host"] == "localhost"
        assert result["config"]["port"] == 8080
        assert result["config"]["options"]["verbose"] is True
        assert result["config"]["options"]["retries"] == 3

    def test_comments(self):
        path = Path(TESTDATA) / "valid" / "comments.lua"
        text = path.read_text(encoding="utf-8")
        result = lua_to_dict(text)
        assert result["name"] == "test"
        assert result["data"]["a"] == 1
        assert result["data"]["b"] == 2
