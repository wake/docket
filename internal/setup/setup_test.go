package setup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

var schema = fstest.MapFS{
	"schema.yaml":           {Data: []byte("name: docket\n")},
	"templates/proposal.md": {Data: []byte("# proposal\n")},
}

func statuses(r Result) map[string]FileStatus {
	m := map[string]FileStatus{}
	for _, f := range r.Files {
		m[f.Path] = f.Status
	}
	return m
}

func TestInitCallsOpenSpecOnlyWhenMissing(t *testing.T) {
	root := t.TempDir()
	calls := 0
	initOS := func() error { calls++; return os.MkdirAll(filepath.Join(root, "openspec"), 0o755) }
	r, err := Init(root, schema, false, initOS)
	if err != nil || calls != 1 || !r.OpenSpecInitialized {
		t.Fatalf("first: %v calls=%d %+v", err, calls, r)
	}
	r, err = Init(root, schema, false, initOS)
	if err != nil || calls != 1 || r.OpenSpecInitialized {
		t.Fatalf("second: %v calls=%d %+v", err, calls, r)
	}
	for p, s := range statuses(r) {
		if s != Unchanged {
			t.Errorf("%s = %s", p, s)
		}
	}
}

func TestInitStatuses(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Init(root, schema, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	const rel = "openspec/schemas/docket/templates/proposal.md"
	if got := statuses(r)[rel]; got != Written {
		t.Fatalf("first status = %s", got)
	}
	p := filepath.Join(root, rel)
	if err := os.WriteFile(p, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err = Init(root, schema, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := statuses(r)[rel]; got != Differs {
		t.Fatalf("status = %s", got)
	}
	if b, _ := os.ReadFile(p); string(b) != "edited\n" {
		t.Fatalf("file changed: %q", b)
	}
	r, err = Init(root, schema, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := statuses(r)[rel]; got != Overwritten {
		t.Fatalf("status = %s", got)
	}
	if b, _ := os.ReadFile(p); string(b) != "# proposal\n" {
		t.Fatalf("not overwritten: %q", b)
	}
}

func TestInitPropagatesOpenSpecError(t *testing.T) {
	boom := errors.New("boom")
	_, err := Init(t.TempDir(), schema, false, func() error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}
