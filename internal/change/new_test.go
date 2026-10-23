package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestValidateSlug(t *testing.T) {
	cases := []struct {
		slug string
		ok   bool
		want string
	}{
		{"add-x", true, ""},
		{"Add-X", false, ""},
		{"add_x", false, ""},
		{"-x", false, ""},
		{"x-", false, ""},
		{"a--b", false, ""},
		{strings.Repeat("a", 200), true, ""},
		{strings.Repeat("a", 201), false, "200"},
		{"2026-10-24-add-x", false, "archiv"},
		{"archive", false, "reserved"},
	}
	for _, c := range cases {
		err := ValidateSlug(c.slug)
		if (err == nil) != c.ok {
			t.Errorf("%q: err = %v, want ok=%v", c.slug, err, c.ok)
		}
		if err != nil && c.want != "" && !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %q lacks %q", c.slug, err, c.want)
		}
	}
}

func readMeta(t *testing.T, path string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	return m
}

func TestAppendMetadataRoundTrip(t *testing.T) {
	for _, src := range []string{"issue 12", `issue #12: "x"`, "需求 R1#3"} {
		p := filepath.Join(t.TempDir(), ".openspec.yaml")
		if err := os.WriteFile(p, []byte("schema: docket\ncreated: 2026-01-02\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := AppendMetadata(p, "feature", src, "abc123"); err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		m := readMeta(t, p)
		if m["x-source"] != src || m["x-kind"] != "feature" || m["x-base"] != "abc123" || m["schema"] != "docket" || m["created"] != "2026-01-02" {
			t.Errorf("%q: %v", src, m)
		}
		if src == "issue 12" {
			b, _ := os.ReadFile(p)
			if !strings.Contains(string(b), "x-source: issue 12\n") {
				t.Errorf("not plain:\n%s", b)
			}
		}
	}
}

func TestAppendMetadataNoTrailingNewline(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".openspec.yaml")
	if err := os.WriteFile(p, []byte("schema: docket\ncreated: 2026-01-02"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AppendMetadata(p, "fix", "x", "abc"); err != nil {
		t.Fatal(err)
	}
	m := readMeta(t, p)
	if m["created"] != "2026-01-02" || m["x-kind"] != "fix" {
		t.Errorf("%v", m)
	}
}

func TestAppendMetadataBadYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), ".openspec.yaml")
	if err := os.WriteFile(p, []byte("schema: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AppendMetadata(p, "fix", "x", "abc"); err == nil {
		t.Fatal("want error")
	}
}
