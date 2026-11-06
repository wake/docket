// Package change holds what Docket knows about a change: its metadata and slug rules.
package change

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Kinds lists the allowed change kinds.
var Kinds = []string{"feature", "change-request", "baseline"}

var datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// ValidLine is the form check for a title or source: the value is never trimmed.
func ValidLine(s string) bool {
	return utf8.ValidString(s) && !strings.ContainsAny(s, "\n\r") && strings.TrimSpace(s) != ""
}

// TitleLimit is the longest a change title may be, in bytes (see Size).
const TitleLimit = 120

// Size is the length of text with runs of white space counted as one space.
// shortcut: Cyrillic, Arabic, Indic and Thai text reach the limit up to three times sooner than English; weigh Chinese, Japanese and Korean characters as 3 and others as 1 if that is reported.
func Size(text string) int {
	return len(strings.Join(strings.Fields(text), " "))
}

// TitleTooLong says whether title is over TitleLimit, with the message to report.
func TitleTooLong(title string) (msg string, tooLong bool) {
	n := Size(title)
	if n <= TitleLimit {
		return "", false
	}
	return fmt.Sprintf("title is %d bytes; the limit is %d", n, TitleLimit), true
}

// HasDate reports whether s starts with a date (YYYY-MM-DD).
func HasDate(s string) bool { return datePrefix.MatchString(s) }

// CheckSlug refuses the slugs Docket reserves; OpenSpec checks the rest.
func CheckSlug(slug string) error {
	if HasDate(slug) {
		return errors.New("slug must not start with a date (YYYY-MM-DD): archiving adds the date")
	}
	if slug == "archive" {
		return errors.New("slug \"archive\" is the name of OpenSpec's archive folder")
	}
	return nil
}

// Suffixed returns the slug for n <= 1, else slug-n.
func Suffixed(slug string, n int) string {
	if n <= 1 {
		return slug
	}
	return slug + "-" + strconv.Itoa(n)
}

// FrontMatter renders the front matter of proposal.md; callers validate first.
func FrontMatter(title, kind, source string) []byte {
	return []byte("---\ntitle: " + strconv.Quote(title) + "\nkind: " + kind + "\nsource: " + strconv.Quote(source) + "\n---\n")
}

// TaskRefs reads one Change trailer value for slug. mine is false when the
// value is for another change; then tasks and ok mean nothing. A value that is
// slug alone names no task. ok is false when the task list after "/" is not
// one or more positive integers separated by commas.
func TaskRefs(value, slug string) (tasks []int, ok, mine bool) {
	if value == slug {
		return nil, true, true
	}
	list, found := strings.CutPrefix(value, slug+"/")
	if !found {
		return nil, false, false
	}
	for _, p := range strings.Split(list, ",") {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil || !taskNumber.MatchString(p) {
			return nil, false, true
		}
		tasks = append(tasks, n)
	}
	return tasks, true, true
}

var taskNumber = regexp.MustCompile(`^[1-9][0-9]*$`)

// SchemaOf returns the schema named by a change's .openspec.yaml, or "docket"
// when it names none.
func SchemaOf(b []byte) string {
	for _, line := range strings.Split(string(b), "\n") {
		v, found := strings.CutPrefix(strings.TrimSuffix(line, "\r"), "schema:")
		if !found {
			continue
		}
		v = strings.TrimSpace(v)
		if i := strings.Index(v, " #"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		if n := len(v); n >= 2 && (v[0] == '"' || v[0] == '\'') && v[n-1] == v[0] {
			v = v[1 : n-1]
		}
		if v == "" {
			return "docket"
		}
		return v
	}
	return "docket"
}
