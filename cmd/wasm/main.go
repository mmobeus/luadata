//go:build js && wasm

package main

import (
	"fmt"
	"io"
	"syscall/js"

	"github.com/mmobeus/luadata"
)

func parseJSOptions(v js.Value) ([]luadata.Option, error) {
	if v.IsUndefined() || v.IsNull() {
		return nil, nil
	}

	var opts []luadata.Option

	if et := v.Get("emptyTable"); !et.IsUndefined() && !et.IsNull() {
		switch et.String() {
		case "null":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableNull))
		case "omit":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableOmit))
		case "array":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableArray))
		case "object":
			opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableObject))
		default:
			return nil, fmt.Errorf("unknown emptyTable value: %q", et.String())
		}
	}

	if am := v.Get("arrayMode"); !am.IsUndefined() && !am.IsNull() {
		switch am.String() {
		case "none":
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeNone{}))
		case "index-only":
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeIndexOnly{}))
		case "sparse":
			maxGap := 20
			if mg := v.Get("arrayMaxGap"); !mg.IsUndefined() && !mg.IsNull() {
				maxGap = mg.Int()
			}
			opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeSparse{MaxGap: maxGap}))
		default:
			return nil, fmt.Errorf("unknown arrayMode value: %q", am.String())
		}
	}

	if st := v.Get("stringTransform"); !st.IsUndefined() && !st.IsNull() {
		ml := st.Get("maxLen")
		if ml.IsUndefined() || ml.IsNull() || ml.Int() <= 0 {
			return nil, fmt.Errorf("stringTransform.maxLen must be a positive number")
		}

		transform := luadata.StringTransform{
			MaxLen: ml.Int(),
		}

		mode := "truncate"
		if m := st.Get("mode"); !m.IsUndefined() && !m.IsNull() {
			mode = m.String()
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
			if r := st.Get("replacement"); !r.IsUndefined() && !r.IsNull() {
				transform.Replacement = r.String()
			}
		default:
			return nil, fmt.Errorf("unknown stringTransform.mode value: %q", mode)
		}

		opts = append(opts, luadata.WithStringTransform(transform))
	}

	return opts, nil
}

func convertWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 1 {
			return map[string]any{"error": "expected a string argument"}
		}
		input := args[0].String()

		var parseOpts []luadata.Option
		if len(args) >= 2 {
			var err error
			parseOpts, err = parseJSOptions(args[1])
			if err != nil {
				return map[string]any{"error": err.Error()}
			}
		}

		reader, err := luadata.TextToJSON("input", input, parseOpts...)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		result, err := io.ReadAll(reader)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"result": string(result)}
	})
}

func main() {
	js.Global().Set("convertLuaDataToJson", convertWrapper())

	// Block forever so the Go runtime stays alive.
	select {}
}
