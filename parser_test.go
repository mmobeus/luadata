package luadata

import (
	"encoding/json"
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
	if val != `hello\"world` {
		t.Errorf("expected %q, got %q", `hello\"world`, val)
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
			name:     "explicit integer keys render as map",
			input:    `data={[1]="a",[3]="c"}`,
			expected: `{"data":{"1":"a","3":"c"}}`,
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
