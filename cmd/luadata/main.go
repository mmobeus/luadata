package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/mmobeus/luadata"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "tojson":
		toJSON()
	case "validate":
		validate()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: luadata <command> [options] [args]

Commands:
  tojson <file>     Convert a Lua data file to JSON (use - for stdin)
  validate <file>   Check that a Lua data file parses successfully

Run 'luadata <command> --help' for command-specific options.
`)
}

type flagValues struct {
	emptyTable        string
	arrayMode         string
	arrayMaxGap       int
	stringMaxLen      int
	stringMode        string
	stringReplacement string
}

func defineFlags(name string) (*flag.FlagSet, *flagValues) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	v := &flagValues{}

	fs.StringVar(&v.emptyTable, "empty-table", "null", "how to render empty tables: null, omit, array, object")
	fs.StringVar(&v.arrayMode, "array-mode", "sparse", "array detection mode: none, index-only, sparse")
	fs.IntVar(&v.arrayMaxGap, "array-max-gap", 20, "max gap between keys for sparse array detection")
	fs.IntVar(&v.stringMaxLen, "string-max-len", 0, "max string length before transform (0 = disabled)")
	fs.StringVar(&v.stringMode, "string-mode", "truncate", "string transform mode: truncate, empty, redact, replace")
	fs.StringVar(&v.stringReplacement, "string-replacement", "", "replacement text for replace mode")

	return fs, v
}

func buildOptions(v *flagValues) ([]luadata.Option, error) {
	var opts []luadata.Option

	switch v.emptyTable {
	case "null":
		opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableNull))
	case "omit":
		opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableOmit))
	case "array":
		opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableArray))
	case "object":
		opts = append(opts, luadata.WithEmptyTableMode(luadata.EmptyTableObject))
	default:
		return nil, fmt.Errorf("unknown --empty-table value: %q (valid: null, omit, array, object)", v.emptyTable)
	}

	switch v.arrayMode {
	case "none":
		opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeNone{}))
	case "index-only":
		opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeIndexOnly{}))
	case "sparse":
		opts = append(opts, luadata.WithArrayDetection(luadata.ArrayModeSparse{MaxGap: v.arrayMaxGap}))
	default:
		return nil, fmt.Errorf("unknown --array-mode value: %q (valid: none, index-only, sparse)", v.arrayMode)
	}

	if v.stringMaxLen > 0 {
		st := luadata.StringTransform{
			MaxLen: v.stringMaxLen,
		}
		switch v.stringMode {
		case "truncate":
			st.Mode = luadata.StringTransformTruncate
		case "empty":
			st.Mode = luadata.StringTransformEmpty
		case "redact":
			st.Mode = luadata.StringTransformRedact
		case "replace":
			st.Mode = luadata.StringTransformReplace
			st.Replacement = v.stringReplacement
		default:
			return nil, fmt.Errorf("unknown --string-mode value: %q (valid: truncate, empty, redact, replace)", v.stringMode)
		}
		opts = append(opts, luadata.WithStringTransform(st))
	}

	return opts, nil
}

func toJSON() {
	fs, v := defineFlags("tojson")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: luadata tojson [options] <file>\n\nOptions:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	opts, err := buildOptions(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var result io.Reader

	if fs.Arg(0) == "-" {
		result, err = luadata.ReaderToJSON("stdin", os.Stdin, opts...)
	} else {
		result, err = luadata.FileToJSON(fs.Arg(0), opts...)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting: %v\n", err)
		os.Exit(1)
	}

	if _, err = io.Copy(os.Stdout, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println()
}

func validate() {
	fs, v := defineFlags("validate")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: luadata validate [options] <file>\n\nOptions:\n")
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}

	opts, err := buildOptions(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	_, err = luadata.FileToJSON(fs.Arg(0), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error validating %s: %v\n", fs.Arg(0), err)
		os.Exit(1)
	}
}
