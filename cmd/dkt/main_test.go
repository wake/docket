package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestReleaseBuild(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "dkt")
	build := exec.Command("go", "build", "-ldflags", "-X github.com/wake/docket/internal/cli.Version=v0.16.0", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	run := func(args ...string) string {
		var out bytes.Buffer
		c := exec.Command(bin, args...)
		c.Stdout = &out
		if err := c.Run(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return out.String()
	}
	if got := run("--version"); got != "dkt v0.16.0\n" {
		t.Errorf("text = %q", got)
	}
	var e struct {
		Version string `json:"version"`
		Result  struct {
			Version string `json:"version"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(run("--version", "--json")), &e); err != nil {
		t.Fatal(err)
	}
	if e.Version != "v0.16.0" || e.Result.Version != "v0.16.0" {
		t.Errorf("json = %+v", e)
	}
}

// copyTree copies the named paths of the module (relative to src) into dst.
func copyTree(t *testing.T, src, dst string, names ...string) {
	t.Helper()
	for _, n := range names {
		err := filepath.WalkDir(filepath.Join(src, n), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(src, p)
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(dst, rel), 0o755)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func envValue(env []string, key string) string {
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}

// A clean checkout at a release tag, built with VCS stamping on, reports the
// tag: main takes the version from the build info.
func TestBuildInfoVersion(t *testing.T) {
	env := testutil.Env(t)
	src := t.TempDir()
	copyTree(t, "../..", src, "go.mod", "cmd", "internal")
	r := &testutil.Repo{T: t, Dir: src, Env: env}
	r.Git("init", "-q")
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "snapshot")
	r.Git("tag", "v1.2.3")
	bin := filepath.Join(t.TempDir(), "dkt") // outside the repository: keeps the tree clean
	build := exec.Command("go", "build", "-buildvcs=true", "-o", bin, "./cmd/dkt")
	build.Dir = src
	// The Go toolchain keeps the process environment minus every GIT_*
	// variable; git itself is isolated.
	var base []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "GIT_") {
			base = append(base, kv)
		}
	}
	build.Env = append(base, "GOFLAGS=", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+envValue(env, "GIT_CONFIG_GLOBAL"))
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	got, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "dkt v1.2.3\n" {
		t.Errorf("--version = %q, want %q", got, "dkt v1.2.3\n")
	}
}
