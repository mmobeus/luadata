package luadata

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func readJSON(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	return string(b)
}

func TestTextToJSON_SimpleValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"string", `foo="bar"`, `{"foo":"bar"}`},
		{"int", "foo=42", `{"foo":42}`},
		{"float", "foo=3.14", `{"foo":3.14}`},
		{"bool true", "foo=true", `{"foo":true}`},
		{"bool false", "foo=false", `{"foo":false}`},
		{"nil", "foo=nil", `{"foo":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestTextToJSON_MultipleVariables(t *testing.T) {
	r, err := TextToJSON("input", "a=1\nb=2\nc=3\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readJSON(t, r)
	if got != `{"a":1,"b":2,"c":3}` {
		t.Errorf("expected {\"a\":1,\"b\":2,\"c\":3}, got %s", got)
	}
}

func TestTextToJSON_NestedTable(t *testing.T) {
	r, err := TextToJSON("input", `foo={[1]="a",[2]="b"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readJSON(t, r)
	if got != `{"foo":["a","b"]}` {
		t.Errorf("expected {\"foo\":[\"a\",\"b\"]}, got %s", got)
	}
}

func TestTextToJSON_ImplicitArray(t *testing.T) {
	r, err := TextToJSON("input", `arr={"a","b","c"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readJSON(t, r)
	if got != `{"arr":["a","b","c"]}` {
		t.Errorf("expected {\"arr\":[\"a\",\"b\",\"c\"]}, got %s", got)
	}
}

func TestTextToJSON_DeeplyNested(t *testing.T) {
	r, err := TextToJSON("input", `root={["child"]={["grandchild"]={["value"]=42}}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readJSON(t, r)
	if got != `{"root":{"child":{"grandchild":{"value":42}}}}` {
		t.Errorf("unexpected: %s", got)
	}
}

func TestTextToJSON_Comments(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"line comment before", "-- comment\nfoo=1"},
		{"line comment after", "foo=1\n-- comment\n"},
		{"inline comment between vars", "a=1\n-- middle\nb=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTextToJSON_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"missing value", "foo="},
		{"missing equals", "foo 42"},
		{"unterminated string", `foo="bar`},
		{"unterminated table", "foo={"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TextToJSON("input", tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestTextToJSON_EscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escaped quote",
			input:    `foo="hello\"world"`,
			expected: `{"foo":"hello\"world"}`,
		},
		{
			name:     "double backslash",
			input:    `foo="hello\\"`,
			expected: `{"foo":"hello\\"}`,
		},
		{
			name:     "backslash n",
			input:    `foo="hello\nworld"`,
			expected: "{\"foo\":\"hello\\nworld\"}",
		},
		{
			name:     "backslash t",
			input:    `foo="hello\tworld"`,
			expected: "{\"foo\":\"hello\\tworld\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestTextToJSON_RawValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"raw table", `{["a"]=1,["b"]=2}`, `{"@root":{"a":1,"b":2}}`},
		{"raw string", `"hello"`, `{"@root":"hello"}`},
		{"raw int", "42", `{"@root":42}`},
		{"raw bool", "true", `{"@root":true}`},
		{"raw nil", "nil", `{"@root":null}`},
		{"raw array", `{"a","b","c"}`, `{"@root":["a","b","c"]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestTextToJSON_InsertionOrder(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "table key order",
			input:    `data={["z"]=1,["a"]=2,["m"]=3}`,
			expected: `{"data":{"z":1,"a":2,"m":3}}`,
		},
		{
			name:     "top-level key order",
			input:    "z=1\na=2\nm=3",
			expected: `{"z":1,"a":2,"m":3}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestWithEmptyTableMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     string
		expected string
	}{
		{"null", "foo={}", "null", `{"foo":null}`},
		{"omit removes key", "foo={}\nbar=1", "omit", `{"bar":1}`},
		{"omit all empty", "foo={}", "omit", `{}`},
		{"array", "foo={}", "array", `{"foo":[]}`},
		{"object", "foo={}", "object", `{"foo":{}}`},
		{"nested omit", `data={["a"]={},["b"]=1}`, "omit", `{"data":{"b":1}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input, WithEmptyTableMode(tt.mode))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestWithArrayMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     string
		maxGap   int
		expected string
	}{
		{
			name:     "sparse contiguous",
			input:    `data={[1]="a",[2]="b",[3]="c"}`,
			mode:     "sparse",
			maxGap:   0,
			expected: `{"data":["a","b","c"]}`,
		},
		{
			name:     "sparse with gap",
			input:    `data={[1]="a",[3]="c"}`,
			mode:     "sparse",
			maxGap:   1,
			expected: `{"data":["a",null,"c"]}`,
		},
		{
			name:     "sparse exceeds gap",
			input:    `data={[1]="a",[5]="e"}`,
			mode:     "sparse",
			maxGap:   1,
			expected: `{"data":{"1":"a","5":"e"}}`,
		},
		{
			name:     "index-only implicit",
			input:    `data={"a","b","c"}`,
			mode:     "index-only",
			expected: `{"data":["a","b","c"]}`,
		},
		{
			name:     "index-only explicit int stays object",
			input:    `data={[1]="a",[2]="b"}`,
			mode:     "index-only",
			expected: `{"data":{"1":"a","2":"b"}}`,
		},
		{
			name:     "none mode",
			input:    `data={"a","b","c"}`,
			mode:     "none",
			expected: `{"data":{"1":"a","2":"b","3":"c"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input, WithArrayMode(tt.mode, tt.maxGap))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestWithStringTransform(t *testing.T) {
	longStr := strings.Repeat("x", 100)
	input := `long="` + longStr + `"`

	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"truncate", "truncate", `{"long":"xxxxxxxxxx"}`},
		{"empty", "empty", `{"long":""}`},
		{"redact", "redact", `{"long":"[redacted]"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", input, WithStringTransform(10, tt.mode))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}

	t.Run("replace", func(t *testing.T) {
		r, err := TextToJSON("input", input, WithStringTransform(10, "replace", "[too long]"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := readJSON(t, r)
		if got != `{"long":"[too long]"}` {
			t.Errorf("expected {\"long\":\"[too long]\"}, got %s", got)
		}
	})

	t.Run("under limit unchanged", func(t *testing.T) {
		r, err := TextToJSON("input", `short="hi"`, WithStringTransform(10, "redact"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := readJSON(t, r)
		if got != `{"short":"hi"}` {
			t.Errorf("expected {\"short\":\"hi\"}, got %s", got)
		}
	})
}

func TestTextToJSON_UTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CJK characters",
			input:    `name="你好世界"`,
			expected: `{"name":"你好世界"}`,
		},
		{
			name:     "emoji",
			input:    `icon="🎮🗡️🛡️"`,
			expected: `{"icon":"🎮🗡️🛡️"}`,
		},
		{
			name:     "Japanese",
			input:    `msg="こんにちは"`,
			expected: `{"msg":"こんにちは"}`,
		},
		{
			name:     "Korean",
			input:    `greeting="안녕하세요"`,
			expected: `{"greeting":"안녕하세요"}`,
		},
		{
			name:     "Cyrillic",
			input:    `text="Привет мир"`,
			expected: `{"text":"Привет мир"}`,
		},
		{
			name:     "mixed ASCII and multi-byte",
			input:    `item="Sword of the André"`,
			expected: `{"item":"Sword of the André"}`,
		},
		{
			name:     "UTF-8 in table keys",
			input:    `data={["名前"]="test"}`,
			expected: `{"data":{"名前":"test"}}`,
		},
		{
			name:     "UTF-8 in implicit array",
			input:    `items={"剑","盾","弓"}`,
			expected: `{"items":["剑","盾","弓"]}`,
		},
		{
			name:     "multi-byte with escapes",
			input:    `msg="line1\nこんにちは"`,
			expected: "{\"msg\":\"line1\\nこんにちは\"}",
		},
		{
			name:     "accented characters",
			input:    `café="résumé"`,
			expected: `{"café":"résumé"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := TextToJSON("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := readJSON(t, r)
			if got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}

			// Verify the output is valid JSON that round-trips correctly.
			var parsed map[string]any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("output is not valid JSON: %v", err)
			}
		})
	}
}

func TestFileToJSON(t *testing.T) {
	r, err := FileToJSON("../testdata/valid/simple.lua")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := readJSON(t, r)
	var v map[string]any
	if err := json.Unmarshal([]byte(got), &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v["name"] != "hello" {
		t.Errorf("expected name=hello, got %v", v["name"])
	}
	if v["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", v["count"])
	}
	if v["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", v["enabled"])
	}
}
