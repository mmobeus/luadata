//go:build js && wasm

package main

import (
	"io"
	"syscall/js"

	"github.com/mmobeus/luadata"
)

func convertWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 1 {
			return map[string]any{"error": "expected a string argument"}
		}
		input := args[0].String()
		reader, err := luadata.TextToJSON("input", input)
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
