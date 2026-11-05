// Package change holds what Docket knows about a change: its metadata and slug rules.
package change

// Kinds lists the allowed change kinds.
var Kinds = []string{"feature", "change-request", "baseline"}

// ValidLine is the form check for a title or source.
func ValidLine(s string) bool { return false }

// CheckSlug refuses the slugs Docket reserves.
func CheckSlug(slug string) error { return nil }

// Suffixed returns the slug for n <= 1, else slug-n.
func Suffixed(slug string, n int) string { return "" }

// FrontMatter renders the front matter of proposal.md.
func FrontMatter(title, kind, source string) []byte { return nil }
