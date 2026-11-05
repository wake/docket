package docket

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

// A docket module small enough to build quickly: its dkt prints the stamped
// version, then its arguments.
var fakeDocket = map[string]string{
	"go.mod":              "module github.com/wake/docket\n\ngo 1.24\n",
	"internal/cli/cli.go": "package cli\n\nvar Version = \"dev\"\n",
	"cmd/dkt/main.go": `package main

import (
	"fmt"
	"os"

	"github.com/wake/docket/internal/cli"
)

func main() { fmt.Println(cli.Version, os.Args[1:]) }
`,
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDevWrapper(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the wrapper is a POSIX shell script")
	}
	wrapper, err := filepath.Abs("tools/dev-bin/dkt")
	if err != nil {
		t.Fatal(err)
	}
	gocache, err := exec.Command("go", "env", "GOCACHE").Output()
	if err != nil {
		t.Fatal(err)
	}
	release := t.TempDir()
	writeFiles(t, release, map[string]string{"dkt": "#!/bin/sh\necho release \"$@\"\n"})

	r := testutil.NewRepo(t)
	var env []string
	for _, kv := range r.Env {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			kv = "PATH=" + filepath.Dir(wrapper) + ":" + release + ":" + v
		}
		env = append(env, kv)
	}
	env = append(env, "GOCACHE="+strings.TrimSpace(string(gocache)))
	dkt := func(dir string) string {
		t.Helper()
		cmd := exec.Command(wrapper, "a b")
		cmd.Dir, cmd.Env = dir, env
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("dkt in %s: %v", dir, err)
		}
		return strings.TrimSpace(string(out))
	}
	short := func() string { return r.Git("rev-parse", "--short", "main") }

	writeFiles(t, r.Dir, fakeDocket)
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "one")
	if got, want := dkt(r.Dir), "main@"+short()+" [a b]"; got != want {
		t.Errorf("first run = %q, want %q", got, want)
	}

	// main moves on; run in a worktree on another branch, with the code edited
	// there and in the main worktree, it still builds the main commit.
	r.Git("commit", "-q", "--allow-empty", "-m", "two")
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", "-b", "feat", wt)
	edited := map[string]string{"cmd/dkt/main.go": "package main\n\nfunc main() { println(\"edited\") }\n"}
	writeFiles(t, wt, edited)
	writeFiles(t, r.Dir, edited)
	if got, want := dkt(wt), "main@"+short()+" [a b]"; got != want {
		t.Errorf("after main moved = %q, want %q", got, want)
	}

	// Outside this repository: a plain directory, and a repository of another project.
	other := testutil.NewRepo(t)
	for _, dir := range []string{t.TempDir(), other.Dir} {
		if got, want := dkt(dir), "release a b"; got != want {
			t.Errorf("in %s = %q, want %q", dir, got, want)
		}
	}
}
