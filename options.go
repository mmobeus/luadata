package luadata

// StringTransformMode defines how strings exceeding MaxLen are transformed.
type StringTransformMode int

const (
	StringTransformTruncate StringTransformMode = iota // truncate to MaxLen
	StringTransformEmpty                               // replace with ""
	StringTransformRedact                              // replace with "[redacted]"
	StringTransformReplace                             // replace with custom string
)

// StringTransform configures how long strings are handled during parsing.
type StringTransform struct {
	MaxLen      int
	Mode        StringTransformMode
	Replacement string // only used with StringTransformReplace
}

// ArrayMode controls how Lua tables with integer keys are rendered in JSON.
// Use one of the concrete types: ArrayModeNone, ArrayModeIndexOnly, or ArrayModeSparse.
type ArrayMode interface {
	arrayMode()
}

// ArrayModeNone disables all array rendering. Every table becomes a JSON object,
// including implicit index tables like {"a","b","c"}.
type ArrayModeNone struct{}

// ArrayModeIndexOnly renders only implicit index tables ({"a","b","c"}) as JSON
// arrays. Tables with explicit integer keys ([1]="a") always render as objects.
type ArrayModeIndexOnly struct{}

// ArrayModeSparse renders tables with explicit integer keys as JSON arrays when
// the maximum gap between consecutive keys (including the gap from 0 to the first
// key) does not exceed MaxGap. Missing indices are filled with null. Implicit
// index tables are always rendered as arrays regardless of MaxGap. A MaxGap of 0
// means only contiguous integer keys (starting at 1) are rendered as arrays.
type ArrayModeSparse struct {
	MaxGap int
}

func (ArrayModeNone) arrayMode()      {}
func (ArrayModeIndexOnly) arrayMode() {}
func (ArrayModeSparse) arrayMode()    {}

type parseConfig struct {
	stringTransform *StringTransform // nil = no transform (default)
	arrayMode       ArrayMode        // nil = default (ArrayModeSparse{MaxGap: 20})
}

func (c *parseConfig) effectiveArrayMode() ArrayMode {
	if c == nil || c.arrayMode == nil {
		return ArrayModeSparse{MaxGap: 20}
	}
	return c.arrayMode
}

// Option configures parsing behavior.
type Option func(*parseConfig)

// WithStringTransform sets a string transform to apply to parsed string values.
// Strings exceeding MaxLen are replaced in both Source and Raw as if the
// transformed value was the original. Value.Transformed is set to true when
// this occurs.
func WithStringTransform(st StringTransform) Option {
	return func(c *parseConfig) {
		c.stringTransform = &st
	}
}

// WithArrayDetection sets the array rendering mode for JSON output. The default
// (when this option is not used) is ArrayModeSparse{MaxGap: 20}.
func WithArrayDetection(mode ArrayMode) Option {
	return func(c *parseConfig) {
		c.arrayMode = mode
	}
}

func (c *parseConfig) transformString(source string) (string, bool) {
	if c.stringTransform == nil || len(source) <= c.stringTransform.MaxLen {
		return source, false
	}
	switch c.stringTransform.Mode {
	case StringTransformTruncate:
		return source[:c.stringTransform.MaxLen], true
	case StringTransformEmpty:
		return "", true
	case StringTransformRedact:
		return "[redacted]", true
	case StringTransformReplace:
		return c.stringTransform.Replacement, true
	default:
		return source, false
	}
}
