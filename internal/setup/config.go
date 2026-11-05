package setup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"regexp"
	"slices"
	"strings"
)

const (
	configPath = "openspec/config.yaml"
	opsKey     = configPath + "#operations"
)

// ConfigState is what openspec/config.yaml says, for dkt inspect.
// Schema is docket, default (no top-level schema, or spec-driven) or other.
// Guidance is same, missing, outdated (an earlier release's section) or differs.
type ConfigState struct {
	Schema   string
	Guidance string
}

// readShippedOps reads the shipped operations section, trailing blank lines trimmed.
func readShippedOps(shipped fs.FS) ([]byte, error) {
	b, err := fs.ReadFile(shipped, "config/operations.yaml")
	if err != nil {
		return nil, err
	}
	return []byte(trimBlank(string(b))), nil
}

// trimBlank drops trailing blank lines and makes s end with one newline.
func trimBlank(s string) string {
	return strings.TrimRight(s, " \t\r\n") + "\n"
}

func isBlank(l string) bool { return strings.TrimSpace(l) == "" }

func isIndented(l string) bool { return l != "" && (l[0] == ' ' || l[0] == '\t') }

var docEnd = regexp.MustCompile(`^\.\.\.([ \t#]|$)`)

var spacedKey = regexp.MustCompile(`^(schema|operations)[ \t]+:`)

// parsedConfig is what the line reader found. A line index of -1 means not found.
type parsedConfig struct {
	lines                  []string
	offs                   []int // byte offset of each line, plus the total length
	schema                 string
	schemaStart, schemaEnd int // byte span of the schema value, quotes included
	schemaLine             int
	opStart, opEnd         int // line range of the section, trailing blank lines excluded
}

func parseConfig(b []byte) (*parsedConfig, error) {
	p := &parsedConfig{schemaLine: -1, opStart: -1}
	if bytes.HasPrefix(b, []byte("\xef\xbb\xbf")) {
		return nil, errors.New("the file starts with a byte order mark")
	}
	p.lines = strings.SplitAfter(string(b), "\n")
	if p.lines[len(p.lines)-1] == "" {
		p.lines = p.lines[:len(p.lines)-1]
	}
	off := 0
	for i, l := range p.lines {
		p.offs = append(p.offs, off)
		lineOff := off
		off += len(l)
		body := strings.TrimSuffix(strings.TrimSuffix(l, "\n"), "\r")
		if body != "" && !isIndented(body) && body[0] != '#' {
			if strings.ContainsRune("{[\"'?%", rune(body[0])) || strings.HasPrefix(body, "---") || docEnd.MatchString(body) {
				return nil, errors.New("a line at the first column starts with " + body[:1] + ", which is not a plain top-level entry")
			}
			if spacedKey.MatchString(body) {
				return nil, errors.New("a key is followed by spaces before its colon")
			}
		}
		switch {
		case strings.HasPrefix(body, "schema:"):
			if err := p.readSchema(i, lineOff, body); err != nil {
				return nil, err
			}
		case strings.HasPrefix(body, "operations:"):
			rest := body[len("operations:"):]
			if rest != "" && !isIndented(rest) {
				continue // another key
			}
			if p.opStart >= 0 {
				return nil, errors.New("the operations key appears twice")
			}
			if v := strings.TrimLeft(rest, " \t"); v != "" && v[0] != '#' {
				return nil, errors.New("the operations key has a value on its line")
			}
			p.opStart = i
		}
	}
	p.offs = append(p.offs, off)
	if p.opStart >= 0 {
		j := p.opStart + 1
		for j < len(p.lines) && (isBlank(p.lines[j]) || isIndented(p.lines[j])) {
			j++
		}
		for j-1 > p.opStart && isBlank(p.lines[j-1]) {
			j--
		}
		p.opEnd = j
	}
	return p, nil
}

func (p *parsedConfig) readSchema(i, lineOff int, body string) error {
	rest := body[len("schema:"):]
	if rest != "" && !isIndented(rest) {
		return nil // another key, such as schema:x
	}
	if p.schemaLine >= 0 {
		return errors.New("the schema key appears twice")
	}
	v := strings.TrimLeft(rest, " \t")
	start := lineOff + len(body) - len(v)
	var n int
	switch {
	case v == "" || v[0] == '#':
		return errors.New("the schema key has no value on its line")
	case v[0] == '"' || v[0] == '\'':
		c := strings.IndexByte(v[1:], v[0])
		if c < 0 {
			return errors.New("the schema value has no closing quote")
		}
		p.schema, n = v[1:1+c], c+2
	default:
		if n = strings.IndexAny(v, " \t"); n < 0 {
			n = len(v)
		}
		p.schema = v[:n]
	}
	if p.schema == "" {
		return errors.New("the schema value is empty")
	}
	if tail := strings.TrimLeft(v[n:], " \t"); tail != "" && tail[0] != '#' {
		return errors.New("the schema key is followed by more than its value")
	}
	if v[0] == '"' || v[0] == '\'' { // replace inside the quotes
		start, n = start+1, n-2
	}
	p.schemaLine, p.schemaStart, p.schemaEnd = i, start, start+n
	return nil
}

func (p *parsedConfig) section() string {
	return trimBlank(strings.ReplaceAll(strings.Join(p.lines[p.opStart:p.opEnd], ""), "\r\n", "\n"))
}

// planConfig works out the bytes openspec/config.yaml should have, given its
// current bytes (empty when missing) and the shipped section ops. It says what
// the file holds now; a schema other than default and guidance that differs
// are never changed.
func planConfig(got, ops []byte) ([]byte, ConfigState, error) {
	if len(got) == 0 {
		return append([]byte("schema: docket\n\n"), ops...), ConfigState{"default", "missing"}, nil
	}
	p, err := parseConfig(got)
	if err != nil {
		return nil, ConfigState{}, err
	}
	st := ConfigState{Schema: "default", Guidance: "missing"}
	switch p.schema {
	case "docket":
		st.Schema = "docket"
	case "", "spec-driven":
	default:
		st.Schema = "other"
	}
	out := got
	switch {
	case p.opStart < 0:
		out = slices.Clone(got)
		if !bytes.HasSuffix(out, []byte("\n")) {
			out = append(out, '\n')
		}
		if last := p.lines[len(p.lines)-1]; !isBlank(last) {
			out = append(out, '\n')
		}
		out = append(out, ops...)
	default:
		sec := p.section()
		sum := sha256.Sum256([]byte(sec))
		switch {
		case sec == string(ops):
			st.Guidance = "same"
		case slices.Contains(earlier[opsKey], hex.EncodeToString(sum[:])):
			st.Guidance = "outdated"
			out = append(append(slices.Clone(got[:p.offs[p.opStart]]), ops...), got[p.offs[p.opEnd]:]...)
		default:
			st.Guidance = "differs"
		}
	}
	if st.Schema == "default" {
		if p.schemaLine < 0 {
			out = append([]byte("schema: docket\n"), out...)
		} else {
			// the schema line precedes or follows the section; offsets shift only after the section
			q, err := parseConfig(out)
			if err != nil {
				return nil, ConfigState{}, err
			}
			out = append(append(slices.Clone(out[:q.schemaStart]), "docket"...), out[q.schemaEnd:]...)
		}
	}
	return out, st, nil
}

// decideConfig settles the openspec/config.yaml target; got is its current
// content, ops the shipped section.
func decideConfig(d Decision, got, ops []byte) (Decision, *Problem) {
	out, st, err := planConfig(got, ops)
	if err != nil {
		return d, &Problem{Path: d.Path, Kind: ProblemUnreadable, Err: err}
	}
	d.Config, d.data = &st, out
	changed := !bytes.Equal(out, got)
	kept := st.Schema == "other" || st.Guidance == "differs"
	switch {
	case !d.existed:
		d.Status, d.Action = StatusMissing, ActionCreate
	case kept && changed:
		d.Status, d.Action = StatusDiffers, ActionUpdate
	case kept:
		d.Status, d.Action = StatusDiffers, ActionKeep
	case changed:
		d.Status, d.Action = StatusOutdated, ActionUpdate
	default:
		d.Status, d.Action = StatusSame, ActionNone
	}
	return d, nil
}
