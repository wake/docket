package delta_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/archive"
	"github.com/wake/docket/internal/delta"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/testutil"
)

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
			got := map[string]delta.Delta{}
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
				got[filepath.ToSlash(rel)] = delta.Parse(b)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			readings, ok, err := archive.CrossCheck(got, out)
			if err != nil {
				t.Fatal(err)
			}
			if len(readings.OpenSpec) == 0 {
				t.Fatalf("OpenSpec reported no deltas:\n%s", out)
			}
			if !ok {
				t.Errorf("mismatch\ndocket:   %s\nopenspec: %s",
					strings.Join(readings.Docket, "\n          "), strings.Join(readings.OpenSpec, "\n          "))
			}
		})
	}
}
