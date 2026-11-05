package docket

import (
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func readShipped(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestShippedFileSet(t *testing.T) {
	var got []string
	err := fs.WalkDir(os.DirFS("."), "shipped", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			got = append(got, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"shipped/config/operations.yaml",
		"shipped/next-steps/commit.md",
		"shipped/next-steps/doctor.md",
		"shipped/next-steps/reload.md",
		"shipped/schema/schema.yaml",
		"shipped/schema/templates/design.md",
		"shipped/schema/templates/plan.md",
		"shipped/schema/templates/proposal.md",
		"shipped/schema/templates/spec.md",
		"shipped/skill/SKILL.md",
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %v, want %v", got, want)
	}
	// The embedded copy must match what is on disk.
	for _, p := range want {
		b, err := Shipped.ReadFile(p)
		if err != nil || string(b) != readShipped(t, p) {
			t.Errorf("%s: embedded copy missing or different (%v)", p, err)
		}
	}
}

func TestShippedSchemaTemplatesExist(t *testing.T) {
	names := regexp.MustCompile(`(?m)^\s*template:\s*(\S+)\s*$`).FindAllStringSubmatch(readShipped(t, "shipped/schema/schema.yaml"), -1)
	if len(names) == 0 {
		t.Fatal("schema names no templates")
	}
	for _, m := range names {
		if _, err := os.Stat("shipped/schema/templates/" + m[1]); err != nil {
			t.Errorf("template %s: %v", m[1], err)
		}
	}
}

func TestShippedStarts(t *testing.T) {
	for path, prefix := range map[string]string{
		"shipped/skill/SKILL.md":               "---\nname: docket\n",
		"shipped/schema/templates/proposal.md": "---\n",
		"shipped/schema/templates/plan.md":     "# Implementation Plan\n",
		"shipped/config/operations.yaml":       "operations:\n",
	} {
		if !strings.HasPrefix(readShipped(t, path), prefix) {
			t.Errorf("%s does not start with %q", path, prefix)
		}
	}
}

func TestShippedNextStepsAreOneLine(t *testing.T) {
	for _, n := range []string{"commit", "reload", "doctor"} {
		s := readShipped(t, "shipped/next-steps/"+n+".md")
		if strings.TrimSpace(s) == "" || strings.Count(s, "\n") != 1 || !strings.HasSuffix(s, "\n") {
			t.Errorf("%s: want one non-empty line, got %q", n, s)
		}
	}
}
