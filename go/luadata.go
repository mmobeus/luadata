// Package luadata converts Lua data files to JSON using a Rust shared library via purego.
//
// This package provides a pure-Go interface (no CGO required) by loading a
// platform-specific Rust shared library at runtime.
package luadata

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mmobeus/luadata/go/internal/ffi"
)

// TextToJSON parses a Lua data string and returns the JSON as an io.Reader.
func TextToJSON(name, text string, opts ...Option) (io.Reader, error) {
	result, err := textToJSONString(text, opts...)
	if err != nil {
		return nil, fmt.Errorf("parse failure in %s: %w", name, err)
	}
	return strings.NewReader(result), nil
}

// ToJSON parses Lua data bytes and returns the JSON as an io.Reader.
func ToJSON(lua []byte, opts ...Option) (io.Reader, error) {
	return TextToJSON("input", string(lua), opts...)
}

// ReaderToJSON parses Lua data from an io.Reader and returns JSON as an io.Reader.
func ReaderToJSON(name string, r io.Reader, opts ...Option) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("parse failure in %s: %w", name, err)
	}
	return TextToJSON(name, string(data), opts...)
}

// FileToJSON parses a Lua data file and returns JSON as an io.Reader.
func FileToJSON(filePath string, opts ...Option) (io.Reader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return ReaderToJSON(filePath, file, opts...)
}

// textToJSONString calls the Rust FFI and returns the raw JSON string result.
func textToJSONString(text string, opts ...Option) (string, error) {
	optionsJSON := buildOptionsJSON(opts)

	envelope, err := ffi.Call(text, optionsJSON)
	if err != nil {
		return "", err
	}

	var resp struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(envelope), &resp); err != nil {
		return "", fmt.Errorf("invalid response from library: %w", err)
	}

	if resp.Error != "" {
		return "", fmt.Errorf("%s", resp.Error)
	}

	return resp.Result, nil
}

// buildOptionsJSON serializes the functional options into JSON for the FFI call.
func buildOptionsJSON(opts []Option) string {
	if len(opts) == 0 {
		return ""
	}

	cfg := &optionsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	m := make(map[string]any)

	if cfg.emptyTable != "" {
		m["empty_table"] = cfg.emptyTable
	}
	if cfg.arrayMode != "" {
		m["array_mode"] = cfg.arrayMode
	}
	if cfg.arrayMaxGap != nil {
		m["array_max_gap"] = *cfg.arrayMaxGap
	}
	if cfg.stringTransform != nil {
		st := map[string]any{
			"max_len": cfg.stringTransform.MaxLen,
		}
		if cfg.stringTransform.Mode != "" {
			st["mode"] = cfg.stringTransform.Mode
		}
		if cfg.stringTransform.Replacement != "" {
			st["replacement"] = cfg.stringTransform.Replacement
		}
		m["string_transform"] = st
	}
	if cfg.schema != "" {
		m["schema"] = cfg.schema
	}
	if cfg.unknownFields != "" {
		m["unknown_fields"] = cfg.unknownFields
	}

	if len(m) == 0 {
		return ""
	}

	b, _ := json.Marshal(m)
	return string(b)
}
