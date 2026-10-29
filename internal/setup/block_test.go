package setup

import (
	"errors"
	"testing"
)

const testBlock = "<!-- docket:start -->\nB\n<!-- docket:end -->"

func TestPlaceBlock(t *testing.T) {
	old := "# A\n<!-- docket:start -->\nold\n<!-- docket:end -->\nafter\n"
	cases := []struct {
		name   string
		cur    []byte
		force  bool
		next   []byte
		status FileStatus
		err    bool
	}{
		{"missing", nil, false, []byte(testBlock + "\n"), Written, false},
		{"empty file", []byte(""), false, []byte(testBlock + "\n"), Written, false},
		{"no final newline", []byte("# A"), false, []byte("# A\n\n" + testBlock + "\n"), Written, false},
		{"ends with newline", []byte("# A\n"), false, []byte("# A\n\n" + testBlock + "\n"), Written, false},
		{"identical", []byte("# A\n\n" + testBlock + "\nmore\n"), false, nil, Unchanged, false},
		{"differs", []byte(old), false, nil, Differs, false},
		{"differs, force", []byte(old), true, []byte("# A\n" + testBlock + "\nafter\n"), Overwritten, false},
		{"no end", []byte("<!-- docket:start -->\nx\n"), false, nil, "", true},
		{"end first", []byte("<!-- docket:end -->\n<!-- docket:start -->\n"), false, nil, "", true},
		{"two blocks", []byte(testBlock + "\n" + testBlock + "\n"), false, nil, "", true},
		{"indented marker", []byte("  <!-- docket:start -->\n"), false, []byte("  <!-- docket:start -->\n\n" + testBlock + "\n"), Written, false},
		{"CRLF copy of the block", []byte("<!-- docket:start -->\r\nB\r\n<!-- docket:end -->\r\n"), false, nil, Differs, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next, status, err := placeBlock(c.cur, testBlock, c.force)
			if c.err {
				if !errors.Is(err, errMalformed) {
					t.Fatalf("err = %v, want errMalformed", err)
				}
			} else if err != nil {
				t.Fatalf("unexpected err %v", err)
			}
			if status != c.status {
				t.Errorf("status = %q, want %q", status, c.status)
			}
			if (next == nil) != (c.next == nil) || string(next) != string(c.next) {
				t.Errorf("next = %q, want %q", next, c.next)
			}
		})
	}
}

func TestFindBlock(t *testing.T) {
	b := []byte("x\n" + testBlock + "\ny\n")
	from, to, found, err := FindBlock(b)
	if err != nil || !found || from != 2 || to != 2+len(testBlock) {
		t.Fatalf("got %d %d %v %v", from, to, found, err)
	}
	if _, _, found, err := FindBlock([]byte("x\n")); found || err != nil {
		t.Fatalf("no marker: found=%v err=%v", found, err)
	}
	for name, in := range map[string]string{
		"no end":     "<!-- docket:start -->\n",
		"end first":  "<!-- docket:end -->\n<!-- docket:start -->\n",
		"two":        testBlock + "\n" + testBlock + "\n",
		"two starts": "<!-- docket:start -->\n<!-- docket:start -->\n<!-- docket:end -->\n",
		"two ends":   "<!-- docket:start -->\n<!-- docket:end -->\n<!-- docket:end -->\n",
	} {
		if _, _, _, err := FindBlock([]byte(in)); !errors.Is(err, errMalformed) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestImportsAgents(t *testing.T) {
	for in, want := range map[string]bool{
		"@AGENTS.md\n":            true,
		"# x\n  @./AGENTS.md\t\n": true,
		"See @AGENTS.md here\n":   false,
		"@AGENTS.md.bak\n":        false,
		"":                        false,
	} {
		if got := importsAgents([]byte(in)); got != want {
			t.Errorf("%q = %v, want %v", in, got, want)
		}
	}
}
