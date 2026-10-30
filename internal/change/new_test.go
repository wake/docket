package change

import (
	"strings"
	"testing"
)

func TestValidateSlug(t *testing.T) {
	cases := []struct {
		slug string
		ok   bool
		want string
	}{
		{"add-x", true, ""},
		{"Add-X", false, ""},
		{"add_x", false, ""},
		{"-x", false, ""},
		{"x-", false, ""},
		{"a--b", false, ""},
		{strings.Repeat("a", 200), true, ""},
		{strings.Repeat("a", 201), false, "200"},
		{"2026-10-24-add-x", false, "archiv"},
		{"archive", false, "reserved"},
	}
	for _, c := range cases {
		err := ValidateSlug(c.slug)
		if (err == nil) != c.ok {
			t.Errorf("%q: err = %v, want ok=%v", c.slug, err, c.ok)
		}
		if err != nil && c.want != "" && !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %q lacks %q", c.slug, err, c.want)
		}
	}
}

func TestSuffixed(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{{0, "add-x"}, {1, "add-x"}, {2, "add-x-2"}, {12, "add-x-12"}} {
		if got := Suffixed("add-x", c.n); got != c.want {
			t.Errorf("Suffixed(add-x, %d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestStartsWithDate(t *testing.T) {
	for s, want := range map[string]bool{
		"2026-10-30-fix-x": true, "2026-10-30": true,
		"add-x": false, "2026-10-x": false, "20261030-x": false,
	} {
		if got := StartsWithDate(s); got != want {
			t.Errorf("StartsWithDate(%q) = %v, want %v", s, got, want)
		}
	}
}
