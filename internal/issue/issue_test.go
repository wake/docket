package issue

import (
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	f, err := ParseFrontMatter([]byte("title: T\nsource: s\ndropped: true\nidea: some-idea\n"))
	if err != nil || f.Title != "T" || f.Dropped == nil || !*f.Dropped || f.Idea != "some-idea" {
		t.Errorf("%+v, %v", f, err)
	}
	// yaml.v3 reads the YAML 1.1 boolean yes as true.
	f, err = ParseFrontMatter([]byte("title: T\ndropped: yes\n"))
	if err != nil || f.Dropped == nil || !*f.Dropped {
		t.Errorf("yes: %+v, %v", f, err)
	}
	for _, in := range []string{"dropped: maybe\n", "- a\n- b\n", "idea: [a]\n"} {
		if _, err := ParseFrontMatter([]byte(in)); err == nil {
			t.Errorf("%q: no error", in)
		}
	}
	if f, err := ParseFrontMatter([]byte("title: \"a\\nb\"\n")); err != nil || f.Title != "" {
		t.Errorf("invalid title: %+v, %v", f, err)
	}
}

func TestRender(t *testing.T) {
	b, err := Render("T", "review", "## Current\n")
	want := "---\ntitle: T\nsource: review\n---\n\n## Current\n"
	if err != nil || string(b) != want {
		t.Errorf("%q, %v", b, err)
	}
	f := Parse(b)
	if f.FrontMatterErr != nil || f.FrontMatter.Source != "review" || len(f.Sections) != 1 || !strings.HasPrefix(f.Sections[0].Name, "Current") {
		t.Errorf("%+v", f)
	}
}
