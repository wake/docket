package projects

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func pathCases(t *testing.T, fn func([]string) (string, error), xdg, fallback, name string) {
	t.Helper()
	home := "/h/" + fallback + "/docket/" + name
	cases := []struct {
		env  []string
		want string
	}{
		{[]string{xdg + "=/s"}, "/s/docket/" + name},
		{[]string{"HOME=/h"}, home},
		{[]string{xdg + "=rel", "HOME=/h"}, home},
		{[]string{"HOME=/x", "HOME=/h"}, home},
	}
	for _, c := range cases {
		got, err := fn(c.env)
		if err != nil || got != c.want {
			t.Errorf("%v: got %q, %v; want %q", c.env, got, err, c.want)
		}
	}
	if _, err := fn(nil); !errors.Is(err, ErrNoHome) {
		t.Errorf("empty env: %v", err)
	}
}

func TestIndexPath(t *testing.T) {
	pathCases(t, IndexPath, "XDG_STATE_HOME", ".local/state", "projects.json")
}

func TestSettingsPath(t *testing.T) {
	pathCases(t, SettingsPath, "XDG_CONFIG_HOME", ".config", "config.json")
}

func TestLoadMissing(t *testing.T) {
	x, err := Load(filepath.Join(t.TempDir(), "none.json"))
	if err != nil || x.Projects == nil || len(x.Projects) != 0 {
		t.Fatalf("%+v, %v", x, err)
	}
}

func write(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.json")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadInvalid(t *testing.T) {
	for _, s := range []string{
		"not json", "{}", `{"projects":"x"}`,
		`{"projects":[{"path":"rel","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"/p","origin":null,"added":"yesterday","ignored":false}]}`,
		`{"projects":[{"path":"/p","origin":null,"added":"2026-13-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"/p","origin":null,"added":"2026-11-08T17:10:11+08:00","ignored":false}]}`,
		`{"projects":[{"path":"/p","origin":null,"added":"2026-11-08T09:10:11Z","ignored":"no"}]}`,
	} {
		p := write(t, s)
		if _, err := Load(p); err == nil || !strings.Contains(err.Error(), p) {
			t.Errorf("%s: %v", s, err)
		}
	}
	valid := `{"projects":[{"path":"/p","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`
	if _, err := Load(write(t, valid)); err != nil {
		t.Errorf("valid: %v", err)
	}
}

func TestStamp(t *testing.T) {
	tm, _ := time.Parse(time.RFC3339Nano, "2026-11-08T17:10:11.5+08:00")
	if got := Stamp(tm); got != "2026-11-08T09:10:11Z" {
		t.Error(got)
	}
}

func TestLoadLink(t *testing.T) {
	p := write(t, `{"projects":[]}`)
	l := p + ".link"
	if err := os.Symlink(p, l); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(l); err == nil {
		t.Error("followed link")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "b", "projects.json")
	o := "git@x:y"
	in := &Index{Projects: []Entry{
		{Path: "/w/b", Origin: nil, Added: "2026-11-08T09:10:11Z"},
		{Path: "/w/a", Origin: &o, Added: "2026-11-08T09:10:11Z", Ignored: true},
	}}
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if !strings.HasSuffix(s, "}\n") || !strings.Contains(s, "\n  \"projects\": [") {
		t.Errorf("format: %q", s)
	}
	if fi, _ := os.Stat(p); fi.Mode()&0o077 != 0 {
		t.Errorf("mode %v", fi.Mode())
	}
	out, err := Load(p)
	if err != nil || len(out.Projects) != 2 || out.Projects[0].Path != "/w/a" || out.Projects[1].Path != "/w/b" {
		t.Fatalf("%+v %v", out, err)
	}
	if out.Projects[1].Origin != nil || *out.Projects[0].Origin != o || !out.Projects[0].Ignored {
		t.Errorf("%+v", out.Projects)
	}
	if out.Find("/w/b") != 1 || out.Find("/nope") != -1 {
		t.Error("Find")
	}
}

func TestAutoRegister(t *testing.T) {
	if v, err := AutoRegister(filepath.Join(t.TempDir(), "none")); !v || err != nil {
		t.Error(v, err)
	}
	for s, want := range map[string]bool{`{}`: true, `{"projects":{}}`: true, `{"projects":{"autoRegister":false}}`: false} {
		if v, err := AutoRegister(write(t, s)); v != want || err != nil {
			t.Errorf("%s: %v %v", s, v, err)
		}
	}
	for _, s := range []string{`{`, `{"projects":{"autoRegister":"no"}}`} {
		p := write(t, s)
		if _, err := AutoRegister(p); err == nil || !strings.Contains(err.Error(), p) {
			t.Errorf("%s: %v", s, err)
		}
	}
}
