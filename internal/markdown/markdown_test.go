package markdown

import (
	"reflect"
	"strings"
	"testing"
)

func TestLines(t *testing.T) {
	got := Lines([]byte("\xef\xbb\xbfa\r\nb\rc\n"))
	want := []string{"a", "b", "c", ""}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines = %q, want %q", got, want)
	}
}

func TestFenceMask(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		want  []bool
	}{
		{"backticks", []string{"a", "```go", "x", "```", "b"}, []bool{false, true, true, true, false}},
		{"tildes", []string{"~~~", "x", "~~~", "b"}, []bool{true, true, true, false}},
		{"indented", []string{"  ```", "x", "  ```", "b"}, []bool{true, true, true, false}},
		{"longer closes", []string{"```", "x", "`````", "b"}, []bool{true, true, true, false}},
		{"shorter does not", []string{"````", "x", "```", "y"}, []bool{true, true, true, true}},
		{"other char does not", []string{"```", "x", "~~~", "y", "```", "z"}, []bool{true, true, true, true, true, false}},
		{"closing with text does not", []string{"```", "``` x", "y"}, []bool{true, true, true}},
		{"nbsp indent", []string{"\u00a0```", "x", "\u00a0```", "b"}, []bool{true, true, true, false}},
		{"ideographic space indent", []string{"\u3000```", "x", "```", "b"}, []bool{true, true, true, false}},
		{"closing with trailing nbsp", []string{"```", "x", "```\u00a0", "b"}, []bool{true, true, true, false}},
		{"unclosed", []string{"a", "```", "b", "c"}, []bool{false, true, true, true}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FenceMask(tc.lines); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("FenceMask = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHeadings(t *testing.T) {
	lines := []string{"## Why", "#NoSpace", "### Task 1: x ###", "```", "# fenced", "```", "####### seven", "# "}
	got := Headings(lines, FenceMask(lines))
	want := []Heading{{2, "Why", 0}, {3, "Task 1: x", 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Headings = %+v, want %+v", got, want)
	}
}

func TestHeadingsJavaScriptSpace(t *testing.T) {
	lines := []string{"##\u00a0Why"}
	got := Headings(lines, FenceMask(lines))
	want := []Heading{{2, "Why", 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Headings = %+v, want %+v", got, want)
	}
}

func TestBody(t *testing.T) {
	lines := []string{"# A", "a1", "## B", "b1", "```", "# no", "```", "# C", "c1"}
	mask := FenceMask(lines)
	hs := Headings(lines, mask)
	if len(hs) != 3 {
		t.Fatalf("headings = %+v", hs)
	}
	cases := []struct {
		h        Heading
		from, to int
	}{
		{hs[0], 1, 7},
		{hs[1], 3, 7},
		{hs[2], 8, 9},
	}
	for _, c := range cases {
		from, to := Body(lines, mask, c.h)
		if from != c.from || to != c.to {
			t.Errorf("Body(%q) = %d,%d want %d,%d", c.h.Text, from, to, c.from, c.to)
		}
	}
}

func TestBlankComments(t *testing.T) {
	sp := func(n int) string { return strings.Repeat(" ", n) }
	tests := []struct {
		name string
		in   []string
		mask []bool
		want []string
	}{
		{"single line", []string{"a<!-- x -->b"}, []bool{false}, []string{"a" + sp(10) + "b"}},
		{"multi line", []string{"a<!--", "x", "-->b<!-- y --> c"}, []bool{false, false, false},
			[]string{"a" + sp(4), sp(1), sp(3) + "b" + sp(10) + " c"}},
		{"bang close", []string{"a<!-- x --!>b"}, []bool{false}, []string{"a" + sp(11) + "b"}},
		{"unclosed", []string{"a<!-- open", "still", "more"}, []bool{false, false, false},
			[]string{"a" + sp(9), sp(5), sp(4)}},
		{"fence does not open", []string{"```", "<!--", "```", "text"}, []bool{true, true, true, false},
			[]string{"```", "<!--", "```", "text"}},
		{"fence inside comment", []string{"<!-- a", "```", "-->b"}, []bool{false, true, false},
			[]string{sp(6), "```", sp(3) + "b"}},
		{"offsets kept", []string{"é<!--é-->é"}, []bool{false}, []string{"é" + sp(9) + "é"}},
	}
	for _, tc := range tests {
		got := BlankComments(tc.in, tc.mask)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: BlankComments = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestCodeSpans(t *testing.T) {
	tests := []struct {
		in   string
		want [][2]int
	}{
		{"a `b` c", [][2]int{{2, 5}}},
		{"``x`y``", [][2]int{{0, 7}}},
		{"a ` b", nil},
		{"`a` and `b`", [][2]int{{0, 3}, {8, 11}}},
		{"`` a ` b", nil},
	}
	for _, tc := range tests {
		got := CodeSpans(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("CodeSpans(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHeadingsTrimFEFF(t *testing.T) {
	lines := []string{"## Why\ufeff"}
	got := Headings(lines, FenceMask(lines))
	if want := []Heading{{2, "Why", 0}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Headings = %+v, want %+v", got, want)
	}
}

func TestCodeSpansShorterRunDoesNotClose(t *testing.T) {
	if got, want := CodeSpans("`a``b`"), [][2]int{{0, 6}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CodeSpans = %v, want %v", got, want)
	}
}

func TestFencesInComments(t *testing.T) {
	b3, b4 := "```", "````"
	tests := []struct {
		name  string
		lines []string
		want  []int
	}{
		{"marker left", []string{"<!--", b3, "-->", "### Task 2: Real"}, []int{1}},
		{"tildes", []string{"<!--", "~~~", "-->"}, []int{1}},
		{"indented marker", []string{"<!--", "  " + b3, "-->"}, []int{1}},
		{"whole block", []string{"<!--", b3, "x", b3, "-->"}, nil},
		{"comment marker in code", []string{b3, "<!--", b3, "### H"}, nil},
		{"opener holds comment marker", []string{b3 + " <!--", "x", b3, "y"}, nil},
		{"marker closes comment", []string{"<!--", b3 + " -->", "b"}, []int{1}},
		{"comment opened mid-line", []string{"text <!-- note", b3, "-->"}, []int{1}},
		{"comment closed before fence", []string{"<!-- a -->", b3, "x", b3}, nil},
		{"unclosed comment", []string{"<!--", b3, "x"}, nil},
		{"shorter run does not close", []string{"<!--", b4, b3, "-->"}, []int{1}},
		{"other character does not close", []string{"<!--", b3, "~~~", "-->"}, []int{1}},
		{"longer run closes", []string{"<!--", b3, b4, "-->"}, nil},
		{"closer with text does not close", []string{"<!--", b3, b3 + " x", "-->"}, []int{1}},
		// In document order line 3 opens code, so lines 4 to 6 are code, not a comment.
		{"code after a reported comment", []string{"<!--", b3, "-->", b3, "<!--", "~~~", "-->"}, []int{1}},
		{"bang close", []string{"<!--", b3, "--!>"}, []int{1}},
		{"two comments", []string{"<!--", b3, "-->", "<!--", "~~~", "-->"}, []int{1, 4}},
		{"reopened on one line", []string{"<!-- a --> <!--", b3, "-->"}, []int{1}},
		{"no comment", []string{"a", b3, "x", b3}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FencesInComments(tt.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FencesInComments = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFenceInfoBackticks(t *testing.T) {
	b3, t3 := "```", "~~~"
	tests := []struct {
		name  string
		lines []string
		want  []int
	}{
		{"inline code at line start", []string{b3 + "npm test" + b3, "### Task 2: Real"}, []int{0}},
		{"four backticks", []string{"````a`", "x"}, []int{0}},
		{"indented", []string{"  " + b3 + "a`", "x"}, []int{0}},
		{"plain info", []string{b3 + "go", "x", b3}, nil},
		{"tilde info with backtick", []string{t3 + " a`b", "x", t3}, nil},
		{"six backticks", []string{"``````", "x", "``````"}, nil},
		{"inside a tilde fence", []string{t3, b3 + "x" + b3, t3}, nil},
		{"backtick after U+2028", []string{b3 + "a\u2028`", "x", b3}, nil},
		{"backtick after U+2029", []string{b3 + "a\u2029`", "x", b3}, nil},
		{"both read inside code", []string{b3 + "a`", b3, b3 + "b`", b3}, []int{0}},
		{"readings swap", []string{b3 + "a`", b3, b3, b3 + "b`", b3}, []int{0, 3}},
		{"after a closed fence", []string{b3, "x", b3, b3 + "a`"}, []int{3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FenceInfoBackticks(tt.lines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FenceInfoBackticks = %v, want %v", got, tt.want)
			}
			// With no line reported, both rules read the file alike.
			if got == nil && !reflect.DeepEqual(FenceMask(tt.lines), fenceMask(tt.lines, openFence116)) {
				t.Errorf("masks differ with no finding")
			}
		})
	}
}

func TestCommentsInCodeSpans(t *testing.T) {
	op, b3 := "<!--", "```"
	tests := []struct {
		name  string
		lines []string
		want  []SpanComment
	}{
		{"runs to the end", []string{"`" + op + "` x", "### T"}, []SpanComment{{0, -1}}},
		{"closed on a later line", []string{"a `" + op + "` b", "c", "-->"}, []SpanComment{{0, 2}}},
		{"closed in a later span", []string{"`" + op + "` and `-->`"}, []SpanComment{{0, 0}}},
		{"span after a comment closes", []string{"x " + op + " a --> `" + op + "` b"}, []SpanComment{{0, -1}}},
		{"comment starting a line", []string{"   " + op + " a --> `" + op + "` b -->"}, nil},
		{"comment block over two lines", []string{op, "a --> `" + op + "` b -->"}, nil},
		{"span on the line after a block", []string{op + " a -->", "`" + op + "`"}, []SpanComment{{1, -1}}},
		{"div block", []string{"<div>", "`" + op + "` hidden -->", "</div>"}, nil},
		{"after a div block's blank line", []string{"<div>", "", "`" + op + "`"}, []SpanComment{{2, -1}}},
		{"details on one line", []string{"<details>`" + op + "` x -->"}, nil},
		{"pre block", []string{"<pre>", "`" + op + "` x -->", "</pre>", "`" + op + "`"}, []SpanComment{{3, -1}}},
		{"tag alone on a line", []string{"<span>", "`" + op + "` x -->"}, nil},
		{"tag alone in a paragraph", []string{"para", "<span>", "`" + op + "` x"}, []SpanComment{{2, -1}}},
		{"tag with text after it", []string{"<span> `" + op + "` x"}, []SpanComment{{0, -1}}},
		{"declaration ends on its line", []string{"<!DOCTYPE html>", "`" + op + "`"}, []SpanComment{{1, -1}}},
		{"declaration over two lines", []string{"<!DOCTYPE", "`" + op + "` --> >", "`" + op + "`"}, []SpanComment{{2, -1}}},
		{"div inside a comment", []string{op, "<div>", "-->", "`" + op + "`"}, []SpanComment{{3, -1}}},
		{"tag after a fence", []string{"para", b3, "x", b3, "<span>", "`" + op + "` x"}, nil},
		{"four spaces do not start a block", []string{"para", "    " + op + " a --> `" + op + "` b"}, []SpanComment{{1, -1}}},
		// The comment opened after the block's comment ends carries over, as in dkt's reading.
		{"comment opened in a block line", []string{op + " a --> " + op + " b", "` --> `" + op + "`"}, []SpanComment{{1, -1}}},
		{"after an escaped backslash", []string{`\\` + "`" + op + "`"}, []SpanComment{{0, -1}}},
		{"two-backtick span", []string{"``a ` " + op + " b` c``"}, []SpanComment{{0, -1}}},
		{"closer only on a fenced line", []string{"`" + op + "`", b3 + "text", "-->", b3, "### T"}, []SpanComment{{0, -1}}},
		{"closed after a fence", []string{"`" + op + "`", b3, "-->", b3, "x", "-->"}, []SpanComment{{0, 5}}},
		{"two openers in one span", []string{"`" + op + " --> " + op + " -->`"}, []SpanComment{{0, 0}, {0, 0}}},
		{"second opener on a later line", []string{"`" + op + "` -->", "`" + op + "`"}, []SpanComment{{0, 0}, {1, -1}}},
		{"comment holding a span", []string{op + " `x` -->", "### T"}, nil},
		{"backticks inside comments", []string{op + " ` --> " + op + " ` -->"}, nil},
		{"escaped backtick", []string{"\\`" + op + " a -->`"}, nil},
		{"opener in fenced code", []string{b3, "`" + op + "`", b3}, nil},
		{"single unmatched backtick", []string{"` " + op + " x -->"}, nil},
		{"unmatched run of two", []string{"`` " + op + " a ` -->"}, nil},
		{"unmatched run inside a closed span", []string{"`a `` b` " + op + " c -->"}, nil},
		{"opener before a span", []string{op + " x -->`y`"}, nil},
		{"span does not cross lines", []string{"`a", op + " b`", "-->"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CommentsInCodeSpans(tt.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CommentsInCodeSpans = %v, want %v", got, tt.want)
			}
		})
	}
}
