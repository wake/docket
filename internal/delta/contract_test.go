package delta_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/delta"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/testutil"
)

type osReport struct {
	Deltas []struct {
		Spec        string `json:"spec"`
		Operation   string `json:"operation"`
		Requirement *struct {
			Name      string `json:"name"`
			Scenarios []struct {
				Name string `json:"name"`
			} `json:"scenarios"`
		} `json:"requirement"`
		Rename *struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"rename"`
	} `json:"deltas"`
}

func reqLine(op, name string, scenarios []string) string {
	return fmt.Sprintf("%s|%s|%q", op, name, scenarios)
}

func docketLines(b []byte) []string {
	d := delta.Parse(b)
	var out []string
	add := func(op string, rs []delta.Requirement) {
		for _, r := range rs {
			names := []string{}
			for _, s := range r.Scenarios {
				names = append(names, s.Name)
			}
			out = append(out, reqLine(op, r.Name, names))
		}
	}
	add("ADDED", d.Added)
	add("MODIFIED", d.Modified)
	add("REMOVED", d.Removed)
	for _, r := range d.Renamed {
		out = append(out, fmt.Sprintf("RENAMED|%s|%s", r.From, r.To))
	}
	return out
}

func openspecLines(t *testing.T, raw string) map[string][]string {
	t.Helper()
	var rep osReport
	if err := json.Unmarshal([]byte(raw), &rep); err != nil {
		t.Fatalf("parse OpenSpec output: %v\n%s", err, raw)
	}
	out := map[string][]string{}
	for _, d := range rep.Deltas {
		switch {
		case d.Rename != nil:
			out[d.Spec] = append(out[d.Spec], fmt.Sprintf("%s|%s|%s", d.Operation, d.Rename.From, d.Rename.To))
		case d.Requirement != nil:
			names := []string{}
			for _, s := range d.Requirement.Scenarios {
				names = append(names, s.Name)
			}
			out[d.Spec] = append(out[d.Spec], reqLine(d.Operation, d.Requirement.Name, names))
		}
	}
	return out
}

func copyChange(t *testing.T, src, dst string, transform func([]byte) []byte) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if transform != nil && info.Name() == "spec.md" {
			b = transform(b)
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func flat(m map[string][]string) string {
	var out []string
	for k, v := range m {
		for _, l := range v {
			out = append(out, k+": "+l)
		}
	}
	return strings.Join(out, "\n          ")
}

func TestDeltasAgreeWithOpenSpec(t *testing.T) {
	cmd := os.Getenv("DOCKET_TEST_OPENSPEC")
	if cmd == "" {
		t.Skip("DOCKET_TEST_OPENSPEC is not set")
	}
	type tc struct {
		name, dir string
		transform func([]byte) []byte
	}
	var cases []tc
	entries, err := os.ReadDir("testdata/deltas")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			cases = append(cases, tc{e.Name(), e.Name(), nil})
		}
	}
	cases = append(cases,
		tc{"basic-crlf", "basic", func(b []byte) []byte { return bytes.ReplaceAll(b, []byte("\n"), []byte("\r\n")) }},
		tc{"basic-bom", "basic", func(b []byte) []byte { return append([]byte("\xef\xbb\xbf"), b...) }},
	)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmp := t.TempDir()
			change := filepath.Join(tmp, "openspec", "changes", c.name)
			copyChange(t, filepath.Join("testdata", "deltas", c.dir), change, c.transform)
			r, err := openspec.New(tmp, append(testutil.Env(t), "DOCKET_OPENSPEC="+cmd))
			if err != nil {
				t.Fatal(err)
			}
			out, err := r.Run("show", c.name, "--type", "change", "--json", "--deltas-only")
			if err != nil {
				t.Fatal(err)
			}
			want := openspecLines(t, out)
			if len(want) == 0 {
				t.Fatalf("OpenSpec reported no deltas:\n%s", out)
			}
			got := map[string][]string{}
			root := filepath.Join(change, "specs")
			err = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() || info.Name() != "spec.md" {
					return err
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(root, filepath.Dir(p))
				if lines := docketLines(b); len(lines) > 0 {
					got[filepath.ToSlash(rel)] = lines
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("mismatch\ndocket:   %s\nopenspec: %s", flat(got), flat(want))
			}
		})
	}
}
