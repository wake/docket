// Package record holds the rules of records: one file per piece of light-review work under docket/records.
package record

import (
	"strings"

	"github.com/wake/docket/internal/change"
)

// Dir is the folder that holds the records, relative to the repository root.
const Dir = "docket/records"

// SlugOf returns the slug of a record file name (no folder): the name without
// ".md" when it is a date, a hyphen and at least one more character, then ".md".
func SlugOf(name string) (string, bool) {
	slug, ok := strings.CutSuffix(name, ".md")
	const date = len("2026-01-02")
	if !ok || !change.StartsWithDate(slug) || len(slug) <= date+1 || slug[date] != '-' {
		return "", false
	}
	return slug, true
}
