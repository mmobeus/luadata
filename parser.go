package luadata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eof = -1

type KeyValuePairs struct {
	orderedPairs []KeyValuePair
	NumValues    int
}

func (kvps KeyValuePairs) Len() int {
	return len(kvps.orderedPairs)
}

func (kvps KeyValuePairs) MaybeGetTable(key string) (KeyValuePairs, bool) {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == TableValue {
			if val, ok := kvp.Value.Raw.(KeyValuePairs); ok {
				return val, true
			}
		}
	}

	return KeyValuePairs{}, false
}

func (kvps KeyValuePairs) GetTableAsPair(key string) KeyValuePair {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == TableValue {
			if _, ok := kvp.Value.Raw.(KeyValuePairs); ok {
				return kvp
			}
		}
	}

	return KeyValuePair{}
}

func (kvps KeyValuePairs) GetTable(key string) KeyValuePairs {
	if val, ok := kvps.MaybeGetTable(key); ok {
		return val
	}

	return KeyValuePairs{}
}

func (kvps KeyValuePairs) MaybeGetString(key string) (string, bool) {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == StringValue {
			if val, ok := kvp.Value.Raw.(string); ok {
				return val, true
			}
		}
	}

	return "", false
}

func (kvps KeyValuePairs) GetString(key string) string {
	if val, ok := kvps.MaybeGetString(key); ok {
		return val
	}

	return ""
}

func (kvps KeyValuePairs) MaybeGetInt(key string) (int64, bool) {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == IntValue {
			if val, ok := kvp.Value.Raw.(int64); ok {
				return val, true
			}
		}
	}

	return 0, false
}

func (kvps KeyValuePairs) GetInt(key string) int64 {
	if val, ok := kvps.MaybeGetInt(key); ok {
		return val
	}

	return 0
}

func (kvps KeyValuePairs) MaybeGetFloat(key string) (float64, bool) {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == FloatValue {
			if val, ok := kvp.Value.Raw.(float64); ok {
				return val, true
			}
		}
	}

	return 0, false
}

func (kvps KeyValuePairs) GetFloat(key string) float64 {
	if val, ok := kvps.MaybeGetFloat(key); ok {
		return val
	}

	return 0
}

func (kvps KeyValuePairs) MaybeGetBool(key string) (bool, bool) {
	for _, kvp := range kvps.orderedPairs {
		if kvp.Key.Source == key && kvp.Value.Type == BoolValue {
			if val, ok := kvp.Value.Raw.(bool); ok {
				return val, true
			}
		}
	}

	return false, false
}

func (kvps KeyValuePairs) GetBool(key string) bool {
	if val, ok := kvps.MaybeGetBool(key); ok {
		return val
	}

	return false
}

func (kvps KeyValuePairs) MarshalJSON() ([]byte, error) {
	bb := &bytes.Buffer{}

	encoder := json.NewEncoder(bb)
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(convertTable(kvps.orderedPairs))
	if err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}

func convertTable(table []KeyValuePair) interface{} {
	// initially, this tried to attempt to swizzle tables with monotonically increasing numeric keys
	// into arrays, but that lead to too much non-determinism with lua's sparse array handling. for
	// example, when parsing several examples for what should be the same data (eg a single addon's
	// data for the same person over time), sometimes the sparse array may look like an array, and render
	// as an array, while other times it was too sparse to look like an array, so was rendered as a map.
	// this way is more inflated, but ultimately more deterministic (and not as beefy as the full on
	// KeyValuePairs type).
	tableMap := make(map[string]interface{})
	for _, kv := range table {
		if _, ok := tableMap[kv.Key.Source]; ok {
			tableMap["_wtf_warning"] = "key_collision"
		}

		tableMap[kv.Key.Source] = kv.Value
	}

	return tableMap
}

func (kvp KeyValuePairs) Pairs() []KeyValuePair {
	return kvp.orderedPairs[:]
}

type KeyValuePair struct {
	Key   Key   `json:"key"`
	Value Value `json:"value"`
}

type KeyType int

const (
	Identifier KeyType = iota
	Index
	String
	Int
	Bool
	Float
)

func (kt KeyType) Label() string {
	switch kt {
	case Identifier:
		return "identifier"
	case Index:
		return "index"
	case String:
		return "string"
	case Int:
		return "int"
	case Bool:
		return "bool"
	}

	return "unknown"
}

type ValueType int

const (
	TableValue ValueType = iota
	StringValue
	IntValue
	FloatValue
	BoolValue
	// EmptyValue is for '{}' which is ambiguously either an empty Table or empty Array
	// JSON may choose to NOT render it, or render it as 'null'.
	EmptyValue
	NilValue
)

type Key struct {
	Type   KeyType
	Source string
	Raw    any
}

func (k Key) MarshalJSON() ([]byte, error) {
	bb := &bytes.Buffer{}

	encoder := json.NewEncoder(bb)
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(keyJSON{
		Type: k.Type.Label(),
		Key:  k.Raw,
	})
	if err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}

type keyJSON struct {
	Type string `json:"type"`
	Key  any    `json:"key"`
}

type Value struct {
	Type   ValueType
	Source string
	Raw    any
}

func (v Value) MarshalJSON() ([]byte, error) {
	bb := &bytes.Buffer{}

	encoder := json.NewEncoder(bb)
	encoder.SetEscapeHTML(false)

	raw := v.Raw
	if v.Type == StringValue && len(v.Source) > 2048 {
		raw = "[removed]"
	}

	err := encoder.Encode(raw)
	if err != nil {
		return nil, err
	}

	return bb.Bytes(), nil
}

func ParseFile(filePath string) (KeyValuePairs, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return KeyValuePairs{}, fmt.Errorf("parse failure in %s: %w", filePath, err)
	}

	return ParseText(filePath, string(file))
}

func ParseText(name, text string) (KeyValuePairs, error) {
	lex := newLexer(name, text)

	kvPairs := KeyValuePairs{
		orderedPairs: make([]KeyValuePair, 0),
	}

	firstIteration := true
	for {
		err := lex.skipWhiteSpace()
		if err != nil {
			end := lex.start + 10
			if end > len(lex.input) {
				end = len(lex.input)
			}

			return KeyValuePairs{}, fmt.Errorf("parse failure in %s: line %d, position %d, next %q: %w",
				name, lex.line, lex.pos, lex.input[lex.start:end], err)
		}

		if lex.peek() == eof {
			break
		}

		if firstIteration {
			firstIteration = false
			r := lex.peek()

			isRawValue := false
			if !unicode.IsLetter(r) && r != '_' {
				// Starts with {, ", digit, -, [ etc. — definitely a raw value
				isRawValue = true
			} else {
				// Could be identifier=value OR a bare keyword (true/false/nil)
				saved := lex.save()
				_, _ = readLuaIdentifier(lex)
				lex.skipSpaceRunes()
				if lex.peek() != '=' {
					isRawValue = true
				}
				lex.restore(saved)
			}

			if isRawValue {
				kvPair, err := readRawValue(lex)
				if err != nil {
					end := lex.start + 10
					if end > len(lex.input) {
						end = len(lex.input)
					}
					return KeyValuePairs{}, fmt.Errorf("parse failure in %s: line %d, position %d, next %q: %w",
						name, lex.line, lex.pos, lex.input[lex.start:end], err)
				}
				kvPairs.orderedPairs = append(kvPairs.orderedPairs, kvPair)

				// Expect EOF after raw value (skip trailing whitespace/comments)
				err = lex.skipWhiteSpace()
				if err != nil {
					return KeyValuePairs{}, fmt.Errorf("parse failure in %s: %w", name, err)
				}
				if lex.peek() != eof {
					return KeyValuePairs{}, fmt.Errorf("parse failure in %s: unexpected content after raw value at line %d, position %d",
						name, lex.line, lex.pos)
				}
				break
			}
		}

		kvPair, err := readSavedVariable(lex)
		if err != nil {
			end := lex.start + 10
			if end > len(lex.input) {
				end = len(lex.input)
			}

			return KeyValuePairs{}, fmt.Errorf("parse failure in %s: line %d, position %d, next %q: %w",
				name, lex.line, lex.pos, lex.input[lex.start:end], err)
		}

		kvPairs.orderedPairs = append(kvPairs.orderedPairs, kvPair)
	}

	kvPairs.NumValues = lex.numValues

	return kvPairs, nil
}

func readRawValue(lex *lexer) (KeyValuePair, error) {
	value, err := readLuaValue(lex)
	if err != nil {
		return KeyValuePair{}, err
	}

	return KeyValuePair{
		Key: Key{
			Type:   Identifier,
			Source: "@root",
			Raw:    "@root",
		},
		Value: value,
	}, nil
}

type lexerState struct {
	start     int
	pos       int
	width     int
	line      int
	numValues int
}

func (l *lexer) save() lexerState {
	return lexerState{
		start:     l.start,
		pos:       l.pos,
		width:     l.width,
		line:      l.line,
		numValues: l.numValues,
	}
}

func (l *lexer) restore(s lexerState) {
	l.start = s.start
	l.pos = s.pos
	l.width = s.width
	l.line = s.line
	l.numValues = s.numValues
}

type lexer struct {
	input     string
	start     int
	pos       int
	width     int // width of last rune read by nextRune; 0 after eof or backup
	line      int
	numValues int
}

func newLexer(name, input string) *lexer {
	l := &lexer{
		input: input,
	}
	return l
}

func (l *lexer) nextRune() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, rWidth := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = rWidth
	l.pos += rWidth

	if r == '\n' {
		l.line++
	}

	return r
}

func (l *lexer) PeekString() string {
	end := l.start + 10
	if end > len(l.input) {
		end = len(l.input)
	}

	return l.input[l.start:end]
}

func (l *lexer) peek() rune {
	r := l.nextRune()

	l.backup()

	return r
}

func (l *lexer) backup() {
	// width == 0 means the last nextRune returned eof without advancing,
	// so there is nothing to back up over.
	if l.pos <= 0 || l.width == 0 {
		return
	}

	r, rWidth := utf8.DecodeLastRuneInString(l.input[:l.pos])
	l.pos -= rWidth
	// Leave width non-zero so that consecutive backup() calls can
	// continue to back up (acceptWhitespace relies on this).

	if r == '\n' {
		l.line--
	}
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) take() string {
	val := l.input[l.start:l.pos]
	l.start = l.pos

	return val
}

func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.nextRune()) {
		return true
	}

	l.backup()

	return false
}

func (l *lexer) acceptUntil(tr rune) {
	for {
		r := l.nextRune()
		if r == tr || r == eof {
			l.backup()

			return
		}
	}
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.nextRune()) {
	}

	l.backup()
}

func (l *lexer) acceptWhitespace() error {
	l.skipSpaceRunes()

	if l.peek() == '-' {
		l.nextRune()
		if l.peek() != '-' {
			l.backup()
		} else {
			l.nextRune()
			// check for a multiline style comment: `--[[ foo bar \n\nbaz --]]`.
			// the final '--' seems to be just sytlistic and not really required.
			// this is also what they do for multiline strings, so i guess technically,
			// need to puth this into a readMultiLineString function?
			skippedBlockComment := false
			if l.peek() == '[' {
				l.nextRune()
				pattern := "]"
				for l.nextRune() == '#' {
					pattern += "#"
				}
				l.backup()

				if l.peek() == '[' {
					// we KNOW it has to be a block comment. if it is not, that is an error.
					pattern += "]"
					// have
					// --[[
					// with possible '#'s between the brackets. now need to look for a matching
					// --]]
					// with the same pattern, where the '--' part is optional apparently.
					found := l.acceptThroughPattern(pattern)
					if found {
						skippedBlockComment = true
					} else {
						// didn't find the close pattern, fail.
						return fmt.Errorf("multiline string not properly closed, looking for %q", pattern)
					}
				} else {
					// not a block comment...back up just this last read b/c it might be a newline.
					// the outer one we know is a '[' and will just be skipped anyway
					l.backup()
				}
			} else {
				l.backup()
			}

			// not a block comment, so just keep ingoring until newline
			if !skippedBlockComment {
				l.acceptUntil('\n')
				err := l.acceptWhitespace()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (l *lexer) acceptThroughPattern(pattern string) bool {
	if len(pattern) == 0 {
		return true
	}

	firstRune := rune(pattern[0])
	done := false
	found := false
	for !done {
		l.acceptUntil(firstRune)

		foundMismatch := false
		for _, r := range pattern {
			nr := l.nextRune()
			if nr == eof {
				foundMismatch = true
				break
			}
			if nr != r {
				foundMismatch = true
				break
			}
		}

		if !foundMismatch {
			found = true
		}

		done = true
	}

	return found
}

func (l *lexer) skipSpaceRunes() {
	for isSpace(l.nextRune()) {
	}

	l.backup()
}

// skipWhitespace AND ignore whatever is buffered up (even if not whitespace).
func (l *lexer) skipWhiteSpace() error {
	err := l.acceptWhitespace()
	if err != nil {
		return err
	}

	l.ignore()

	return nil
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func readSavedVariable(lex *lexer) (KeyValuePair, error) {
	ident, err := readLuaIdentifier(lex)
	if err != nil {
		return KeyValuePair{}, err
	}

	err = lex.skipWhiteSpace()
	if err != nil {
		return KeyValuePair{}, err
	}

	if lex.nextRune() != '=' {
		return KeyValuePair{}, fmt.Errorf("expected '=' after identifier")
	}

	err = lex.skipWhiteSpace()
	if err != nil {
		return KeyValuePair{}, err
	}

	value, err := readLuaValue(lex)
	if err != nil {
		return KeyValuePair{}, err
	}

	return KeyValuePair{
		Key: Key{
			Type:   Identifier,
			Source: ident,
			Raw:    ident,
		},
		Value: value,
	}, nil
}

func readLuaIdentifier(lex *lexer) (string, error) {
	if unicode.IsDigit(lex.peek()) {
		return "", fmt.Errorf("expected identifier, but got a number")
	}

	for isAlphaNumeric(lex.nextRune()) {
	}

	lex.backup()

	ident := lex.take()
	if len(ident) == 0 {
		return "", fmt.Errorf("expected identifier")
	}

	return ident, nil
}

func readLuaValue(lex *lexer) (val Value, rerr error) {
	r := lex.peek()

	defer func() {
		if rerr == nil {
			lex.numValues++
		}
	}()

	switch {
	case r == '{':
		return readLuaTable(lex)
	case r == '"':
		return readQuotedStringValue(lex)
	case unicode.IsDigit(r) || r == '-':
		return readNumberValue(lex)
	default:
		value, err := readLuaIdentifier(lex)
		if err != nil {
			return Value{}, err
		}

		if value == "true" {
			return Value{
				Type:   BoolValue,
				Source: "true",
				Raw:    true,
			}, nil
		}
		if value == "false" {
			return Value{
				Type:   BoolValue,
				Source: "false",
				Raw:    false,
			}, nil
		}
		if value == "nil" {
			return Value{
				Type:   NilValue,
				Source: "nil",
				Raw:    nil,
			}, nil
		}

		return Value{}, fmt.Errorf("expected to read a value, got %q", value)
	}
}

func readLuaTable(lex *lexer) (Value, error) {
	if !lex.accept("{") {
		return Value{}, fmt.Errorf("expected '{'")
	}

	if lex.accept("}") {
		lex.ignore()

		return Value{
			Type:   EmptyValue,
			Source: "",
			Raw:    nil,
		}, nil
	}

	tableValue := KeyValuePairs{
		orderedPairs: make([]KeyValuePair, 0),
	}

	for {
		err := lex.skipWhiteSpace()
		if err != nil {
			return Value{}, err
		}

		if lex.accept("}") {
			err = lex.skipWhiteSpace()
			if err != nil {
				return Value{}, err
			}

			return Value{
				Type:   TableValue,
				Source: "",
				Raw:    tableValue,
			}, nil
		}

		r := lex.peek()

		var key Key
		if r == '[' {
			k, err := readLuaTableKey(lex)
			if err != nil {
				return Value{}, err
			}
			key = k

			err = lex.skipWhiteSpace()
			if err != nil {
				return Value{}, err
			}
			if !lex.accept("=") {
				return Value{}, fmt.Errorf("expected '='")
			}

			err = lex.skipWhiteSpace()
			if err != nil {
				return Value{}, err
			}
		} else {
			index := len(tableValue.orderedPairs) + 1
			key = Key{
				Type:   Index,
				Source: strconv.FormatInt(int64(index), 10),
				Raw:    len(tableValue.orderedPairs),
			}
		}

		val, err := readLuaValue(lex)
		if err != nil {
			return Value{}, err
		}

		tableValue.orderedPairs = append(tableValue.orderedPairs, KeyValuePair{
			Key:   key,
			Value: val,
		})

		err = lex.skipWhiteSpace()
		if err != nil {
			return Value{}, err
		}

		r = lex.peek()
		switch r {
		case ',':
			lex.accept(",")
		case '}':
			// ok, continue... this allows for simple "array" tables like {"foo"}, or at least
			// ones that are inlined, and have no trailing ',' after the last element.
		default:
			return Value{}, fmt.Errorf("expected ',' or '}'")
		}
	}
}

func readLuaTableKey(lex *lexer) (Key, error) {
	if !lex.accept("[") {
		return Key{}, fmt.Errorf("expected '['")
	}

	err := lex.skipWhiteSpace()
	if err != nil {
		return Key{}, err
	}

	val, err := readLuaValue(lex)
	if err != nil {
		return Key{}, err
	}

	err = lex.skipWhiteSpace()
	if err != nil {
		return Key{}, err
	}

	if !lex.accept("]") {
		return Key{}, fmt.Errorf("expected ']'")
	}

	err = lex.skipWhiteSpace()
	if err != nil {
		return Key{}, err
	}

	switch val.Type {
	case StringValue:
		return Key{
			Type:   String,
			Source: val.Source,
			Raw:    val.Raw,
		}, nil
	case IntValue:
		return Key{
			Type:   Int,
			Source: val.Source,
			Raw:    val.Raw,
		}, nil
	case BoolValue:
		return Key{
			Type:   Bool,
			Source: val.Source,
			Raw:    val.Raw,
		}, nil
	case FloatValue:
		return Key{
			Type:   Float,
			Source: val.Source,
			Raw:    val.Raw,
		}, nil
	default:
		return Key{}, fmt.Errorf("unsupported value type for key: %v", val.Type)
	}
}

func readQuotedStringValue(lex *lexer) (Value, error) {
	if !lex.accept("\"") {
		return Value{}, fmt.Errorf("expected '\"'")
	}

	for {
		switch lex.nextRune() {
		case '\\':
			curr := lex.pos
			lex.acceptRun("\\")
			numEscapes := (lex.pos - curr) + 1

			if numEscapes%2 != 0 && lex.peek() == '"' {
				_ = lex.nextRune()
			}
		case eof, '\n':
			return Value{}, fmt.Errorf("unterminated quoted string")
		case '"':
			quotedVal := lex.take()
			sourceVal := quotedVal[1 : len(quotedVal)-1]

			return Value{
				Type:   StringValue,
				Source: sourceVal,
				Raw:    sourceVal,
			}, nil
		}
	}
}

func readNumberValue(lex *lexer) (Value, error) {
	lex.accept("-")

	for unicode.IsDigit(lex.nextRune()) {
	}

	lex.backup()

	isInt := true
	if lex.accept(".") {
		isInt = false
		for unicode.IsDigit(lex.nextRune()) {
		}

		lex.backup()
	}
	if lex.accept("eE") {
		lex.accept("+-")
		lex.acceptRun("0123456789")
	}

	numValAsString := lex.take()

	if isInt {
		val, err := strconv.ParseInt(numValAsString, 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("invalid int: %w", err)
		}

		return Value{
			Type:   IntValue,
			Source: numValAsString,
			Raw:    val,
		}, nil
	}

	val, err := strconv.ParseFloat(numValAsString, 64)
	if err != nil {
		return Value{}, fmt.Errorf("invalid float: %w", err)
	}

	return Value{
		Type:   FloatValue,
		Source: numValAsString,
		Raw:    val,
	}, nil
}
