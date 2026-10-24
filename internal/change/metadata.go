package change

import (
	"bytes"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// Metadata is the content of a change's .openspec.yaml that Docket reads.
// A key that is absent leaves its field empty.
type Metadata struct {
	Schema  string `yaml:"schema"`
	Created string `yaml:"created"`
	Kind    string `yaml:"x-kind"`
	Source  string `yaml:"x-source"`
	Base    string `yaml:"x-base"`
}

// ParseMetadata reads .openspec.yaml. It fails when b is not YAML or not a
// mapping.
func ParseMetadata(b []byte) (Metadata, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Metadata{}, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return Metadata{}, fmt.Errorf("not a YAML mapping")
	}
	var m Metadata
	if err := doc.Content[0].Decode(&m); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

// SetBase replaces the value of the one top-level `x-base:` line and keeps
// every other byte, including the line ending. It fails when there is not
// exactly one such line, or when the result does not read back as base.
func SetBase(b []byte, base string) ([]byte, error) {
	const key = "x-base:"
	at := -1
	for off := 0; off < len(b); {
		end := len(b)
		if i := bytes.IndexByte(b[off:], '\n'); i >= 0 {
			end = off + i + 1
		}
		if bytes.HasPrefix(b[off:end], []byte(key)) {
			if at >= 0 {
				return nil, fmt.Errorf("more than one %s line", key)
			}
			at = off
		}
		off = end
	}
	if at < 0 {
		return nil, fmt.Errorf("no %s line", key)
	}
	start := at + len(key)
	end := start
	for end < len(b) && b[end] != '\n' && b[end] != '\r' {
		end++
	}
	out := make([]byte, 0, len(b)+len(base))
	out = append(out, b[:start]...)
	out = append(out, ' ')
	out = append(out, base...)
	out = append(out, b[end:]...)
	m, err := ParseMetadata(out)
	if err != nil {
		return nil, fmt.Errorf("result is not valid YAML: %w", err)
	}
	if m.Base != base {
		return nil, fmt.Errorf("%s did not read back as %q", key, base)
	}
	return out, nil
}
