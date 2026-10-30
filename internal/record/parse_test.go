package record

import (
	"reflect"
	"strings"
	"testing"
)

func TestWords(t *testing.T) {
	for _, c := range []struct {
		in   string
		want float64
	}{
		{"", 0},
		{"one two", 2},
		{"don't", 2},
		{"v0.9.0 dkt-record", 5},
		{"中文字", 1.5},
		{"中文abc", 2},
		{"かなカナ", 2},
		{"한국", 1},
		{"한국어", 1.5},
		{"nai\u0308ve", 1},
		{"2026", 1},
		{"- [ ] **bold** text", 2},
	} {
		if got := Words(c.in); got != c.want {
			t.Errorf("Words(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

const parseSrc = "---\ntitle: T\nkind: fix\nsource: s\n---\n\n## What\n\n<!-- ## Notes -->\nText here.\n\n## Tasks\n\n### Task 1: Move\n"

func TestParse(t *testing.T) {
	f := Parse([]byte(parseSrc))
	if f.FrontMatterErr != nil || f.FrontMatter.Title != "T" || f.FrontMatter.Kind != "fix" || f.FrontMatter.Source != "s" {
		t.Fatalf("front matter = %+v, %v", f.FrontMatter, f.FrontMatterErr)
	}
	if len(f.Sections) != 2 || f.Sections[0].Name != "What" || f.Sections[0].Line != 7 ||
		f.Sections[1].Name != "Tasks" || f.Sections[1].Line != 12 {
		t.Fatalf("sections = %+v", f.Sections)
	}
	if got := strings.Fields(f.Sections[0].Text); !reflect.DeepEqual(got, []string{"Text", "here."}) {
		t.Fatalf("What text = %q", f.Sections[0].Text)
	}
	if len(f.Plan.Tasks) != 1 || f.Plan.Tasks[0].Line != 14 {
		t.Fatalf("tasks = %+v", f.Plan.Tasks)
	}
}

func TestParseNoFrontMatter(t *testing.T) {
	f := Parse([]byte("## What\n\nx\n"))
	if f.FrontMatterErr == nil {
		t.Fatal("no front matter error")
	}
	if len(f.Sections) != 1 || f.Sections[0].Line != 1 {
		t.Fatalf("sections = %+v", f.Sections)
	}
}
