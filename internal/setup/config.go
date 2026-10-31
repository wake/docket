package setup

import (
	"errors"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ConfigSchema says what openspec/config.yaml names as the project's schema.
type ConfigSchema string

const (
	ConfigDocket  ConfigSchema = "docket"
	ConfigNoFile  ConfigSchema = "no-file"
	ConfigDefault ConfigSchema = "default" // no schema key, or spec-driven
	ConfigOther   ConfigSchema = "other"
)

// configFile is the OpenSpec configuration file, relative to the repository
// root.
const configFile = "openspec/config.yaml"

// schemaKey finds the top-level schema key in b: its value, the byte span of
// the value (quotes included) and the 0-based line. line is -1 with no key.
func schemaKey(b []byte) (value string, start, end, line int, err error) {
	line = -1
	off := 0
	for i, l := range strings.SplitAfter(string(b), "\n") {
		lineOff := off
		off += len(l)
		if !strings.HasPrefix(l, "schema:") {
			continue
		}
		body := strings.TrimSuffix(strings.TrimSuffix(l, "\n"), "\r")
		rest := body[len("schema:"):]
		if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
			continue // another key, such as schema:x
		}
		if line >= 0 {
			return "", 0, 0, 0, errors.New("the schema key appears twice")
		}
		v := strings.TrimLeft(rest, " \t")
		s := lineOff + len(body) - len(v)
		var n int
		switch {
		case v == "" || v[0] == '#':
			return "", 0, 0, 0, errors.New("the schema key has no value on its line")
		case v[0] == '"' || v[0] == '\'':
			c := strings.IndexByte(v[1:], v[0])
			if c < 0 {
				return "", 0, 0, 0, errors.New("the schema value has no closing quote")
			}
			value, n = v[1:1+c], c+2
		default:
			n = strings.IndexAny(v, " \t")
			if n < 0 {
				n = len(v)
			}
			value = v[:n]
		}
		if value == "" {
			return "", 0, 0, 0, errors.New("the schema value is empty")
		}
		if tail := strings.TrimLeft(v[n:], " \t"); tail != "" && tail[0] != '#' {
			return "", 0, 0, 0, errors.New("the schema key is followed by more than its value")
		}
		line, start, end = i, s, s+n
	}
	if line < 0 {
		value = ""
	}
	if err := crossCheck(b, value, line >= 0); err != nil {
		return "", 0, 0, 0, err
	}
	return value, start, end, line, nil
}

// crossCheck asks the YAML parser what the line reader found: any other
// spelling of a top-level schema key (quoted, spaced, in a flow mapping)
// would be missed by the reader, and NameDocket would add a second key.
func crossCheck(b []byte, value string, found bool) error {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return errors.New("the file is not valid YAML: " + err.Error())
	}
	if doc.Kind == 0 {
		return nil // empty file or only comments
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode {
		return errors.New("the file is not a YAML mapping")
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "schema" && (!found || m.Content[i+1].Value != value) {
			return errors.New("the schema key is not one plain top-level line")
		}
	}
	return nil
}

// SchemaKey returns the top-level schema key of a configuration file and its
// 0-based line, or -1 with no key. A key line starts with schema: in column
// 0; its value is a bare word or a quoted one on the same line, optionally
// followed by a comment. Anything else is an error.
func SchemaKey(b []byte) (value string, line int, err error) {
	value, _, _, line, err = schemaKey(b)
	return value, line, err
}

// NameDocket returns b with the schema key's value replaced by docket, every
// other byte kept; without a key it puts schema: docket on a first line.
func NameDocket(b []byte) ([]byte, error) {
	_, start, end, line, err := schemaKey(b)
	if err != nil {
		return nil, err
	}
	if line < 0 {
		return append([]byte("schema: docket\n"), b...), nil
	}
	out := append([]byte{}, b[:start]...)
	out = append(out, "docket"...)
	return append(out, b[end:]...), nil
}

// decideConfig settles the openspec/config.yaml target; got is its current
// content (nil when missing).
func decideConfig(d decided, got []byte) (decided, *Problem) {
	if !d.existed {
		d.status, d.data, d.config = Written, []byte("schema: docket\n"), ConfigNoFile
		return d, nil
	}
	value, line, err := SchemaKey(got)
	if err != nil {
		return d, &Problem{Path: d.rel, Kind: ProblemUnreadable, Err: err}
	}
	switch {
	case line >= 0 && value == "docket":
		d.status, d.config = Unchanged, ConfigDocket
	case line < 0 || value == "spec-driven":
		next, _ := NameDocket(got)
		d.status, d.data, d.config = Written, next, ConfigDefault
	default:
		d.status, d.config = Differs, ConfigOther
	}
	return d, nil
}
