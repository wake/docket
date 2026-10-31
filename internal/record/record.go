// Package record holds the rules of records: one file per piece of light-review work under docket/records.
package record

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/wake/docket/internal/change"
	"github.com/wake/docket/internal/gitx"
)

// CommitsFor returns the commits whose Record trailers hold slug, in order.
func CommitsFor(slug string, commits []gitx.Commit) []gitx.Commit {
	var out []gitx.Commit
	for _, c := range commits {
		if slices.Contains(c.Record, slug) {
			out = append(out, c)
		}
	}
	return out
}

// Kinds are the values --kind accepts for a record.
var Kinds = []string{"fix", "refactor", "test", "prompt", "checks"}

// Sections are the level-two headings of a record, in order.
var Sections = []string{"What", "Why", "Verification", "Review"}

// datePrefixLen is the length of "2006-01-02-", the date and its hyphen.
const datePrefixLen = len("2006-01-02-")

// ValidateName checks the name given to dkt record, before the date is added.
func ValidateName(name string) error {
	if change.StartsWithDate(name) {
		return fmt.Errorf("name %q must not start with a date: dkt record adds the date", name)
	}
	if err := change.ValidateSlug(name); err != nil {
		return err
	}
	if full := len(name) + datePrefixLen; full > change.MaxSlug {
		return fmt.Errorf("slug is %d characters long; the limit is %d", full, change.MaxSlug)
	}
	return nil
}

// Slug is the slug of a record named name, written at now: the date in now's
// own location, a hyphen and the name.
func Slug(now time.Time, name string) string {
	return now.Format("2006-01-02") + "-" + name
}

// Render is the content of a new record: the front matter, a blank line and
// the template.
func Render(title, kind, source, template string) ([]byte, error) {
	front, err := change.Docket{Title: title, Kind: kind, Source: source}.Marshal()
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("---\n")
	b.Write(front)
	b.WriteString("---\n\n")
	b.WriteString(template)
	return b.Bytes(), nil
}

// Result is what `dkt record` reports on success.
type Result struct {
	Slug   string `json:"slug"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	// Requested is the slug asked for; Slug differs when a suffix was taken.
	Requested string `json:"requested"`
	// Used lists where the requested slug was found; never null.
	Used []string `json:"used"`
}

// ErrTooLong means no free slug is short enough: the suffix pushed it past
// change.MaxSlug.
var ErrTooLong = errors.New("slug too long")

// LinkError means a record file name is taken by a symbolic link, which
// Create does not write through.
type LinkError struct{ Name string }

func (e *LinkError) Error() string {
	return e.Name + " is a symbolic link; Docket does not write through links"
}

// Create writes content to the first free record file in dir, creating dir
// when it first needs to open a file (so nothing is created when it returns
// ErrTooLong without trying one): slug, then
// slug-2, slug-3, and so on. A candidate is skipped when used reports it or
// when its file exists; the file is created exclusively, so concurrent runs
// never take the same one. It returns the slug written. A symbolic link in
// the way stops it with a *LinkError; a candidate past change.MaxSlug stops it
// with an error wrapping ErrTooLong. When the write fails the file is removed.
func Create(dir, slug string, used func(slug string) bool, content []byte) (string, error) {
	for n := 1; ; n++ {
		cand := change.Suffixed(slug, n)
		if len(cand) > change.MaxSlug {
			return "", fmt.Errorf("%w: %q would be %d characters long; the limit is %d", ErrTooLong, cand, len(cand), change.MaxSlug)
		}
		if used(cand) {
			continue
		}
		path := filepath.Join(dir, cand+".md")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o644)
		if errors.Is(err, fs.ErrExist) {
			fi, lerr := os.Lstat(path)
			if lerr == nil && fi.Mode()&fs.ModeSymlink != 0 {
				return "", &LinkError{Name: cand + ".md"}
			}
			continue
		}
		if err != nil {
			return "", err
		}
		_, werr := f.Write(content)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return "", errors.Join(werr, os.Remove(path))
		}
		return cand, nil
	}
}

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
