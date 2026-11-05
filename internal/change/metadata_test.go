package change

import (
	"strconv"
	"strings"
	"testing"
)

func TestFrontMatter(t *testing.T) {
	got := string(FrontMatter("Add x", "feature", "issue 7"))
	want := "---\ntitle: \"Add x\"\nkind: feature\nsource: \"issue 7\"\n---\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidLine(t *testing.T) {
	cases := map[string]bool{
		"ok": true, " padded ": true, "中文": true,
		"": false, "   ": false, "\t": false, "a\nb": false, "a\rb": false, "bad\xff": false,
	}
	for s, want := range cases {
		if ValidLine(s) != want {
			t.Errorf("ValidLine(%q) = %v, want %v", s, !want, want)
		}
	}
}

func TestFrontMatterRoundTrip(t *testing.T) {
	in := "a \"q\" \\ : # \t é"
	fm := string(FrontMatter(in, "feature", in))
	for _, p := range []string{"title: ", "source: "} {
		line := strings.SplitN(strings.SplitN(fm, p, 2)[1], "\n", 2)[0]
		out, err := strconv.Unquote(line)
		if err != nil || out != in {
			t.Errorf("%s round trip: %q, %v", p, out, err)
		}
	}
}

func TestCheckSlug(t *testing.T) {
	for _, s := range []string{"add-2026-10-24", "archived-x", "add-x"} {
		if err := CheckSlug(s); err != nil {
			t.Errorf("CheckSlug(%q) = %v", s, err)
		}
	}
	if err := CheckSlug("2026-10-24-add-x"); err == nil || !strings.Contains(err.Error(), "date") {
		t.Errorf("date slug: %v", err)
	}
	if err := CheckSlug("archive"); err == nil || !strings.Contains(err.Error(), "archive folder") {
		t.Errorf("archive slug: %v", err)
	}
}

func TestSuffixed(t *testing.T) {
	for n, want := range map[int]string{0: "s", 1: "s", 2: "s-2", 10: "s-10"} {
		if got := Suffixed("s", n); got != want {
			t.Errorf("Suffixed(%d) = %q, want %q", n, got, want)
		}
	}
}
