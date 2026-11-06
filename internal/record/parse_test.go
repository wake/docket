package record

import (
	"reflect"
	"strings"
	"testing"
)

const front = "---\ntitle: T\nkind: fix\nsource: s\n---\n"

func names(p Parsed) []string {
	var out []string
	for _, s := range p.Sections {
		out = append(out, s.Name)
	}
	return out
}

func TestParse(t *testing.T) {
	p := Parse([]byte(front + "\n## What\n\nwhat <!-- gone -->\n\n## Why\n\n```\n## x\n```\n\n## Notes\nn\n"))
	if !p.HasFront || p.FrontErr != nil || p.Front.Fields["kind"].Value != "fix" {
		t.Fatalf("%+v", p)
	}
	if got := names(p); !reflect.DeepEqual(got, []string{"What", "Why", "Notes"}) {
		t.Fatalf("%v", got)
	}
	if s := p.Sections[0]; s.Line != 7 || strings.TrimSpace(s.Text) != "what" {
		t.Errorf("%+v", s)
	}
	if s := p.Sections[1]; s.Line != 11 || !strings.Contains(s.Text, "## x") {
		t.Errorf("%+v", s)
	}
	if len(p.Outside) != 0 {
		t.Errorf("outside %v", p.Outside)
	}
}

func TestParseFrontMatterCommentIsNoHeading(t *testing.T) {
	p := Parse([]byte("---\n# a comment\ntitle: T\n---\n## What\nw\n"))
	if got := names(p); !reflect.DeepEqual(got, []string{"What"}) || len(p.Outside) != 0 || p.Sections[0].Line != 5 {
		t.Fatalf("%v %v %+v", got, p.Outside, p.Sections)
	}
}

func TestParseOutside(t *testing.T) {
	p := Parse([]byte(front + "<!-- c -->\n\nlead\n## What\nw\n"))
	if !reflect.DeepEqual(p.Outside, []int{8}) {
		t.Errorf("paragraph: %v", p.Outside)
	}
	p = Parse([]byte(front + "# Title\n\ntext\n\n## What\nw\n"))
	if !reflect.DeepEqual(p.Outside, []int{6, 8}) {
		t.Errorf("title: %v", p.Outside)
	}
	p = Parse([]byte("lead\n"))
	if p.HasFront || !reflect.DeepEqual(p.Outside, []int{1}) {
		t.Errorf("no front: %+v", p)
	}
}

func TestParseTasks(t *testing.T) {
	p := Parse([]byte(front + "## What\n### Task 1: Move\n\n## Tasks\n\n### Task 1: Rename\n- [ ] a\n"))
	if len(p.Tasks.Tasks) != 1 || p.Tasks.Tasks[0].Name != "Rename" || p.Tasks.Tasks[0].Line != 11 {
		t.Fatalf("%+v", p.Tasks)
	}
	if len(Parse([]byte(front+"## What\nw\n")).Tasks.Tasks) != 0 {
		t.Error("tasks without ## Tasks")
	}
}

func TestParseCRLFAndBOM(t *testing.T) {
	src := front + "## What\nw\n\n## Why\ny\n"
	want := Parse([]byte(src))
	got := Parse([]byte("\xef\xbb\xbf" + strings.ReplaceAll(src, "\n", "\r\n")))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v\n%+v", got, want)
	}
}

func TestBodySize(t *testing.T) {
	small := Parse([]byte(front + "## What\nab <!-- long long long -->\n## Why\ncd\n## Verification\n\n## Review\nef\n## Tasks\n" + strings.Repeat("x", 500)))
	if n := small.BodySize(); n != 6 {
		t.Errorf("comment and tasks counted: %d", n)
	}
	cn := strings.Repeat("字", 120)
	zh := Parse([]byte(front + "## What\n" + cn + "\n## Why\n" + cn + "\n## Verification\n" + cn + "\n## Review\n" + cn + "\n"))
	if n := zh.BodySize(); n != 1440 {
		t.Errorf("chinese: %d", n)
	}
}
