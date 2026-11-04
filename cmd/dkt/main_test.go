package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
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
