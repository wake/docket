package idea

import (
	"reflect"
	"testing"
)

func TestParseOrder(t *testing.T) {
	const head = "# Order\n\n"
	a, b := "docket/ideas/a.md", "docket/ideas/b.md"
	cases := []struct {
		name  string
		in    string
		lines []Line
		probs []int // problem lines
	}{
		{"two lines", head + "1. `" + a + "`\n2. `" + b + "`\n", []Line{{1, a, 3}, {2, b, 4}}, nil},
		{"crlf and trailing spaces", "# Order  \r\n\r\n1. `" + a + "`  \r\n2. `" + b + "` \r\n", []Line{{1, a, 3}, {2, b, 4}}, nil},
		{"checkbox", head + "1. `" + a + "`\n2. `" + b + "`\n3. [ ] `docket/ideas/c.md`\n", []Line{{1, a, 3}, {2, b, 4}}, []int{5}},
		{"gap", head + "1. `" + a + "`\n2. `" + b + "`\n4. `docket/ideas/c.md`\n5. `docket/ideas/d.md`\n",
			[]Line{{1, a, 3}, {2, b, 4}, {4, "docket/ideas/c.md", 5}, {5, "docket/ideas/d.md", 6}}, []int{5}},
		{"first not 1", head + "2. `" + a + "`\n", []Line{{2, a, 3}}, []int{3}},
		{"no heading", "\n1. `" + a + "`\n", []Line{{1, a, 2}}, []int{2}},
		{"stray line", head + "1. `" + a + "`\n- later\n2. `" + b + "`\n", []Line{{1, a, 3}, {2, b, 5}}, []int{4}},
	}
	for _, c := range cases {
		lines, probs := ParseOrder([]byte(c.in))
		var at []int
		for _, p := range probs {
			at = append(at, p.At)
		}
		if !reflect.DeepEqual(lines, c.lines) || !reflect.DeepEqual(at, c.probs) {
			t.Errorf("%s: lines %+v problems %+v", c.name, lines, probs)
		}
	}
}

func TestParse(t *testing.T) {
	f, err := Parse([]byte("---\ntitle: X\nneeds: [docket/ideas/y.md]\n---\nbody\n"))
	if err != nil || f.Title != "X" || len(f.Needs) != 1 || f.Needs[0] != "docket/ideas/y.md" {
		t.Fatalf("%+v %v", f, err)
	}
	for name, in := range map[string]string{
		"needs a string": "---\ntitle: X\nneeds: docket/ideas/y.md\n---\n",
		"dropped maybe":  "---\ntitle: X\ndropped: maybe\n---\n",
		"no front":       "title: X\n",
	} {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	f, err = Parse([]byte("---\ntitle: \"a\\nb\"\n---\n"))
	if err != nil || f.Title != "" {
		t.Fatalf("%+v %v", f, err)
	}
}

func TestIsIdeaPath(t *testing.T) {
	for p, want := range map[string]bool{
		"docket/ideas/a.md":         true,
		"README.md":                 false,
		"docket/ideas/../README.md": false,
		"docket/ideas/drafts/a.md":  false,
		"docket/ideas/a.txt":        false,
		"./docket/ideas/a.md":       false,
	} {
		if got := IsIdeaPath(p); got != want {
			t.Errorf("%q = %v", p, got)
		}
	}
}
