package docket

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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
		"shipped/record/template.md",
		"shipped/schema/schema.yaml",
		"shipped/schema/templates/design.md",
		"shipped/schema/templates/plan.md",
		"shipped/schema/templates/proposal.md",
		"shipped/schema/templates/spec.md",
		"shipped/skill/SKILL.md",
		"shipped/workflow/workflow.md",
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
		"shipped/workflow/workflow.md":         "---\ntemplate: ",
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

// workflowTemplates maps each template version to the SHA-256 of the template as shipped at that version.
var workflowTemplates = map[int]string{
	1: "6642708242c599fe3930bce7873c8feca7234f3ea84e3c5a48f3870fa2453be6",
}

func TestWorkflowTemplatePinned(t *testing.T) {
	body := readShipped(t, "shipped/workflow/workflow.md")
	m := regexp.MustCompile(`(?m)\A---\n(?:.*\n)*?template: (\d+)\n`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("shipped/workflow/workflow.md has no template version")
	}
	version, _ := strconv.Atoi(m[1])
	largest := 0
	for v := range workflowTemplates {
		if v > largest {
			largest = v
		}
	}
	if version < 1 || version != largest {
		t.Fatalf("template version = %d, want the largest pinned version %d", version, largest)
	}
	sum := sha256.Sum256([]byte(body))
	if hex.EncodeToString(sum[:]) != workflowTemplates[version] {
		t.Errorf("shipped/workflow/workflow.md changed: raise its template version and add the new hash to workflowTemplates")
	}
}

func TestSkillSectionsInTemplate(t *testing.T) {
	skill := strings.Join(strings.Fields(readShipped(t, "shipped/skill/SKILL.md")), " ")
	var names []string
	for _, m := range regexp.MustCompile(`"([^"]+)"(?: and "([^"]+)")? in `+"`docket/workflow.md`").FindAllStringSubmatch(skill, -1) {
		for _, n := range m[1:] {
			if n != "" {
				names = append(names, n)
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("shipped/skill/SKILL.md names no section of docket/workflow.md")
	}
	headings := map[string]bool{}
	for _, l := range strings.Split(readShipped(t, "shipped/workflow/workflow.md"), "\n") {
		if h, ok := strings.CutPrefix(strings.TrimRight(l, "\r"), "### "); ok {
			headings[strings.TrimSpace(h)] = true
		}
	}
	for _, n := range names {
		if !headings[n] {
			t.Errorf("skill names section %q, which is not a ### heading of the shipped template", n)
		}
	}
	t.Logf("collected %d names: %q", len(names), names)
}
