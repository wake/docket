// Package change creates and describes OpenSpec changes.
package change

import (
	"fmt"
	"regexp"
)

// Kinds are the values --kind accepts.
var Kinds = []string{"feature", "change-request", "baseline"}

// NewResult is what `dkt new` reports on success.
type NewResult struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Base   string `json:"base"`
	// Requested is the slug asked for; Slug differs when a suffix was taken.
	Requested string `json:"requested"`
	// Used lists where the requested slug was found; never null.
	Used []string `json:"used"`
}

// MaxSlug is the longest slug, suffix included.
const MaxSlug = 200

var (
	slugForm   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
)

// ValidateSlug checks the slug rules.
func ValidateSlug(slug string) error {
	switch {
	case !slugForm.MatchString(slug):
		return fmt.Errorf("slug %q must be lowercase kebab-case: letters a-z and digits in groups joined by single hyphens", slug)
	case len(slug) > MaxSlug:
		return fmt.Errorf("slug is %d characters long; the limit is %d", len(slug), MaxSlug)
	case slug == "archive":
		return fmt.Errorf("slug %q is reserved for OpenSpec's archive folder", slug)
	case StartsWithDate(slug):
		return fmt.Errorf("slug %q must not start with a date: archiving adds the date", slug)
	}
	return nil
}

// StartsWithDate reports whether s starts with a YYYY-MM-DD date.
func StartsWithDate(s string) bool { return datePrefix.MatchString(s) }

// Suffixed is slug for n <= 1, else slug with "-n" appended.
func Suffixed(slug string, n int) string {
	if n <= 1 {
		return slug
	}
	return fmt.Sprintf("%s-%d", slug, n)
}
