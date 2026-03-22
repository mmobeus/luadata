package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Section struct {
	Type string // "prose", "lua", "go", "js", "bash", "json", "output"
	Body string
}

type Example struct {
	Title    string
	Slug     string
	Group    string
	Options  string // JSON string for WASM conversion options, empty = default
	Sections []Section
	Prev     *Example
	Next     *Example
}

// Row pairs prose (left column) with code/output (right column) for the
// two-column layout. Rows with only one side set render full-width.
type Row struct {
	Left  []Section // prose sections
	Right []Section // code/output sections
}

func (r Row) HasLeft() bool  { return len(r.Left) > 0 }
func (r Row) HasRight() bool { return len(r.Right) > 0 }
func (r Row) FullWidth() bool {
	// Only code-only rows (no prose) go full-width.
	// Prose-only rows stay in the left column.
	return len(r.Left) == 0 && len(r.Right) > 0
}

// Rows pairs an example's sections into two-column rows. Prose sections go
// left, code/output sections go right. A new row starts each time prose
// appears after code/output content.
func (ex *Example) Rows() []Row {
	var rows []Row
	var left, right []Section

	flush := func() {
		if len(left) > 0 || len(right) > 0 {
			rows = append(rows, Row{Left: left, Right: right})
			left = nil
			right = nil
		}
	}

	for _, s := range ex.Sections {
		if s.Type == "prose" {
			// New prose after code/output starts a new row
			if len(right) > 0 {
				flush()
			}
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}
	flush()
	return rows
}

type IndexGroup struct {
	Name     string
	Examples []*Example
}

func groupFor(index int) string {
	switch {
	case index < 6:
		return "Basics"
	case index < 14:
		return "Options"
	case index < 17:
		return "Schema"
	default:
		return "WebAssembly"
	}
}

func parseExample(path string) (*Example, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty example file: %s", path)
	}

	ex := &Example{Title: strings.TrimSpace(lines[0])}

	// Derive slug from directory name, stripping numeric prefix
	dir := filepath.Base(filepath.Dir(path))
	parts := strings.SplitN(dir, "-", 2)
	if len(parts) == 2 {
		ex.Slug = parts[1]
	} else {
		ex.Slug = dir
	}

	var currentType string
	var currentLines []string

	flush := func() {
		if currentType == "" || currentType == "options" {
			return
		}
		body := strings.Join(currentLines, "\n")
		body = strings.TrimRight(body, "\n")
		if body != "" || currentType != "prose" {
			ex.Sections = append(ex.Sections, Section{Type: currentType, Body: body})
		}
		currentLines = nil
	}

	for _, line := range lines[1:] {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			flush()
			currentType = "prose"
			continue
		}

		if strings.HasPrefix(trimmed, "---") && !strings.HasPrefix(trimmed, "----") {
			flush()
			sectionType := strings.TrimPrefix(trimmed, "---")
			if sectionType == "options" {
				// Options section: collect lines and store as example-level options
				currentType = "options"
			} else {
				currentType = sectionType
			}
			continue
		}

		if currentType == "options" {
			ex.Options += strings.TrimSpace(line)
			continue
		}

		if currentType != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	return ex, nil
}

func main() {
	outDir := flag.String("out", "bin/web/docs", "output directory")
	flag.Parse()

	genDir := filepath.Dir(mustAbs(os.Args[0]))
	// When run via `go run`, os.Args[0] is a temp path. Use working dir instead.
	// Find the gen directory by looking for templates/ relative to the source.
	srcDir := findSrcDir()

	// Read examples
	examplesDir := filepath.Join(srcDir, "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading examples: %v (genDir=%s)\n", err, genDir)
		os.Exit(1)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)

	var examples []*Example
	for i, dir := range dirs {
		path := filepath.Join(examplesDir, dir, "example.txt")
		ex, err := parseExample(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parsing %s: %v\n", path, err)
			os.Exit(1)
		}
		ex.Group = groupFor(i)
		examples = append(examples, ex)
	}

	// Link prev/next
	for i := range examples {
		if i > 0 {
			examples[i].Prev = examples[i-1]
		}
		if i < len(examples)-1 {
			examples[i].Next = examples[i+1]
		}
	}

	// Build groups for index
	var groups []IndexGroup
	var currentGroup string
	for _, ex := range examples {
		if ex.Group != currentGroup {
			groups = append(groups, IndexGroup{Name: ex.Group})
			currentGroup = ex.Group
		}
		groups[len(groups)-1].Examples = append(groups[len(groups)-1].Examples, ex)
	}

	// Load templates
	tmplDir := filepath.Join(srcDir, "templates")
	funcMap := template.FuncMap{
		"hasPrefix": func(s template.HTML, prefix string) bool {
			return strings.HasPrefix(string(s), prefix)
		},
		"isCode": func(t string) bool {
			return t != "prose" && t != "output"
		},
		"hlClass": func(t string) string {
			// Map section type to highlight.js language class
			switch t {
			case "js":
				return "language-javascript"
			default:
				return "language-" + t
			}
		},
		"looksLikeJSON": func(s string) bool {
			trimmed := strings.TrimSpace(s)
			return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
		},
		"split": func(s string) []template.HTML {
			// Split prose into paragraphs on blank lines, with support
			// for markdown-style bullet lists (lines starting with "- ").
			// Also wraps standalone "luadata" mentions with a styled span
			// and converts [text](url) links to HTML <a> tags.
			var paragraphs []template.HTML
			var current []string
			var listItems []string

			processProse := func(text string) string {
				return markLinks(markCode(markBold(markToolName(template.HTMLEscapeString(text)))))
			}

			flushParagraph := func() {
				if len(current) > 0 {
					paragraphs = append(paragraphs, template.HTML(processProse(strings.Join(current, " ")))) //nolint:gosec // content is escaped above
					current = nil
				}
			}

			flushList := func() {
				if len(listItems) > 0 {
					var buf strings.Builder
					buf.WriteString("<ul>")
					for _, item := range listItems {
						buf.WriteString("<li>")
						buf.WriteString(processProse(item))
						buf.WriteString("</li>")
					}
					buf.WriteString("</ul>")
					paragraphs = append(paragraphs, template.HTML(buf.String())) //nolint:gosec // content is escaped above
					listItems = nil
				}
			}

			for _, line := range strings.Split(s, "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					flushParagraph()
					flushList()
				} else if strings.HasPrefix(trimmed, "- ") {
					flushParagraph()
					listItems = append(listItems, strings.TrimPrefix(trimmed, "- "))
				} else if len(listItems) > 0 && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t")) {
					// Continuation of previous list item
					listItems[len(listItems)-1] += " " + trimmed
				} else {
					flushList()
					current = append(current, trimmed)
				}
			}
			flushParagraph()
			flushList()
			return paragraphs
		},
	}

	indexTmpl, err := template.New("index.html.tmpl").Funcs(funcMap).ParseFiles(filepath.Join(tmplDir, "index.html.tmpl"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing index template: %v\n", err)
		os.Exit(1)
	}

	exampleTmpl, err := template.New("example.html.tmpl").Funcs(funcMap).ParseFiles(filepath.Join(tmplDir, "example.html.tmpl"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parsing example template: %v\n", err)
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "creating output dir: %v\n", err)
		os.Exit(1)
	}

	// Render index
	indexFile, err := os.Create(filepath.Join(*outDir, "index.html"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating index.html: %v\n", err)
		os.Exit(1)
	}
	if err := indexTmpl.Execute(indexFile, groups); err != nil {
		fmt.Fprintf(os.Stderr, "rendering index: %v\n", err)
		os.Exit(1)
	}
	_ = indexFile.Close()

	// Render each example
	for _, ex := range examples {
		f, err := os.Create(filepath.Join(*outDir, ex.Slug+".html"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "creating %s.html: %v\n", ex.Slug, err)
			os.Exit(1)
		}
		if err := exampleTmpl.Execute(f, ex); err != nil {
			fmt.Fprintf(os.Stderr, "rendering %s: %v\n", ex.Slug, err)
			os.Exit(1)
		}
		_ = f.Close()
	}

	// Copy static assets
	for _, asset := range []string{"style.css", "interactive.js"} {
		data, err := os.ReadFile(filepath.Join(tmplDir, asset))
		if err != nil {
			fmt.Fprintf(os.Stderr, "reading %s: %v\n", asset, err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(*outDir, asset), data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", asset, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Generated %d examples in %s\n", len(examples), *outDir)
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// markToolName wraps standalone occurrences of "luadata" in prose text with
// a <span class="tool-name"> tag for styling. It avoids matching inside HTML
// tags or when "luadata" is part of a longer word.
// markCode converts markdown-style `text` to HTML <code> tags.
// Must be called after HTML-escaping.
func markCode(s string) string {
	var result strings.Builder
	remaining := s
	for {
		start := strings.Index(remaining, "`")
		if start == -1 {
			result.WriteString(remaining)
			break
		}
		end := strings.Index(remaining[start+1:], "`")
		if end == -1 {
			result.WriteString(remaining)
			break
		}
		end += start + 1
		result.WriteString(remaining[:start])
		result.WriteString("<code>")
		result.WriteString(remaining[start+1 : end])
		result.WriteString("</code>")
		remaining = remaining[end+1:]
	}
	return result.String()
}

// markBold converts markdown-style **text** to HTML <strong> tags.
// Must be called after HTML-escaping.
func markBold(s string) string {
	var result strings.Builder
	remaining := s
	for {
		start := strings.Index(remaining, "**")
		if start == -1 {
			result.WriteString(remaining)
			break
		}
		end := strings.Index(remaining[start+2:], "**")
		if end == -1 {
			result.WriteString(remaining)
			break
		}
		end += start + 2
		result.WriteString(remaining[:start])
		result.WriteString("<strong>")
		result.WriteString(remaining[start+2 : end])
		result.WriteString("</strong>")
		remaining = remaining[end+2:]
	}
	return result.String()
}

// markLinks converts markdown-style [text](url) links to HTML <a> tags.
// Must be called after HTML-escaping (the []() chars are not HTML-special).
func markLinks(s string) string {
	var result strings.Builder
	remaining := s
	for {
		openBracket := strings.Index(remaining, "[")
		if openBracket == -1 {
			result.WriteString(remaining)
			break
		}
		closeBracket := strings.Index(remaining[openBracket:], "](")
		if closeBracket == -1 {
			result.WriteString(remaining)
			break
		}
		closeBracket += openBracket
		closeParen := strings.Index(remaining[closeBracket+2:], ")")
		if closeParen == -1 {
			result.WriteString(remaining)
			break
		}
		closeParen += closeBracket + 2

		text := remaining[openBracket+1 : closeBracket]
		url := remaining[closeBracket+2 : closeParen]
		result.WriteString(remaining[:openBracket])
		result.WriteString(`<a href="`)
		result.WriteString(url)
		result.WriteString(`">`)
		result.WriteString(text)
		result.WriteString(`</a>`)
		remaining = remaining[closeParen+1:]
	}
	return result.String()
}

func markToolName(s string) string {
	var result strings.Builder
	remaining := s
	for {
		idx := strings.Index(remaining, "luadata")
		if idx == -1 {
			result.WriteString(remaining)
			break
		}
		// Check that it's not part of a longer word
		before := idx > 0 && isWordChar(remaining[idx-1])
		after := idx+7 < len(remaining) && isWordChar(remaining[idx+7])
		if before || after {
			result.WriteString(remaining[:idx+7])
			remaining = remaining[idx+7:]
			continue
		}
		result.WriteString(remaining[:idx])
		result.WriteString(`<span class="tool-name">luadata</span>`)
		remaining = remaining[idx+7:]
	}
	return result.String()
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func findSrcDir() string {
	// Try to find the gen source directory. When run via `go run ./web/docs/gen`,
	// the working directory is the module root. When run via `cd web/docs/gen && go run .`,
	// the working directory is already the gen directory.
	candidates := []string{
		".",
		"web/docs/gen",
		filepath.Join(filepath.Dir(mustAbs(os.Args[0])), "web/docs/gen"),
	}
	for _, c := range candidates {
		if info, err := os.Stat(filepath.Join(c, "templates")); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	fmt.Fprintln(os.Stderr, "cannot find gen source directory (templates/examples)")
	os.Exit(1)
	return ""
}
