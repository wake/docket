// Package change creates and describes OpenSpec changes.
package change

import (
	"bytes"
	"fmt"
	"os"
	"regexp"

	"go.yaml.in/yaml/v3"
)

// Kinds are the values --kind accepts.
var Kinds = []string{"feature", "change-request", "fix", "baseline"}

// NewResult is what `dkt new` reports on success.
type NewResult struct {
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Base   string `json:"base"`
}

const maxSlug = 200

var (
	slugForm   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
)

// ValidateSlug checks the slug rules.
func ValidateSlug(slug string) error {
	switch {
	case !slugForm.MatchString(slug):
		return fmt.Errorf("slug %q must be lowercase kebab-case: letters a-z and digits in groups joined by single hyphens", slug)
	case len(slug) > maxSlug:
		return fmt.Errorf("slug is %d characters long; the limit is %d", len(slug), maxSlug)
	case datePrefix.MatchString(slug):
		return fmt.Errorf("slug %q must not start with a date: archiving adds the date", slug)
	}
	return nil
}

// AppendMetadata appends x-kind, x-source and x-base to the .openspec.yaml at
// path, then parses the whole file and checks the three values read back.
func AppendMetadata(path, kind, source, base string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.Write(b)
	if len(b) > 0 && b[len(b)-1] != '\n' {
		buf.WriteByte('\n')
	}
	want := []struct{ key, val string }{{"x-kind", kind}, {"x-source", source}, {"x-base", base}}
	for _, kv := range want {
		enc, err := yaml.Marshal(map[string]string{kv.key: kv.val})
		if err != nil {
			return err
		}
		buf.Write(enc)
	}
	var got map[string]any
	if err := yaml.Unmarshal(buf.Bytes(), &got); err != nil {
		return fmt.Errorf("%s is not valid YAML: %w", path, err)
	}
	for _, kv := range want {
		if s, ok := got[kv.key].(string); !ok || s != kv.val {
			return fmt.Errorf("%s: %s did not read back as %q", path, kv.key, kv.val)
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
