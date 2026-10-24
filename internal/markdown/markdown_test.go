package markdown

import (
	"reflect"
	"regexp"
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

func TestTrim(t *testing.T) {
	if got := Trim("\ufeff Why\u2028"); got != "Why" {
		t.Fatalf("Trim = %q, want U+FEFF and U+2028 removed", got)
	}
	if got := Trim("\u0085x\u0085"); got != "\u0085x\u0085" {
		t.Fatalf("Trim = %q, want U+0085 kept", got)
	}
	re := regexp.MustCompile("^" + Space + "$")
	for r := rune(0); r <= 0x10ffff; r++ {
		in := re.MatchString(string(r))
		if in != isSpace(r) {
			t.Fatalf("isSpace(%U) disagrees with Space", r)
		}
		if in && Trim("a"+string(r)) != "a" {
			t.Fatalf("%U not trimmed", r)
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
