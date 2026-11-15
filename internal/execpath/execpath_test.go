package execpath

import (
	"io/fs"
	"slices"
	"testing"
)

func TestNames(t *testing.T) {
	for _, c := range []struct {
		windows bool
		name    string
		env     []string
		want    []string
	}{
		{false, "git", []string{"PATHEXT=.EXE"}, []string{"git"}},
		{false, "git.exe", nil, []string{"git.exe"}},
		{true, "git", nil, []string{"git.com", "git.exe", "git.bat", "git.cmd"}},
		{true, "git", []string{"PATHEXT="}, []string{"git.com", "git.exe", "git.bat", "git.cmd"}},
		{true, "openspec", []string{"PATHEXT=.CMD;.Exe"}, []string{"openspec.cmd", "openspec.exe"}},
		{true, "openspec", []string{"PATHEXT=EXE;;cmd"}, []string{"openspec.exe", "openspec.cmd"}},
		{true, `C:\tools\openspec.EXE`, []string{"PATHEXT=.exe;.cmd"}, []string{`C:\tools\openspec.EXE`}},
		{true, "open.spec", []string{"PATHEXT=.exe"}, []string{"open.spec.exe"}},
		{true, "git", []string{"PathExt=.cmd", "PATHEXT=.exe"}, []string{"git.exe"}},
	} {
		if got := names(c.windows, c.name, c.env); !slices.Equal(got, c.want) {
			t.Errorf("names(%v, %q, %q) = %q, want %q", c.windows, c.name, c.env, got, c.want)
		}
	}
}

func TestGetenv(t *testing.T) {
	env := []string{"PATH=/a", "Path=/b", "OTHER=x"}
	if v, ok := getenv(true, env, "PATH"); v != "/b" || !ok {
		t.Errorf("windows, last spelling wins: %q %v", v, ok)
	}
	if v, ok := getenv(false, env, "PATH"); v != "/a" || !ok {
		t.Errorf("elsewhere, exact name: %q %v", v, ok)
	}
	if v, ok := getenv(true, []string{"Path=/b", "PATH=/c"}, "PATH"); v != "/c" || !ok {
		t.Errorf("windows, last of two: %q %v", v, ok)
	}
	if _, ok := getenv(false, []string{"Path=/b"}, "PATH"); ok {
		t.Error("elsewhere, Path is not PATH")
	}
}

func TestMayRun(t *testing.T) {
	for _, c := range []struct {
		windows bool
		mode    uint32
		want    bool
	}{
		{false, 0o644, false}, {false, 0o744, true}, {false, 0o601, true},
		{true, 0o666, true}, {true, 0o444, true},
	} {
		if got := mayRun(c.windows, fs.FileMode(c.mode)); got != c.want {
			t.Errorf("mayRun(%v, %o) = %v", c.windows, c.mode, got)
		}
	}
}
