package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const four = "## Full review\n## Light review\n## Check\n## Direct merge\n"

func fm(h string) string { return "---\ntemplate: " + h + "\n---\n" }

func TestInspect(t *testing.T) {
	const h = "abcdefabcdef"
	all := []string{"Full review", "Light review", "Check", "Direct merge"}
	none := []string{}
	noCheck := "## Full review\n## Light review\n## Direct merge\n"
	ptr := func(s string) *string { return &s }
	cases := []struct {
		name   string
		in     string
		status Status
		rec    *string
		miss   []string
	}{
		{"complete", fm(h) + four, OK, ptr(h), none},
		{"quoted", fm("\"abcdefabcdef\"") + four, OK, ptr(h), none},
		{"crlf", "---\r\ntemplate: abcdefabcdef\r\n---\r\n## Full review  \r\n## Light review\t\r\n## Check\r\n## Direct merge\r\n", OK, ptr(h), none},
		{"bom", "\xef\xbb\xbf" + fm(h) + four, OK, ptr(h), none},
		{"digits", fm("012345678901") + four, Stale, ptr("012345678901"), none},
		{"older", fm("0123456789ab") + four, Stale, ptr("0123456789ab"), none},
		{"older no check", fm("0123456789ab") + noCheck, Incomplete, ptr("0123456789ab"), []string{"Check"}},
		{"no headings", fm(h) + "# x\n", Incomplete, ptr(h), all},
		{"near misses", fm(h) + "## Full review\n## Light review\n## Checklist\n ## Check\n## Direct merge\n", Incomplete, ptr(h), []string{"Check"}},
		{"no frontmatter", four, Stale, nil, none},
		{"unclosed", "---\ntemplate: abcdefabcdef\n" + four, Stale, nil, none},
		{"invalid yaml", fm("[") + four, Stale, nil, none},
		{"null", fm("~") + four, Stale, nil, none},
		{"no key", "---\nother: x\n---\n" + four, Stale, nil, none},
		{"empty", "", Incomplete, nil, all},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Inspect([]byte(c.in), h)
			if r.Status != c.status {
				t.Errorf("status = %q, want %q", r.Status, c.status)
			}
			if !reflect.DeepEqual(r.Recorded, c.rec) {
				t.Errorf("recorded = %v, want %v", r.Recorded, c.rec)
			}
			if r.MissingSections == nil || !reflect.DeepEqual(r.MissingSections, c.miss) {
				t.Errorf("missing = %#v, want %#v", r.MissingSections, c.miss)
			}
		})
	}
}

func TestLocate(t *testing.T) {
	home := func(p string) func() (string, error) { return func() (string, error) { return p, nil } }
	cases := []struct {
		name string
		env  []string
		look func() (string, error)
		want string
	}{
		{"home", []string{"HOME=/h"}, nil, "/h/.config/docket/workflow.md"},
		{"xdg", []string{"HOME=/h", "XDG_CONFIG_HOME=/x"}, nil, "/x/docket/workflow.md"},
		{"xdg slash", []string{"HOME=/h", "XDG_CONFIG_HOME=/x/"}, nil, "/x/docket/workflow.md"},
		{"xdg relative", []string{"HOME=/h", "XDG_CONFIG_HOME=rel"}, nil, "/h/.config/docket/workflow.md"},
		{"xdg empty", []string{"HOME=/h", "XDG_CONFIG_HOME="}, nil, "/h/.config/docket/workflow.md"},
		{"home twice", []string{"HOME=/a", "HOME=/b"}, nil, "/b/.config/docket/workflow.md"},
		{"xdg twice", []string{"HOME=/h", "XDG_CONFIG_HOME=/x", "XDG_CONFIG_HOME=/y"}, nil, "/y/docket/workflow.md"},
		{"xdg then relative", []string{"HOME=/h", "XDG_CONFIG_HOME=/x", "XDG_CONFIG_HOME=rel"}, nil, "/h/.config/docket/workflow.md"},
		{"relative home", []string{"HOME=rel"}, home("/u"), "/u/.config/docket/workflow.md"},
		{"no home", nil, home("/u"), "/u/.config/docket/workflow.md"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Locate(c.env, c.look)
			if err != nil {
				t.Fatal(err)
			}
			if want := filepath.Join(c.want); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
	bad := map[string]func() (string, error){
		"nil":      nil,
		"error":    func() (string, error) { return "", errors.New("boom") },
		"empty":    home(""),
		"relative": home("rel"),
	}
	for name, look := range bad {
		t.Run("no home "+name, func(t *testing.T) {
			if _, err := Locate(nil, look); !errors.Is(err, ErrNoHome) {
				t.Errorf("err = %v, want ErrNoHome", err)
			}
		})
	}
}

func TestHash(t *testing.T) {
	if got := Hash(""); got != "e3b0c44298fc" {
		t.Errorf("Hash(\"\") = %q", got)
	}
	if len(Hash("x")) != 12 {
		t.Errorf("len(Hash(x)) = %d", len(Hash("x")))
	}
}

func TestRead(t *testing.T) {
	d := t.TempDir()
	if b, found, err := Read(filepath.Join(d, "none")); found || err != nil || b != nil {
		t.Errorf("missing: %q %v %v", b, found, err)
	}
	f := filepath.Join(d, "f")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, found, err := Read(f); !found || err != nil || string(b) != "hi" {
		t.Errorf("file: %q %v %v", b, found, err)
	}
	ln := filepath.Join(d, "ln")
	if err := os.Symlink(f, ln); err != nil {
		t.Fatal(err)
	}
	if b, found, err := Read(ln); !found || err != nil || string(b) != "hi" {
		t.Errorf("link: %q %v %v", b, found, err)
	}
	dang := filepath.Join(d, "dang")
	if err := os.Symlink(filepath.Join(d, "nowhere"), dang); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(dang); err == nil {
		t.Error("dangling link: want error")
	}
	if _, _, err := Read(d); err == nil {
		t.Error("directory: want error")
	}
	gone := filepath.Join(d, "gone")
	badDir := filepath.Join(d, "baddir")
	if err := os.Symlink(gone, badDir); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Read(filepath.Join(badDir, "workflow.md")); err == nil || found {
		t.Errorf("parent link to a missing directory: found %v err %v", found, err)
	}
	realDir := filepath.Join(d, "realdir")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	okDir := filepath.Join(d, "okdir")
	if err := os.Symlink(realDir, okDir); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Read(filepath.Join(okDir, "workflow.md")); err != nil || found {
		t.Errorf("parent link to an existing directory: found %v err %v", found, err)
	}
}
