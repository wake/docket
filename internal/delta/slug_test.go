package delta

import "testing"

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Join when full":   "join-when-full",
		"Join (when full)": "join-when-full",
		"需求 R1#3":          "需求-r1-3",
		"  --A--  ":        "a",
	}
	for in, want := range cases {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
