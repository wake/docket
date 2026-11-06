package idea

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want Idea
	}{
		"title only":   {"---\ntitle: One\n---\nbody\n", Idea{Title: "One"}},
		"all keys":     {"---\ntitle: One\ndropped: true\nneeds: [docket/ideas/a.md, docket/ideas/b.md]\n---\n", Idea{Title: "One", Dropped: true, Needs: []string{"docket/ideas/a.md", "docket/ideas/b.md"}}},
		"dropped no":   {"---\ntitle: One\ndropped: false\n---\n", Idea{Title: "One"}},
		"empty needs":  {"---\ntitle: One\nneeds: []\n---\n", Idea{Title: "One"}},
		"crlf":         {"---\r\ntitle: One\r\nneeds: [docket/ideas/a.md]\r\n---\r\n", Idea{Title: "One", Needs: []string{"docket/ideas/a.md"}}},
		"quoted title": {"---\ntitle: \"A: b\"\n---\n", Idea{Title: "A: b"}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Parse([]byte(tc.in))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		in    string
		needs bool
	}{
		"no front matter": {"title: One\n", true},
		"no closing":      {"---\ntitle: One\n", true},
		"no title":        {"---\ndropped: true\n---\n", false},
		"blank title":     {"---\ntitle: \"  \"\n---\n", false},
		"dropped maybe":   {"---\ntitle: One\ndropped: maybe\n---\n", false},
		"no brackets":     {"---\ntitle: One\nneeds: docket/ideas/a.md\n---\n", true},
		"bad path":        {"---\ntitle: One\nneeds: [a.md]\n---\n", true},
		"nested path":     {"---\ntitle: One\nneeds: [docket/ideas/x/a.md]\n---\n", true},
		"lines under":     {"---\ntitle: One\nneeds:\n  - docket/ideas/a.md\n---\n", true},
		"needs an issue":  {"---\ntitle: One\nneeds: [docket/issues/x.md]\n---\n", true},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(tc.in))
			if err == nil {
				t.Fatal("want an error")
			}
			if tc.needs && !strings.Contains(err.Error(), NeedsForm) {
				t.Fatalf("error %q does not name %q", err, NeedsForm)
			}
		})
	}
}

func TestParseOrder(t *testing.T) {
	want := []Line{{3, "docket/ideas/a.md"}, {4, "docket/ideas/b.md"}}
	for _, in := range []string{
		"# Order\n\n- docket/ideas/a.md\n- docket/ideas/b.md\n",
		"# Order\r\n\r\n- docket/ideas/a.md\r\n- docket/ideas/b.md\r\n",
	} {
		lines, bad := ParseOrder([]byte(in))
		if !reflect.DeepEqual(lines, want) || len(bad) != 0 {
			t.Fatalf("got %v %v", lines, bad)
		}
	}
	for _, l := range []string{"1. docket/ideas/a.md", "- docket/ideas/a.md: note", "- docket/x.md", "- https://example.com/issues/7", "* docket/ideas/a.md"} {
		lines, bad := ParseOrder([]byte("# Order\n" + l + "\n"))
		if len(lines) != 0 || !reflect.DeepEqual(bad, []int{2}) {
			t.Fatalf("%q: got %v %v", l, lines, bad)
		}
	}
	if lines, bad := ParseOrder(nil); len(lines) != 0 || len(bad) != 0 {
		t.Fatalf("empty: got %v %v", lines, bad)
	}
}

func TestParseOrderIssue(t *testing.T) {
	lines, bad := ParseOrder([]byte("# Order\n- docket/issues/2026-11-01-fix-x.md\n"))
	if !reflect.DeepEqual(lines, []Line{{2, "docket/issues/2026-11-01-fix-x.md"}}) || len(bad) != 0 {
		t.Fatalf("got %v %v", lines, bad)
	}
}

func TestParseItem(t *testing.T) {
	const is = "docket/issues/x.md"
	got, err := ParseItem(is, []byte("---\ntitle: Fix X\nsource: s\ndropped: true\nneeds: [docket/ideas/a.md]\n---\n"))
	if err != nil || !reflect.DeepEqual(got, Idea{Title: "Fix X"}) {
		t.Fatalf("got %+v %v", got, err)
	}
	for _, in := range []string{"title: One\n", "---\ndropped: true\n---\n"} {
		if _, err := ParseItem(is, []byte(in)); err == nil {
			t.Fatalf("%q: want an error", in)
		}
	}
	got, err = ParseItem("docket/ideas/a.md", []byte("---\ntitle: A\ndropped: true\n---\n"))
	if err != nil || !got.Dropped {
		t.Fatalf("idea: got %+v %v", got, err)
	}
	if _, err := ParseItem("docket/ideas/a.md", []byte("---\ntitle: A\ndropped: maybe\n---\n")); err == nil {
		t.Fatal("idea: want an error")
	}
}

func TestNext(t *testing.T) {
	const a, b, c = "docket/ideas/a.md", "docket/ideas/b.md", "docket/ideas/c.md"
	order := func(p ...string) []Line {
		var l []Line
		for i, x := range p {
			l = append(l, Line{i + 1, x})
		}
		return l
	}
	needA := Idea{Needs: []string{a}}
	for name, tc := range map[string]struct {
		lines   []Line
		ideas   map[string]Idea
		work    map[string]Work
		next    string
		waiting []string
	}{
		"first":          {order(a, b), map[string]Idea{a: {}, b: {}}, nil, a, nil},
		"first dropped":  {order(a, b), map[string]Idea{a: {Dropped: true}, b: {}}, nil, b, nil},
		"first taken":    {order(a, b), map[string]Idea{a: {}, b: {}}, map[string]Work{a: {Any: true}}, b, nil},
		"need merged":    {order(b), map[string]Idea{a: {}, b: needA}, map[string]Work{a: {Any: true, OnTip: true}}, b, nil},
		"need in work":   {order(b), map[string]Idea{a: {}, b: needA}, map[string]Work{a: {Any: true}}, "", []string{a}},
		"need later":     {order(b, a), map[string]Idea{a: {}, b: needA}, nil, a, nil},
		"need dropped":   {order(b), map[string]Idea{a: {Dropped: true}, b: needA}, nil, b, nil},
		"nothing left":   {order(a, b), map[string]Idea{a: {Dropped: true}, b: {}}, map[string]Work{b: {Any: true}}, "", nil},
		"second can":     {order(b, c), map[string]Idea{a: {}, b: needA, c: {}}, map[string]Work{a: {Any: true}}, c, nil},
		"first waiter":   {order(b, c), map[string]Idea{a: {}, b: needA, c: {Needs: []string{b, a}}}, map[string]Work{a: {Any: true}}, "", []string{a}},
		"needs in order": {order(c), map[string]Idea{a: {}, b: {}, c: {Needs: []string{b, a}}}, map[string]Work{a: {Any: true}}, "", []string{b, a}},
	} {
		t.Run(name, func(t *testing.T) {
			next, waiting := Next(tc.lines, tc.ideas, tc.work)
			if next != tc.next || !reflect.DeepEqual(waiting, tc.waiting) {
				t.Fatalf("got %q %v, want %q %v", next, waiting, tc.next, tc.waiting)
			}
		})
	}
}
