package projects

import (
	"errors"
	"github.com/wake/docket/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/wake/docket/internal/execpath"
	"github.com/wake/docket/internal/gitx"
)

func pathCases(t *testing.T, fn func([]string) (string, error), xdg, fallback, name string) {
	t.Helper()
	s, h, x := t.TempDir(), t.TempDir(), t.TempDir()
	hv := homeVar(isWindows)
	home := filepath.Join(h, filepath.FromSlash(fallback), "docket", name)
	cases := []struct {
		env  []string
		want string
	}{
		{[]string{xdg + "=" + s}, filepath.Join(s, "docket", name)},
		{[]string{hv + "=" + h}, home},
		{[]string{xdg + "=rel", hv + "=" + h}, home},
		{[]string{hv + "=" + x, hv + "=" + h}, home},
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

func TestHomeVar(t *testing.T) {
	if homeVar(true) != "USERPROFILE" || homeVar(false) != "HOME" {
		t.Errorf("windows %q, elsewhere %q", homeVar(true), homeVar(false))
	}
}

// The rule is os.UserHomeDir's, read from the given environment instead of the process's.
func TestHomeVarMatchesUserHomeDir(t *testing.T) {
	want, err := os.UserHomeDir()
	if err != nil {
		t.Skip("this process has no home folder:", err)
	}
	if got, _ := execpath.Getenv(os.Environ(), homeVar(isWindows)); got != want {
		t.Errorf("%s gives %q, os.UserHomeDir %q", homeVar(isWindows), got, want)
	}
}

func TestUserFoldersElsewhere(t *testing.T) {
	if isWindows {
		t.Skip("the rule for macOS and Linux")
	}
	h, u := t.TempDir(), t.TempDir()
	if got, err := IndexPath([]string{"HOME=" + h, "USERPROFILE=" + u}); err != nil || got != filepath.Join(h, ".local", "state", "docket", "projects.json") {
		t.Errorf("HOME and USERPROFILE: %q, %v", got, err)
	}
	if _, err := IndexPath([]string{"USERPROFILE=" + u}); !errors.Is(err, ErrNoHome) {
		t.Errorf("USERPROFILE only: %v", err)
	}
	if !strings.HasSuffix(ErrNoHome.Error(), "set HOME") {
		t.Errorf("message %q", ErrNoHome)
	}
}

func TestUserFoldersOnWindows(t *testing.T) {
	if !isWindows {
		t.Skip("the rule for Windows")
	}
	h, u, s, c := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	if _, err := IndexPath([]string{"HOME=" + h}); !errors.Is(err, ErrNoHome) {
		t.Errorf("HOME only: %v", err)
	}
	if got, err := IndexPath([]string{"HOME=" + h, "UserProfile=" + u}); err != nil || got != filepath.Join(u, ".local", "state", "docket", "projects.json") {
		t.Errorf("UserProfile: %q, %v", got, err)
	}
	if got, err := SessionsDir([]string{"xdg_state_home=" + s}); err != nil || got != filepath.Join(s, "docket", "sessions") {
		t.Errorf("xdg_state_home: %q, %v", got, err)
	}
	if got, err := UserWorkflowPath([]string{"Xdg_Config_Home=" + c}); err != nil || got != filepath.Join(c, "docket", "workflow.md") {
		t.Errorf("Xdg_Config_Home: %q, %v", got, err)
	}
	if !strings.HasSuffix(ErrNoHome.Error(), "set USERPROFILE") {
		t.Errorf("message %q", ErrNoHome)
	}
}

// Register names the system's home variable when it finds no folder.
func TestRegisterNoHomeNamesVariable(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docket"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docket", "workflow.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Register(root, gitx.Git{Dir: root}, nil, time.Now())
	if err == nil || !strings.Contains(err.Error(), "set "+homeVar(isWindows)+", XDG_CONFIG_HOME") {
		t.Errorf("%v", err)
	}
}

func TestIndexPath(t *testing.T) {
	pathCases(t, IndexPath, "XDG_STATE_HOME", ".local/state", "projects.json")
}

func TestSessionsDir(t *testing.T) {
	pathCases(t, SessionsDir, "XDG_STATE_HOME", ".local/state", "sessions")
}

func TestSettingsPath(t *testing.T) {
	pathCases(t, SettingsPath, "XDG_CONFIG_HOME", ".config", "config.json")
}

func TestUserWorkflowPath(t *testing.T) {
	pathCases(t, UserWorkflowPath, "XDG_CONFIG_HOME", ".config", "workflow.md")
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

// drive makes the Unix paths of the test data absolute on Windows too.
var drive = map[bool]string{false: "", true: "C:"}[runtime.GOOS == "windows"]

func TestLoadRemovesUserInfo(t *testing.T) {
	p := write(t, `{"projects":[
{"path":"`+drive+`/w/a","origin":"https://user:p%zz@example.com/org/app.git","added":"2026-11-08T09:10:11Z","ignored":false},
{"path":"`+drive+`/w/b","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`)
	x, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if o := x.Projects[0].Origin; o == nil || *o != "https://example.com/org/app.git" {
		t.Fatalf("origin = %v", o)
	}
	if x.Projects[1].Origin != nil {
		t.Fatalf("origin = %v, want nil", x.Projects[1].Origin)
	}
	out := filepath.Join(t.TempDir(), "out.json")
	if err := x.Save(out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "p%zz") {
		t.Fatalf("saved file keeps the token: %s", b)
	}
}

func TestLoadInvalid(t *testing.T) {
	for _, s := range []string{
		"not json", "{}", `{"projects":"x"}`,
		`{"projects":[{"path":"rel","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":null,"added":"yesterday","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":null,"added":"2026-13-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":null,"added":"2026-11-08T17:10:11+08:00","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":null,"added":"2026-11-08T09:10:11Z","ignored":"no"}]}`,
		`{"projects":[{"path":"` + drive + `/p","added":"2026-11-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":null,"added":"2026-11-08T09:10:11Z"}]}`,
		`{"projects":[{"path":"` + drive + `/p","origin":3,"added":"2026-11-08T09:10:11Z","ignored":false}]}`,
		`{"projects":[{"path":"` + drive + `/w/app","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false},{"path":"` + drive + `/w/app","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`,
	} {
		p := write(t, s)
		if _, err := Load(p); err == nil || !strings.Contains(err.Error(), p) {
			t.Errorf("%s: %v", s, err)
		}
	}
	valid := `{"projects":[{"path":"` + drive + `/p","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`
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
		{Path: drive + "/w/b", Origin: nil, Added: "2026-11-08T09:10:11Z"},
		{Path: drive + "/w/a", Origin: &o, Added: "2026-11-08T09:10:11Z", Ignored: true},
	}}
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	s := string(b)
	if !strings.HasSuffix(s, "}\n") || !strings.Contains(s, "\n  \"projects\": [") {
		t.Errorf("format: %q", s)
	}
	if fi, _ := os.Stat(p); testutil.ModeBits && fi.Mode()&0o077 != 0 {
		t.Errorf("mode %v", fi.Mode())
	}
	out, err := Load(p)
	if err != nil || len(out.Projects) != 2 || out.Projects[0].Path != drive+"/w/a" || out.Projects[1].Path != drive+"/w/b" {
		t.Fatalf("%+v %v", out, err)
	}
	if out.Projects[1].Origin != nil || *out.Projects[0].Origin != o || !out.Projects[0].Ignored {
		t.Errorf("%+v", out.Projects)
	}
	if out.Find(drive+"/w/b") != 1 || out.Find("/nope") != -1 {
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

func TestEntryModeSynced(t *testing.T) {
	p := write(t, `{"projects":[
{"path":"`+drive+`/w/a","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false,"mode":"sync","synced":{"Plan review":"ab"}},
{"path":"`+drive+`/w/b","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false}]}`)
	x, err := Load(p)
	if err != nil || x.Projects[0].Mode != "sync" || x.Projects[0].Synced["Plan review"] != "ab" || x.Projects[1].Mode != "" {
		t.Fatalf("%+v %v", x, err)
	}
	out := filepath.Join(t.TempDir(), "o.json")
	if err := x.Save(out); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if strings.Count(string(b), `"mode"`) != 1 || strings.Count(string(b), `"synced"`) != 1 {
		t.Errorf("keys: %s", b)
	}
	y, err := Load(out)
	if err != nil || y.Projects[0].Mode != "sync" || y.Projects[0].Synced["Plan review"] != "ab" {
		t.Errorf("%+v %v", y, err)
	}
	bad := write(t, `{"projects":[{"path":"`+drive+`/w/a","origin":null,"added":"2026-11-08T09:10:11Z","ignored":false,"mode":"always"}]}`)
	if _, err := Load(bad); err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("unknown mode: %v", err)
	}
}

func TestDefaultMode(t *testing.T) {
	if v, err := DefaultMode(filepath.Join(t.TempDir(), "none")); v != "" || err != nil {
		t.Error(v, err)
	}
	for s, want := range map[string]string{`{"workflow":{"default":"sync"}}`: "sync", `{"workflow":{"default":"independent"}}`: "independent", `{}`: "", `{"workflow":{}}`: ""} {
		if v, err := DefaultMode(write(t, s)); v != want || err != nil {
			t.Errorf("%s: %q %v", s, v, err)
		}
	}
	for _, s := range []string{`{"workflow":{"default":"always"}}`, `not json`} {
		p := write(t, s)
		if _, err := DefaultMode(p); err == nil || !strings.Contains(err.Error(), p) {
			t.Errorf("%s: %v", s, err)
		}
	}
}

func TestAutoRegisterLink(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "real.json")
	if err := os.WriteFile(target, []byte(`{"projects":{"autoRegister":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	l := filepath.Join(d, "config.json")
	if err := os.Symlink(target, l); err != nil {
		t.Fatal(err)
	}
	if on, err := AutoRegister(l); err != nil || on {
		t.Errorf("%v %v", on, err)
	}
}
