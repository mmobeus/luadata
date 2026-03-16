// Package luadata converts Lua data table files into JSON.
package luadata

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// ToJSON parses Lua data bytes and returns the equivalent JSON as an io.Reader.
func ToJSON(lua []byte) (io.Reader, error) {
	return TextToJSON("input", string(lua))
}

// TextToJSON parses a Lua data string and returns the equivalent JSON as an io.Reader.
func TextToJSON(name, text string) (io.Reader, error) {
	parsed, err := ParseText(name, text)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(parsed)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// ReaderToJSON parses Lua data from an io.Reader and returns the equivalent JSON as an io.Reader.
func ReaderToJSON(name string, r io.Reader) (io.Reader, error) {
	parsed, err := ParseReader(name, r)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(parsed)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// FileToJSON parses a Lua data file and returns the equivalent JSON as an io.Reader.
func FileToJSON(filePath string) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return ReaderToJSON(filePath, file)
}
