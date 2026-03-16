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

type parseConfig struct {
	stringTransform *StringTransform // nil = no transform (default)
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

func (c *parseConfig) transformString(source string) string {
	if c.stringTransform == nil || len(source) <= c.stringTransform.MaxLen {
		return source
	}
	switch c.stringTransform.Mode {
	case StringTransformTruncate:
		return source[:c.stringTransform.MaxLen]
	case StringTransformEmpty:
		return ""
	case StringTransformRedact:
		return "[redacted]"
	case StringTransformReplace:
		return c.stringTransform.Replacement
	default:
		return source
	}
}
