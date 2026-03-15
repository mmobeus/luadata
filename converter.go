// Package luadata converts Lua data table files into JSON.
package luadata

import "encoding/json"

// ToJSON parses a Lua data table and returns the equivalent JSON bytes.
func ToJSON(lua []byte) ([]byte, error) {
	parsed, err := ParseText("input", string(lua))
	if err != nil {
		return nil, err
	}
	return json.Marshal(parsed)
}
