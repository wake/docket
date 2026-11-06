package record

import (
	"strings"
	"testing"
	"time"
)

func TestCheckName(t *testing.T) {
	ok := []string{"fix-x", "a", "x1-y2", strings.Repeat("a", 200)}
	bad := map[string]string{
		"2026-10-30-fix-x":       "adds the date",
		"Fix_X":                  "",
		"-x":                     "",
		"x-":                     "",
		"x--y":                   "",
		"":                       "",
		strings.Repeat("a", 201): "200",
	}
	for _, n := range ok {
		if err := CheckName(n); err != nil {
			t.Errorf("%q: %v", n, err)
		}
	}
	for n, want := range bad {
		if err := CheckName(n); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", n, err, want)
		}
	}
}

func TestSlugUsesTheLocation(t *testing.T) {
	now := time.Date(2026, 10, 30, 23, 30, 0, 0, time.FixedZone("x", 9*3600))
	if got := Slug(now, "fix-x"); got != "2026-10-30-fix-x" {
		t.Errorf("%q", got)
	}
	if got := Path("2026-10-30-fix-x"); got != "docket/records/2026-10-30-fix-x.md" {
		t.Errorf("%q", got)
	}
}
