package lint

import "testing"

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"Join twice — rejected": "join-twice-rejected",
		"Ünïcode 名前 2":          "ünïcode-名前-2",
		"--x--":                 "x",
	} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}
