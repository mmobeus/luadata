package luadata

// Option configures conversion behavior.
type Option func(*optionsConfig)

type optionsConfig struct {
	emptyTable      string           // "null", "omit", "array", "object"
	arrayMode       string           // "none", "index-only", "sparse"
	arrayMaxGap     *int             // only for sparse mode
	stringTransform *stringTransform // nil = no transform
	schema          string           // JSON Schema string
	unknownFields   string           // "ignore", "include", "fail"
}

type stringTransform struct {
	MaxLen      int    `json:"max_len"`
	Mode        string `json:"mode,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// WithEmptyTableMode sets how empty Lua tables ({}) are rendered in JSON output.
// Valid values: "null" (default), "omit", "array", "object".
func WithEmptyTableMode(mode string) Option {
	return func(c *optionsConfig) {
		c.emptyTable = mode
	}
}

// WithArrayMode sets the array detection mode for JSON output.
// Valid values: "none", "index-only", "sparse" (default).
// For sparse mode, use WithArrayMaxGap to set the gap threshold.
func WithArrayMode(mode string, maxGap ...int) Option {
	return func(c *optionsConfig) {
		c.arrayMode = mode
		if len(maxGap) > 0 {
			gap := maxGap[0]
			c.arrayMaxGap = &gap
		}
	}
}

// WithSchema sets a JSON Schema to guide parsing.
// The schema is a standard JSON Schema string (subset: type, properties, items, format).
func WithSchema(schemaJSON string) Option {
	return func(c *optionsConfig) {
		c.schema = schemaJSON
	}
}

// WithUnknownFieldMode sets how fields not in the schema are handled.
// Valid values: "ignore" (default), "include", "fail".
func WithUnknownFieldMode(mode string) Option {
	return func(c *optionsConfig) {
		c.unknownFields = mode
	}
}

// WithStringTransform configures how long strings are handled.
// Strings exceeding maxLen are transformed according to mode.
// Valid modes: "truncate" (default), "empty", "redact", "replace".
// For "replace" mode, pass the replacement string.
func WithStringTransform(maxLen int, mode string, replacement ...string) Option {
	return func(c *optionsConfig) {
		st := &stringTransform{
			MaxLen: maxLen,
			Mode:   mode,
		}
		if len(replacement) > 0 {
			st.Replacement = replacement[0]
		}
		c.stringTransform = st
	}
}
