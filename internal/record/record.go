// Package record holds what Docket knows about a record: where it lives, its
// kinds and its name rules.
package record

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/wake/docket/internal/change"
)

// Dir is the folder holding the records, relative to the repository root.
const Dir = "docket/records"

// BodyLimit is the most the four sections of a record may hold, in bytes (see change.Size).
const BodyLimit = 1440

// Kinds lists the allowed record kinds.
var Kinds = []string{"fix", "refactor", "test", "prompt", "checks"}

var kebab = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// CheckName refuses a name that is not lowercase kebab case of at most 200
// characters, or that starts with a date.
func CheckName(name string) error {
	switch {
	case change.HasDate(name):
		return errors.New("name must not start with a date (YYYY-MM-DD): dkt record adds the date")
	case len(name) > 200:
		return fmt.Errorf("name is %d characters; the limit is 200", len(name))
	case !kebab.MatchString(name):
		return errors.New("name must be lowercase letters and digits in runs separated by single hyphens")
	}
	return nil
}

// Slug is the record's slug: the date in now's location, a hyphen and name.
func Slug(now time.Time, name string) string { return now.Format("2006-01-02") + "-" + name }

// Path is the record file of slug, relative to the repository root.
func Path(slug string) string { return Dir + "/" + slug + ".md" }
