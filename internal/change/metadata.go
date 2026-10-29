package change

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Metadata is what a change's metadata files record. File names the file the
// values came from, or the file at fault when reading failed.
type Metadata struct {
	Title, Kind, Source, Base, Created, File string
}

// openspecFile is the part of .openspec.yaml that Docket reads.
type openspecFile struct {
	Created string `yaml:"created"`
	Kind    string `yaml:"x-kind"`
	Source  string `yaml:"x-source"`
	Base    string `yaml:"x-base"`
}

// parseMetadata reads .openspec.yaml. It fails when b is not YAML or not a
// mapping.
func parseMetadata(b []byte) (Metadata, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Metadata{}, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return Metadata{}, fmt.Errorf("not a YAML mapping")
	}
	var o openspecFile
	if err := doc.Content[0].Decode(&o); err != nil {
		return Metadata{}, err
	}
	return Metadata{Kind: o.Kind, Source: o.Source, Base: o.Base, Created: o.Created, File: OpenSpecFile}, nil
}

// ReadMetadata reads a change's metadata from its files, keyed by path
// relative to the change folder. docket.yaml wins when present; otherwise the
// x- keys of .openspec.yaml are used and there is no title. Created always
// comes from .openspec.yaml, and a failure to read it there is not an error
// when docket.yaml is present.
func ReadMetadata(files map[string][]byte) (Metadata, error) {
	ob, hasOpenSpec := files[OpenSpecFile]
	var legacy Metadata
	var legacyErr error
	if hasOpenSpec {
		legacy, legacyErr = parseMetadata(ob)
	}
	db, hasDocket := files[DocketFile]
	if !hasDocket {
		if !hasOpenSpec {
			return Metadata{File: DocketFile}, fmt.Errorf("%s is missing", DocketFile)
		}
		if legacyErr != nil {
			return Metadata{File: OpenSpecFile}, legacyErr
		}
		return legacy, nil
	}
	d, err := ParseDocket(db)
	if err != nil {
		return Metadata{File: DocketFile}, err
	}
	m := Metadata{Title: d.Title, Kind: d.Kind, Source: d.Source, Base: d.Base, File: DocketFile}
	if legacyErr == nil {
		m.Created = legacy.Created
	}
	return m, nil
}

// SetKey replaces the value of the one top-level `key:` line and keeps every
// other byte, including the line ending. With no such line it appends one,
// adding a line ending first when b lacks one (CRLF when b uses CRLF). The
// value is written as yaml.Marshal writes the string. It fails when more than
// one line starts with `key:`, when value spans lines, or when the result does
// not read back as a mapping where key equals value.
func SetKey(b []byte, key, value string) ([]byte, error) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, fmt.Errorf("%s value spans lines", key)
	}
	enc, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	enc = bytes.TrimRight(enc, "\n")
	prefix := key + ":"
	at := -1
	for off := 0; off < len(b); {
		end := len(b)
		if i := bytes.IndexByte(b[off:], '\n'); i >= 0 {
			end = off + i + 1
		}
		if bytes.HasPrefix(b[off:end], []byte(prefix)) {
			if at >= 0 {
				return nil, fmt.Errorf("more than one %s line", prefix)
			}
			at = off
		}
		off = end
	}
	var out []byte
	if at < 0 {
		eol := "\n"
		if bytes.Contains(b, []byte("\r\n")) {
			eol = "\r\n"
		}
		out = append(out, b...)
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, eol...)
		}
		out = append(out, prefix...)
		out = append(out, ' ')
		out = append(out, enc...)
		out = append(out, eol...)
	} else {
		start := at + len(prefix)
		end := start
		for end < len(b) && b[end] != '\n' && b[end] != '\r' {
			end++
		}
		out = append(out, b[:start]...)
		out = append(out, ' ')
		out = append(out, enc...)
		out = append(out, b[end:]...)
	}
	var m map[string]any
	if err := yaml.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("result is not valid YAML: %w", err)
	}
	if got, ok := m[key].(string); !ok || got != value {
		return nil, fmt.Errorf("%s did not read back as %q", prefix, value)
	}
	return out, nil
}
