//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/mmobeus/luadata"
)

func convertWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) < 1 {
			return map[string]any{"error": "expected a string argument"}
		}
		input := args[0].String()
		result, err := luadata.ToJSON([]byte(input))
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
