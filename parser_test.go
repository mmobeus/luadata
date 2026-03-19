package luadata

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestParseText_EmptyTables(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		valueType ValueType // {} is EmptyValue; whitespace-only tables are TableValue with 0 entries
	}{
		{"inline empty", "foo={}", EmptyValue},
		{"newline before close", "foo={\n}", TableValue},
		{"newline before close with trailing newline", "foo={\n}\n", TableValue},
		{"spaces inside", "foo={ }", TableValue},
		{"tabs inside", "foo={\t}", TableValue},
		{"multiple newlines inside", "foo={\n\n\n}", TableValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Len() != 1 {
				t.Fatalf("expected 1 top-level pair, got %d", result.Len())
			}
			pair := result.orderedPairs[0]
			if pair.Key.Source != "foo" {
				t.Errorf("expected key %q, got %q", "foo", pair.Key.Source)
			}
			if pair.Value.Type != tt.valueType {
				t.Errorf("expected value type %v, got %v", tt.valueType, pair.Value.Type)
			}
		})
	}
}

func TestParseText_SimpleValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		key       string
		valueType ValueType
		raw       any
	}{
		{"string", `foo="bar"`, "foo", StringValue, "bar"},
		{"int", "foo=42", "foo", IntValue, int64(42)},
		{"float", "foo=3.14", "foo", FloatValue, 3.14},
		{"bool true", "foo=true", "foo", BoolValue, true},
		{"bool false", "foo=false", "foo", BoolValue, false},
		{"nil", "foo=nil", "foo", NilValue, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Len() != 1 {
				t.Fatalf("expected 1 pair, got %d", result.Len())
			}
			pair := result.orderedPairs[0]
			if pair.Key.Source != tt.key {
				t.Errorf("expected key %q, got %q", tt.key, pair.Key.Source)
			}
			if pair.Value.Type != tt.valueType {
				t.Errorf("expected value type %v, got %v", tt.valueType, pair.Value.Type)
			}
			if pair.Value.Raw != tt.raw {
				t.Errorf("expected raw %v, got %v", tt.raw, pair.Value.Raw)
			}
		})
	}
}

func TestParseText_MultipleVariables(t *testing.T) {
	input := "a=1\nb=2\nc=3\n"
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 3 {
		t.Fatalf("expected 3 pairs, got %d", result.Len())
	}

	expected := []struct {
		key string
		raw int64
	}{
		{"a", 1},
		{"b", 2},
		{"c", 3},
	}
	for i, e := range expected {
		pair := result.orderedPairs[i]
		if pair.Key.Source != e.key {
			t.Errorf("pair %d: expected key %q, got %q", i, e.key, pair.Key.Source)
		}
		if pair.Value.Raw != e.raw {
			t.Errorf("pair %d: expected raw %v, got %v", i, e.raw, pair.Value.Raw)
		}
	}
}

func TestParseText_NestedTable(t *testing.T) {
	input := `foo={[1]="a",[2]="b"}`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	table, ok := result.MaybeGetTable("foo")
	if !ok {
		t.Fatal("expected table for key 'foo'")
	}
	if table.Len() != 2 {
		t.Fatalf("expected 2 entries in table, got %d", table.Len())
	}
}

func TestParseText_TableWithBracketedKeys(t *testing.T) {
	input := `data={["name"]="test",["count"]=5}`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	table := result.GetTable("data")
	name := table.GetString("name")
	if name != "test" {
		t.Errorf("expected name %q, got %q", "test", name)
	}
	count := table.GetInt("count")
	if count != 5 {
		t.Errorf("expected count 5, got %d", count)
	}
}

func TestParseText_ImplicitIndexKeys(t *testing.T) {
	input := `arr={"a","b","c"}`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	table := result.GetTable("arr")
	if table.Len() != 3 {
		t.Fatalf("expected 3 entries, got %d", table.Len())
	}
	for i, pair := range table.Pairs() {
		if pair.Key.Type != Index {
			t.Errorf("entry %d: expected Index key type, got %v", i, pair.Key.Type)
		}
	}
}

func TestParseText_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"spaces around equals", "foo = 42"},
		{"tabs around equals", "foo\t=\t42"},
		{"leading whitespace", "  foo=42"},
		{"trailing newline", "foo=42\n"},
		{"multiple trailing newlines", "foo=42\n\n\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Len() != 1 {
				t.Fatalf("expected 1 pair, got %d", result.Len())
			}
			if result.orderedPairs[0].Value.Raw != int64(42) {
				t.Errorf("expected 42, got %v", result.orderedPairs[0].Value.Raw)
			}
		})
	}
}

func TestParseText_Comments(t *testing.T) {
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
			_, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseText_DeeplyNestedTable(t *testing.T) {
	input := `root={["child"]={["grandchild"]={["value"]=42}}}`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	child := result.GetTable("root")
	grandchild, ok := child.MaybeGetTable("child")
	if !ok {
		t.Fatal("expected child table")
	}
	leaf, ok := grandchild.MaybeGetTable("grandchild")
	if !ok {
		t.Fatal("expected grandchild table")
	}
	val := leaf.GetInt("value")
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestParseText_MultilineTable(t *testing.T) {
	input := "data={\n[\"a\"]=1,\n[\"b\"]=2,\n[\"c\"]=3,\n}\n"
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	table := result.GetTable("data")
	if table.Len() != 3 {
		t.Fatalf("expected 3 entries, got %d", table.Len())
	}
}

func TestParseText_Errors(t *testing.T) {
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
			_, err := ParseText("input", tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestParseText_EscapedStrings(t *testing.T) {
	input := `foo="hello\"world"`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	val := result.GetString("foo")
	if val != `hello"world` {
		t.Errorf("expected %q, got %q", `hello"world`, val)
	}
}

func TestParseText_EscapeSequences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantRaw  string
		wantJSON string
	}{
		{
			// Lua: \" is an escaped quote → string value contains a literal "
			name:     "escaped quote mid-string",
			input:    `foo="hello\"world"`,
			wantRaw:  `hello"world`,
			wantJSON: `{"foo":"hello\"world"}`,
		},
		{
			// Lua: \\ is an escaped backslash → string value contains a literal \
			name:     "double backslash before closing quote",
			input:    `foo="hello\\"`,
			wantRaw:  `hello\`,
			wantJSON: `{"foo":"hello\\"}`,
		},
		{
			// Lua: \\ then \" → literal backslash followed by literal quote
			name:     "triple backslash before quote",
			input:    `foo="hello\\\"world"`,
			wantRaw:  `hello\"world`,
			wantJSON: `{"foo":"hello\\\"world"}`,
		},
		{
			// Lua: \n is a newline escape → string value contains a newline
			name:     "backslash n is newline",
			input:    `foo="hello\nworld"`,
			wantRaw:  "hello\nworld",
			wantJSON: "{\"foo\":\"hello\\nworld\"}",
		},
		{
			// Lua: \t is a tab escape → string value contains a tab
			name:     "backslash t is tab",
			input:    `foo="hello\tworld"`,
			wantRaw:  "hello\tworld",
			wantJSON: "{\"foo\":\"hello\\tworld\"}",
		},
		{
			// Lua: \\\\ is two escaped backslashes → string value contains \\
			name:     "four backslashes is two literal backslashes",
			input:    `foo="hello\\\\"`,
			wantRaw:  `hello\\`,
			wantJSON: `{"foo":"hello\\\\"}`,
		},
		{
			// Lua: \" at start → string starts with a literal quote
			name:     "escaped quote at start",
			input:    `foo="\"hello"`,
			wantRaw:  `"hello`,
			wantJSON: `{"foo":"\"hello"}`,
		},
		{
			// Lua: multiple \" → each is a literal quote in the string
			name:     "multiple escaped quotes",
			input:    `foo="say \"hi\" ok"`,
			wantRaw:  `say "hi" ok`,
			wantJSON: `{"foo":"say \"hi\" ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			raw := result.GetString("foo")
			if raw != tt.wantRaw {
				t.Errorf("Raw: expected %q, got %q", tt.wantRaw, raw)
			}

			gotJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}
			gotStr := strings.TrimRight(string(gotJSON), "\n")
			if gotStr != tt.wantJSON {
				t.Errorf("JSON: expected %s, got %s", tt.wantJSON, gotStr)
			}
		})
	}
}

func TestParseText_EmptyInput(t *testing.T) {
	result, err := ParseText("input", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 0 {
		t.Errorf("expected 0 pairs, got %d", result.Len())
	}
}

func TestParseText_WhitespaceOnlyInput(t *testing.T) {
	result, err := ParseText("input", "   \n\t\n  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 0 {
		t.Errorf("expected 0 pairs, got %d", result.Len())
	}
}

func TestParseText_TableNoTrailingComma(t *testing.T) {
	input := `foo={["a"]=1,["b"]=2}`
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	table := result.GetTable("foo")
	if table.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", table.Len())
	}
}

func TestParseText_RawValues(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		valueType ValueType
		raw       any
	}{
		{"raw table", `{["a"]=1,["b"]=2}`, TableValue, nil},
		{"raw string", `"hello"`, StringValue, "hello"},
		{"raw int", "42", IntValue, int64(42)},
		{"raw negative int", "-7", IntValue, int64(-7)},
		{"raw float", "3.14", FloatValue, 3.14},
		{"raw bool true", "true", BoolValue, true},
		{"raw bool false", "false", BoolValue, false},
		{"raw nil", "nil", NilValue, nil},
		{"raw empty table", "{}", EmptyValue, nil},
		{"raw array", `{"a","b","c"}`, TableValue, nil},
		{"with leading comment", "-- comment\n42", IntValue, int64(42)},
		{"with surrounding whitespace", "  42  ", IntValue, int64(42)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Len() != 1 {
				t.Fatalf("expected 1 pair, got %d", result.Len())
			}
			pair := result.orderedPairs[0]
			if pair.Key.Source != "@root" {
				t.Errorf("expected key %q, got %q", "@root", pair.Key.Source)
			}
			if pair.Key.Type != Identifier {
				t.Errorf("expected key type Identifier, got %v", pair.Key.Type)
			}
			if pair.Value.Type != tt.valueType {
				t.Errorf("expected value type %v, got %v", tt.valueType, pair.Value.Type)
			}
			// Skip raw comparison for table types (compared by reference)
			if tt.raw != nil && pair.Value.Raw != tt.raw {
				t.Errorf("expected raw %v, got %v", tt.raw, pair.Value.Raw)
			}
		})
	}
}

func TestParseText_RawValueErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing content after int", "42 foo"},
		{"trailing content after table", "{} foo=1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseText("input", tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestParseText_RawValueNoRegression(t *testing.T) {
	// Identifiers that start with keywords should still parse as variable assignments
	input := "true_val=true"
	result, err := ParseText("input", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Len() != 1 {
		t.Fatalf("expected 1 pair, got %d", result.Len())
	}
	pair := result.orderedPairs[0]
	if pair.Key.Source != "true_val" {
		t.Errorf("expected key %q, got %q", "true_val", pair.Key.Source)
	}
	if pair.Value.Type != BoolValue {
		t.Errorf("expected BoolValue, got %v", pair.Value.Type)
	}
}

func TestParseText_NegativeNumbers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		key   string
		raw   int64
	}{
		{"top level", "foo=-7", "foo", -7},
		{"in table", `data={["val"]=-7}`, "val", -7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var val int64
			if tt.name == "top level" {
				val = result.GetInt(tt.key)
			} else {
				table := result.GetTable("data")
				val = table.GetInt(tt.key)
			}
			if val != tt.raw {
				t.Errorf("expected %d, got %d", tt.raw, val)
			}
		})
	}
}

func TestParseText_StringTransform(t *testing.T) {
	longStr := strings.Repeat("x", 100)
	shortStr := "short"
	input := fmt.Sprintf(`long="%s"`, longStr)
	shortInput := fmt.Sprintf(`short="%s"`, shortStr)

	t.Run("default no transform", func(t *testing.T) {
		result, err := ParseText("input", input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != longStr {
			t.Errorf("expected Source to be unmodified, got %q", pair.Value.Source)
		}
		if pair.Value.Raw != longStr {
			t.Errorf("expected Raw to be unmodified, got %q", pair.Value.Raw)
		}
		if pair.Value.Transformed {
			t.Error("expected Transformed to be false")
		}
	})

	t.Run("truncate", func(t *testing.T) {
		result, err := ParseText("input", input, WithStringTransform(StringTransform{
			MaxLen: 10,
			Mode:   StringTransformTruncate,
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != "xxxxxxxxxx" {
			t.Errorf("expected truncated Source, got %q", pair.Value.Source)
		}
		if pair.Value.Raw != "xxxxxxxxxx" {
			t.Errorf("expected truncated Raw, got %q", pair.Value.Raw)
		}
		if !pair.Value.Transformed {
			t.Error("expected Transformed to be true")
		}
	})

	t.Run("empty", func(t *testing.T) {
		result, err := ParseText("input", input, WithStringTransform(StringTransform{
			MaxLen: 10,
			Mode:   StringTransformEmpty,
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != "" {
			t.Errorf("expected empty Source, got %q", pair.Value.Source)
		}
		if pair.Value.Raw != "" {
			t.Errorf("expected empty Raw, got %q", pair.Value.Raw)
		}
		if !pair.Value.Transformed {
			t.Error("expected Transformed to be true")
		}
	})

	t.Run("redact", func(t *testing.T) {
		result, err := ParseText("input", input, WithStringTransform(StringTransform{
			MaxLen: 10,
			Mode:   StringTransformRedact,
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != "[redacted]" {
			t.Errorf("expected [redacted] Source, got %q", pair.Value.Source)
		}
		if pair.Value.Raw != "[redacted]" {
			t.Errorf("expected [redacted] Raw, got %q", pair.Value.Raw)
		}
		if !pair.Value.Transformed {
			t.Error("expected Transformed to be true")
		}
	})

	t.Run("replace", func(t *testing.T) {
		result, err := ParseText("input", input, WithStringTransform(StringTransform{
			MaxLen:      10,
			Mode:        StringTransformReplace,
			Replacement: "[too long]",
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != "[too long]" {
			t.Errorf("expected [too long] Source, got %q", pair.Value.Source)
		}
		if pair.Value.Raw != "[too long]" {
			t.Errorf("expected [too long] Raw, got %q", pair.Value.Raw)
		}
		if !pair.Value.Transformed {
			t.Error("expected Transformed to be true")
		}
	})

	t.Run("under limit unchanged", func(t *testing.T) {
		result, err := ParseText("input", shortInput, WithStringTransform(StringTransform{
			MaxLen: 10,
			Mode:   StringTransformRedact,
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pair := result.orderedPairs[0]
		if pair.Value.Source != shortStr {
			t.Errorf("expected Source %q, got %q", shortStr, pair.Value.Source)
		}
		if pair.Value.Raw != shortStr {
			t.Errorf("expected Raw %q, got %q", shortStr, pair.Value.Raw)
		}
		if pair.Value.Transformed {
			t.Error("expected Transformed to be false")
		}
	})
}

func TestConvertTable_ImplicitArrayRendersAsJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple implicit array",
			input:    `data={"foo","bar","baz"}`,
			expected: `{"data":["foo","bar","baz"]}`,
		},
		{
			name: "nested implicit array",
			input: `data={
["list"]={"a","b","c"},
}`,
			expected: `{"data":{"list":["a","b","c"]}}`,
		},
		{
			name:     "explicit integer keys render as sparse array by default",
			input:    `data={[1]="a",[3]="c"}`,
			expected: `{"data":["a",null,"c"]}`,
		},
		{
			name: "mixed keys render as map",
			input: `data={
"foo",
["name"]="bar",
}`,
			expected: `{"data":{"1":"foo","name":"bar"}}`,
		},
		{
			name:     "preserves insertion order",
			input:    `data={["z"]=1,["a"]=2,["m"]=3}`,
			expected: `{"data":{"z":1,"a":2,"m":3}}`,
		},
		{
			name:     "top-level key order preserved",
			input:    "z=1\na=2\nm=3",
			expected: `{"z":1,"a":2,"m":3}`,
		},
		{
			name:     "nested table order preserved",
			input:    `outer={["inner"]={["z"]=1,["a"]=2}}`,
			expected: `{"outer":{"inner":{"z":1,"a":2}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseText("input", tt.input)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			got, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			// json.NewEncoder adds a trailing newline; json.Marshal does not,
			// but MarshalJSON uses NewEncoder internally, so trim.
			gotStr := string(got)
			// Remove trailing newline if present
			if len(gotStr) > 0 && gotStr[len(gotStr)-1] == '\n' {
				gotStr = gotStr[:len(gotStr)-1]
			}

			if gotStr != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, gotStr)
			}
		})
	}
}

func TestWithArrayDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     ArrayMode // nil means no option (default)
		expected string
	}{
		{
			name:     "contiguous int keys as array",
			input:    `data={[1]="a",[2]="b",[3]="c"}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":["a","b","c"]}`,
		},
		{
			name:     "sparse within gap",
			input:    `data={[1]="a",[3]="c"}`,
			mode:     ArrayModeSparse{MaxGap: 1},
			expected: `{"data":["a",null,"c"]}`,
		},
		{
			name:     "sparse exceeds gap",
			input:    `data={[1]="a",[5]="e"}`,
			mode:     ArrayModeSparse{MaxGap: 1},
			expected: `{"data":{"1":"a","5":"e"}}`,
		},
		{
			name:     "mixed int and string keys stays object",
			input:    `data={[1]="a",["name"]="b"}`,
			mode:     ArrayModeSparse{MaxGap: 10},
			expected: `{"data":{"1":"a","name":"b"}}`,
		},
		{
			name:     "nested tables propagate option",
			input:    `data={["items"]={[1]="x",[2]="y"}}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":{"items":["x","y"]}}`,
		},
		{
			name:     "default renders contiguous int keys as array",
			input:    `data={[1]="a",[2]="b"}`,
			mode:     nil,
			expected: `{"data":["a","b"]}`,
		},
		{
			name:     "index only mode renders int keys as object",
			input:    `data={[1]="a",[2]="b"}`,
			mode:     ArrayModeIndexOnly{},
			expected: `{"data":{"1":"a","2":"b"}}`,
		},
		{
			name:     "index only mode renders implicit index as array",
			input:    `data={"a","b","c"}`,
			mode:     ArrayModeIndexOnly{},
			expected: `{"data":["a","b","c"]}`,
		},
		{
			name:     "none mode renders everything as object",
			input:    `data={"a","b","c"}`,
			mode:     ArrayModeNone{},
			expected: `{"data":{"1":"a","2":"b","3":"c"}}`,
		},
		{
			name:     "none mode renders int keys as object",
			input:    `data={[1]="a",[2]="b"}`,
			mode:     ArrayModeNone{},
			expected: `{"data":{"1":"a","2":"b"}}`,
		},
		{
			name:     "implicit index unaffected by sparse option",
			input:    `data={"a","b","c"}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":["a","b","c"]}`,
		},
		{
			name:     "keys not starting at 1 with gap 0",
			input:    `data={[2]="a",[3]="b"}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":{"2":"a","3":"b"}}`,
		},
		{
			name:     "keys not starting at 1 gap ok",
			input:    `data={[2]="a",[3]="b"}`,
			mode:     ArrayModeSparse{MaxGap: 1},
			expected: `{"data":[null,"a","b"]}`,
		},
		{
			name:     "single element array",
			input:    `data={[1]="only"}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":["only"]}`,
		},
		{
			name:     "int keys with nested table values",
			input:    `data={[1]={["name"]="a"},[2]={["name"]="b"}}`,
			mode:     ArrayModeSparse{MaxGap: 0},
			expected: `{"data":[{"name":"a"},{"name":"b"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.mode != nil {
				opts = append(opts, WithArrayDetection(tt.mode))
			}

			result, err := ParseText("input", tt.input, opts...)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			got, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			gotStr := string(got)
			if len(gotStr) > 0 && gotStr[len(gotStr)-1] == '\n' {
				gotStr = gotStr[:len(gotStr)-1]
			}

			if gotStr != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, gotStr)
			}
		})
	}
}

func TestWithEmptyTableMode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		mode     EmptyTableMode
		useOpt   bool // false = use default (no option)
		expected string
	}{
		// Default behavior (no option): null
		{
			name:     "default renders empty as null",
			input:    `foo={}`,
			expected: `{"foo":null}`,
		},
		// EmptyTableNull
		{
			name:     "null mode inline empty",
			input:    `foo={}`,
			mode:     EmptyTableNull,
			useOpt:   true,
			expected: `{"foo":null}`,
		},
		{
			name:     "null mode whitespace empty",
			input:    "foo={\n}",
			mode:     EmptyTableNull,
			useOpt:   true,
			expected: `{"foo":null}`,
		},
		// EmptyTableOmit
		{
			name:     "omit mode removes key",
			input:    "foo={}\nbar=1",
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{"bar":1}`,
		},
		{
			name:     "omit mode whitespace empty",
			input:    "foo={\n}\nbar=1",
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{"bar":1}`,
		},
		{
			name:     "omit mode all empty",
			input:    `foo={}`,
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{}`,
		},
		// EmptyTableArray
		{
			name:     "array mode inline empty",
			input:    `foo={}`,
			mode:     EmptyTableArray,
			useOpt:   true,
			expected: `{"foo":[]}`,
		},
		{
			name:     "array mode whitespace empty",
			input:    "foo={\n}",
			mode:     EmptyTableArray,
			useOpt:   true,
			expected: `{"foo":[]}`,
		},
		// EmptyTableObject
		{
			name:     "object mode inline empty",
			input:    `foo={}`,
			mode:     EmptyTableObject,
			useOpt:   true,
			expected: `{"foo":{}}`,
		},
		{
			name:     "object mode whitespace empty",
			input:    "foo={\n}",
			mode:     EmptyTableObject,
			useOpt:   true,
			expected: `{"foo":{}}`,
		},
		// Nested empty tables
		{
			name:     "nested empty null",
			input:    `data={["a"]={},["b"]=1}`,
			mode:     EmptyTableNull,
			useOpt:   true,
			expected: `{"data":{"a":null,"b":1}}`,
		},
		{
			name:     "nested empty omit",
			input:    `data={["a"]={},["b"]=1}`,
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{"data":{"b":1}}`,
		},
		{
			name:     "nested empty array",
			input:    `data={["a"]={},["b"]=1}`,
			mode:     EmptyTableArray,
			useOpt:   true,
			expected: `{"data":{"a":[],"b":1}}`,
		},
		{
			name:     "nested empty object",
			input:    `data={["a"]={},["b"]=1}`,
			mode:     EmptyTableObject,
			useOpt:   true,
			expected: `{"data":{"a":{},"b":1}}`,
		},
		// Empty tables inside implicit index arrays
		{
			name:     "array element empty null",
			input:    `data={{},1,2}`,
			mode:     EmptyTableNull,
			useOpt:   true,
			expected: `{"data":[null,1,2]}`,
		},
		{
			name:     "array element empty omit",
			input:    `data={{},1,2}`,
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{"data":[1,2]}`,
		},
		{
			name:     "array element empty array",
			input:    `data={{},1,2}`,
			mode:     EmptyTableArray,
			useOpt:   true,
			expected: `{"data":[[],1,2]}`,
		},
		{
			name:     "array element empty object",
			input:    `data={{},1,2}`,
			mode:     EmptyTableObject,
			useOpt:   true,
			expected: `{"data":[{},1,2]}`,
		},
		// Raw value (bare {}) — @root must never be omitted
		{
			name:     "raw empty table omit falls back to null",
			input:    `{}`,
			mode:     EmptyTableOmit,
			useOpt:   true,
			expected: `{"@root":null}`,
		},
		{
			name:     "raw empty table null",
			input:    `{}`,
			mode:     EmptyTableNull,
			useOpt:   true,
			expected: `{"@root":null}`,
		},
		{
			name:     "raw empty table array",
			input:    `{}`,
			mode:     EmptyTableArray,
			useOpt:   true,
			expected: `{"@root":[]}`,
		},
		{
			name:     "raw empty table object",
			input:    `{}`,
			mode:     EmptyTableObject,
			useOpt:   true,
			expected: `{"@root":{}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.useOpt {
				opts = append(opts, WithEmptyTableMode(tt.mode))
			}

			result, err := ParseText("input", tt.input, opts...)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			got, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("unexpected marshal error: %v", err)
			}

			gotStr := string(got)
			if len(gotStr) > 0 && gotStr[len(gotStr)-1] == '\n' {
				gotStr = gotStr[:len(gotStr)-1]
			}

			if gotStr != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, gotStr)
			}
		})
	}
}
