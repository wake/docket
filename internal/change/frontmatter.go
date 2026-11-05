package change

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/wake/docket/internal/markdown"
)

// Front is a proposal's front matter as written: values are not validated here.
type Front struct {
	Fields    map[string]Field  // top-level keys other than specs
	Specs     map[string]string // capability path -> name
	SpecsLine int               // 1-based line of "specs:", 0 when absent
}

// Field is one top-level value and its 1-based line in the file.
type Field struct {
	Value string
	Line  int
}

var frontKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// ReadFrontMatter reads the front matter of proposal.md. The bool is false,
// with a nil error, when the first line is not "---".
func ReadFrontMatter(b []byte) (Front, bool, error) {
	lines := markdown.Lines(b)
	if lines[0] != "---" {
		return Front{}, false, nil
	}
	fm := Front{Fields: map[string]Field{}, Specs: map[string]string{}}
	lastKey := ""
	for i := 1; i < len(lines); i++ {
		line, n := lines[i], i+1
		if line == "---" {
			return fm, true, nil
		}
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || trimmed[0] == '#' {
			continue
		}
		fail := func(format string, a ...any) (Front, bool, error) {
			return Front{}, true, fmt.Errorf("front matter line %d: %s", n, fmt.Sprintf(format, a...))
		}
		if line[0] == ' ' {
			if lastKey != "specs" {
				return fail("indented line outside specs")
			}
			path, rest, found := strings.Cut(trimmed, ":")
			if !found || path == "" || strings.ContainsAny(path, " \t") {
				return fail("expected <capability>: <name>")
			}
			name, err := frontValue(rest)
			if err != nil {
				return fail("%s", err)
			}
			if _, dup := fm.Specs[path]; dup {
				return fail("capability %s given twice", path)
			}
			fm.Specs[path] = name
			continue
		}
		key, rest, found := strings.Cut(line, ":")
		if !found || !frontKey.MatchString(key) {
			return fail("expected key: value")
		}
		if _, dup := fm.Fields[key]; dup || (key == "specs" && fm.SpecsLine != 0) {
			return fail("key %s given twice", key)
		}
		lastKey = key
		if key == "specs" {
			if strings.TrimSpace(rest) != "" {
				return fail("specs takes no value")
			}
			fm.SpecsLine = n
			continue
		}
		v, err := frontValue(rest)
		if err != nil {
			return fail("%s", err)
		}
		fm.Fields[key] = Field{Value: v, Line: n}
	}
	return Front{}, true, fmt.Errorf("front matter line %d: no closing ---", len(lines))
}

// frontValue reads the part after "key:" (it must start with a space or be empty).
func frontValue(rest string) (string, error) {
	if rest != "" && rest[0] != ' ' {
		return "", fmt.Errorf("expected a space after the colon")
	}
	s := strings.TrimLeft(rest, " ")
	if s == "" || s[0] == '#' {
		return "", nil
	}
	switch s[0] {
	case '"':
		return quoted(s, '"')
	case '\'':
		return quoted(s, '\'')
	}
	if i := strings.Index(s, " #"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s), nil
}

// quoted reads a quoted value that starts at s[0] and checks what follows it.
func quoted(s string, q byte) (string, error) {
	var out strings.Builder
	i := 1
	for {
		if i >= len(s) {
			return "", fmt.Errorf("unclosed quote")
		}
		c := s[i]
		switch {
		case c == q && q == '\'' && i+1 < len(s) && s[i+1] == '\'':
			out.WriteByte('\'')
			i += 2
			continue
		case c == q:
			rest := s[i+1:]
			if t := strings.TrimLeft(rest, " "); t != "" && (t[0] != '#' || t == rest) {
				return "", fmt.Errorf("unexpected text after the closing quote")
			}
			return out.String(), nil
		case c == '\\' && q == '"':
			r, n, err := unescape(s[i+1:])
			if err != nil {
				return "", err
			}
			out.WriteRune(r)
			i += 1 + n
			continue
		}
		out.WriteByte(c)
		i++
	}
}

// yamlEscapes is YAML's set of one-character escapes.
var yamlEscapes = map[byte]rune{
	'0': 0, 'a': 7, 'b': 8, 't': 9, 'n': 10, 'v': 11, 'f': 12, 'r': 13, 'e': 0x1b,
	' ': ' ', '"': '"', '/': '/', '\\': '\\', 'N': 0x85, '_': 0xa0, 'L': 0x2028, 'P': 0x2029,
}

// unescape reads the escape after a backslash and returns its rune and length.
func unescape(s string) (rune, int, error) {
	if s == "" {
		return 0, 0, fmt.Errorf("unclosed quote")
	}
	if r, ok := yamlEscapes[s[0]]; ok {
		return r, 1, nil
	}
	digits := map[byte]int{'x': 2, 'u': 4, 'U': 8}[s[0]]
	if digits == 0 {
		return 0, 0, fmt.Errorf("unknown escape \\%c", s[0])
	}
	if len(s) < 1+digits {
		return 0, 0, fmt.Errorf("short \\%c escape", s[0])
	}
	h := s[1 : 1+digits]
	if h[0] == '+' || h[0] == '-' {
		return 0, 0, fmt.Errorf("bad \\%c escape", s[0])
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil || !utf8.ValidRune(rune(v)) {
		return 0, 0, fmt.Errorf("bad \\%c escape", s[0])
	}
	return rune(v), 1 + digits, nil
}

var specTitle = regexp.MustCompile(`^# (.+) Specification$`)

// SpecTitle reads the title from the first line of a main spec. A name equal
// to the capability path is what OpenSpec writes, so it counts as no title.
func SpecTitle(firstLine, capability string) (string, bool) {
	m := specTitle.FindStringSubmatch(firstLine)
	if m == nil {
		return "", false
	}
	name := strings.TrimSpace(m[1])
	if name == "" || name == capability {
		return "", false
	}
	return name, true
}
