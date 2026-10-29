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
