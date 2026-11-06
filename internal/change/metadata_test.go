package change

import (
	"reflect"
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

func TestFrontMatterWithoutKind(t *testing.T) {
	got := string(FrontMatter("Lint times out", "", "review of #40"))
	want := "---\ntitle: \"Lint times out\"\nsource: \"review of #40\"\n---\n"
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

func TestTaskRefs(t *testing.T) {
	cases := []struct {
		value    string
		tasks    []int
		ok, mine bool
	}{
		{"add-x", nil, true, true},
		{"add-x/1", []int{1}, true, true},
		{"add-x/1,2", []int{1, 2}, true, true},
		{"add-x/2, 3", []int{2, 3}, true, true},
		{"add-x/2,2", []int{2, 2}, true, true},
		{"add-x-2/1", nil, false, false},
		{"add-x/", nil, false, true},
		{"add-x/0", nil, false, true},
		{"add-x/two", nil, false, true},
		{"add-x/1,,2", nil, false, true},
	}
	for _, c := range cases {
		tasks, ok, mine := TaskRefs(c.value, "add-x")
		if mine != c.mine {
			t.Errorf("%q: mine %v", c.value, mine)
			continue
		}
		if mine && c.value != "add-x" && (ok != c.ok || (ok && !reflect.DeepEqual(tasks, c.tasks))) {
			t.Errorf("%q: got %v %v", c.value, tasks, ok)
		}
		if c.value == "add-x" && len(tasks) != 0 {
			t.Errorf("slug alone names tasks %v", tasks)
		}
	}
}

func TestSchemaOf(t *testing.T) {
	cases := map[string]string{
		"schema: spec-driven\ncreated: 2026-01-01\n": "spec-driven",
		"created: 2026-01-01\n":                      "docket",
		"":                                           "docket",
		"schema: \"my-schema\"\n":                    "my-schema",
		"schema: 'my-schema'\n":                      "my-schema",
		"schema: my-schema # note\n":                 "my-schema",
		"  schema: nested\n":                         "docket",
		"schema:\n":                                  "docket",
		"schema: \"\"\n":                             "docket",
		"a: 1\r\nschema: crlf\r\n":                   "crlf",
	}
	for in, want := range cases {
		if got := SchemaOf([]byte(in)); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestSize(t *testing.T) {
	for s, want := range map[string]int{"  a   b \n": 3, strings.Repeat("字", 40): 120, "": 0} {
		if got := Size(s); got != want {
			t.Errorf("Size(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestTitleTooLong(t *testing.T) {
	for name, tc := range map[string]struct {
		title string
		want  bool
	}{
		"120 bytes":        {strings.Repeat("a", 120), false},
		"121 bytes":        {strings.Repeat("a", 121), true},
		"40 Chinese":       {strings.Repeat("字", 40), false},
		"spaces collapsed": {strings.Repeat("a", 118) + "   b", false},
	} {
		msg, got := TitleTooLong(tc.title)
		if got != tc.want {
			t.Errorf("%s: got %v", name, got)
		}
		if got && (!strings.Contains(msg, "121") || !strings.Contains(msg, "120")) {
			t.Errorf("%s: message %q", name, msg)
		}
	}
}

func TestHasDate(t *testing.T) {
	for s, want := range map[string]bool{"2026-10-30-x": true, "2026-10-30": true, "add-x": false} {
		if HasDate(s) != want {
			t.Errorf("HasDate(%q) = %v", s, !want)
		}
	}
}
