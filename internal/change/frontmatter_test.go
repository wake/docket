package change

import (
	"strings"
	"testing"
)

func read(t *testing.T, s string) Front {
	t.Helper()
	fm, ok, err := ReadFrontMatter([]byte(s))
	if err != nil || !ok {
		t.Fatalf("ReadFrontMatter(%q) = ok %v, err %v", s, ok, err)
	}
	return fm
}

func TestReadFrontMatterValues(t *testing.T) {
	cases := []struct{ name, value, want string }{
		{"quoted title", `"Booking: waitlist \"v2\""`, `Booking: waitlist "v2"`},
		{"backslash", `"a\\b"`, `a\b`},
		{"tab", `"a\tb"`, "a\tb"},
		{"slash", `"a\/b"`, "a/b"},
		{"esc", `"\e"`, "\x1b"},
		{"next line", `"\N"`, "\u0085"},
		{"nbsp", `"\_"`, "\u00a0"},
		{"line sep", `"\L\P"`, "\u2028\u2029"},
		{"escaped space", `"a\ b"`, "a b"},
		{"hex", `"\x41\u00e9\U0001F600"`, "Aé😀"},
		{"literal accent", `"é"`, "é"},
		{"single quoted", `'it''s'`, "it's"},
		{"single quoted comment", `'x' # note`, "x"},
		{"double quoted comment", `"x"   # note`, "x"},
		{"plain", `  plain value  `, "plain value"},
		{"plain comment", `plain # comment`, "plain"},
		{"plain hash kept", `a#b`, "a#b"},
		{"plain colon", `Booking: waitlist`, "Booking: waitlist"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fm := read(t, "---\ntitle: "+c.value+"\n---\n")
			if got := fm.Fields["title"].Value; got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestReadFrontMatterSpecs(t *testing.T) {
	fm := read(t, "---\ntitle: T\nspecs:\n  booking: Booking Waitlist\n  identity/user-auth: \"User Auth\"\nkind: feature\n---\n")
	if fm.Specs["booking"] != "Booking Waitlist" || fm.Specs["identity/user-auth"] != "User Auth" || len(fm.Specs) != 2 {
		t.Errorf("Specs = %v", fm.Specs)
	}
	if fm.SpecsLine != 3 {
		t.Errorf("SpecsLine = %d", fm.SpecsLine)
	}
	if fm.Fields["kind"].Value != "feature" {
		t.Errorf("kind = %v", fm.Fields["kind"])
	}
}

func TestReadFrontMatterLines(t *testing.T) {
	fm := read(t, "\xef\xbb\xbf---\r\n# comment\r\n\r\ntitle: A\r\n  \r\nkind: feature\r\nsource: x\r\n---\r\n")
	for k, want := range map[string]int{"title": 4, "kind": 6, "source": 7} {
		if got := fm.Fields[k].Line; got != want {
			t.Errorf("%s line = %d, want %d", k, got, want)
		}
	}
}

func TestReadFrontMatterNone(t *testing.T) {
	for _, s := range []string{"# Proposal\n", "", " ---\n", "----\n"} {
		_, ok, err := ReadFrontMatter([]byte(s))
		if ok || err != nil {
			t.Errorf("%q: ok %v, err %v", s, ok, err)
		}
	}
}

func TestReadFrontMatterErrors(t *testing.T) {
	cases := map[string]string{
		"unclosed":         "---\ntitle: A\n",
		"unknown escape":   "---\ntitle: \"a\\qb\"\n---\n",
		"unclosed quote":   "---\ntitle: \"abc\n---\n",
		"unclosed single":  "---\ntitle: 'abc\n---\n",
		"trailing text":    "---\ntitle: \"a\" b\n---\n",
		"duplicate key":    "---\ntitle: A\ntitle: B\n---\n",
		"duplicate spec":   "---\nspecs:\n  a: A\n  a: B\n---\n",
		"not key value":    "---\njust text\n---\n",
		"empty value":      "---\ntitle:\n---\n",
		"specs with value": "---\nspecs: x\n---\n",
		"indent outside":   "---\ntitle: A\n  booking: B\n---\n",
		"indent after key": "---\nspecs:\n  a: A\nkind: x\n  b: B\n---\n",
		"bad spec line":    "---\nspecs:\n  nocolon\n---\n",
		"bad hex":          "---\ntitle: \"\\xZZ\"\n---\n",
		"short unicode":    "---\ntitle: \"\\u12\"\n---\n",
		"bad key":          "---\n1a: x\n---\n",
	}
	for name, s := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok, err := ReadFrontMatter([]byte(s))
			if err == nil {
				t.Fatalf("no error (ok %v)", ok)
			}
			if name != "unclosed" && !strings.Contains(err.Error(), "line ") {
				t.Errorf("error names no line: %v", err)
			}
		})
	}
}

func TestReadFrontMatterRoundTrip(t *testing.T) {
	for _, title := range []string{`Booking: waitlist "v2"`, `a\b`, "a\tb", "é 😀", "it's # not", "\u00a0x", " lead"} {
		fm := read(t, string(FrontMatter(title, "feature", "src \"q\"")))
		if fm.Fields["title"].Value != title || fm.Fields["kind"].Value != "feature" || fm.Fields["source"].Value != `src "q"` {
			t.Errorf("%q read back as %v", title, fm.Fields)
		}
	}
}

func TestSpecTitle(t *testing.T) {
	cases := []struct {
		first, cap, want string
		ok               bool
	}{
		{"# booking Specification", "booking", "", false},
		{"# identity/user-auth Specification", "identity/user-auth", "", false},
		{"# Booking Specification", "booking", "Booking", true},
		{"# Booking Waitlist Specification", "booking", "Booking Waitlist", true},
		{"# Booking", "booking", "", false},
		{"", "booking", "", false},
	}
	for _, c := range cases {
		got, ok := SpecTitle(c.first, c.cap)
		if got != c.want || ok != c.ok {
			t.Errorf("SpecTitle(%q, %q) = %q, %v", c.first, c.cap, got, ok)
		}
	}
}
