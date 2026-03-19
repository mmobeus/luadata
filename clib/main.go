package main

// #include <stdlib.h>
import "C"

import (
	"encoding/json"
	"fmt"
	"io"
	"unsafe"

	"github.com/mmobeus/luadata"
)

// optionsJSON mirrors the JSON structure passed from Python.
type optionsJSON struct {
	EmptyTable      string               `json:"empty_table,omitempty"`
	ArrayMode       string               `json:"array_mode,omitempty"`
	ArrayMaxGap     *int                 `json:"array_max_gap,omitempty"`
	StringTransform *stringTransformJSON `json:"string_transform,omitempty"`
}

type stringTransformJSON struct {
	MaxLen      int    `json:"max_len"`
	Mode        string `json:"mode,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

func parseOptions(raw string) ([]luadata.Option, error) {
	if raw == "" {
		return nil, nil
	}

	var oj optionsJSON
	if err := json.Unmarshal([]byte(raw), &oj); err != nil {
		return nil, fmt.Errorf("invalid options JSON: %w", err)
	}

	var opts []luadata.Option

	if oj.EmptyTable != "" {
		switch oj.EmptyTable {
		case "null":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableNull))
		case "omit":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableOmit))
		case "array":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableArray))
		case "object":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableObject))
		default:
			return nil, fmt.Errorf("unknown empty_table value: %q", oj.EmptyTable)
		}
	}

	if oj.ArrayMode != "" {
		switch oj.ArrayMode {
		case "none":
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeNone{}))
		case "index-only":
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeIndexOnly{}))
		case "sparse":
			maxGap := 20
			if oj.ArrayMaxGap != nil {
				maxGap = *oj.ArrayMaxGap
			}
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeSparse{MaxGap: maxGap}))
		default:
			return nil, fmt.Errorf("unknown array_mode value: %q", oj.ArrayMode)
		}
	}

	if oj.StringTransform != nil {
		st := oj.StringTransform
		if st.MaxLen <= 0 {
			return nil, fmt.Errorf("string_transform.max_len must be a positive number")
		}

		transform := luadata.StringTransform{
			MaxLen: st.MaxLen,
		}

		mode := "truncate"
		if st.Mode != "" {
			mode = st.Mode
		}

		switch mode {
		case "truncate":
			transform.Mode = luadata.StringTransformTruncate
		case "empty":
			transform.Mode = luadata.StringTransformEmpty
		case "redact":
			transform.Mode = luadata.StringTransformRedact
		case "replace":
			transform.Mode = luadata.StringTransformReplace
			transform.Replacement = st.Replacement
		default:
			return nil, fmt.Errorf("unknown string_transform.mode value: %q", mode)
		}

		opts = append(opts, luadata.WithStringTransform(transform))
	}

	return opts, nil
}

// resultJSON is the response envelope.
type resultJSON struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func marshalResult(r resultJSON) *C.char {
	b, _ := json.Marshal(r)
	return C.CString(string(b))
}

//export LuaDataToJSON
func LuaDataToJSON(cinput *C.char, coptions *C.char) *C.char {
	input := C.GoString(cinput)
	optionsStr := C.GoString(coptions)

	opts, err := parseOptions(optionsStr)
	if err != nil {
		return marshalResult(resultJSON{Error: err.Error()})
	}

	reader, err := luadata.TextToJSON("input", input, opts...)
	if err != nil {
		return marshalResult(resultJSON{Error: err.Error()})
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return marshalResult(resultJSON{Error: err.Error()})
	}

	return marshalResult(resultJSON{Result: string(result)})
}

//export LuaDataFree
func LuaDataFree(p *C.char) {
	C.free(unsafe.Pointer(p))
}

func main() {}
