package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mmobeus/luadata"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: luadata <command> [args]\n\nCommands:\n  tojson <file>   Convert a Lua data file to JSON (use - for stdin)\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "tojson":
		toJSON()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func toJSON() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: luadata tojson <file>\n")
		os.Exit(1)
	}

	var input []byte
	var err error

	if os.Args[2] == "-" {
		input, err = io.ReadAll(os.Stdin)
	} else {
		input, err = os.ReadFile(os.Args[2])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	result, err := luadata.ToJSON(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(result))
}
