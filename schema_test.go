package docket

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestSchemaHasSchemaFile(t *testing.T) {
	b, err := fs.ReadFile(Schema(), "schema.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "name: docket") {
		t.Errorf("schema.yaml lacks %q", "name: docket")
	}
}

func TestSchemaHasTemplates(t *testing.T) {
	for _, n := range []string{"proposal", "spec", "design", "plan"} {
		if _, err := fs.ReadFile(Schema(), "templates/"+n+".md"); err != nil {
			t.Errorf("templates/%s.md: %v", n, err)
		}
	}
	if _, err := fs.ReadFile(Schema(), "templates/tasks.md"); err == nil {
		t.Error("templates/tasks.md must not exist")
	}
}

func TestSkillFrontmatter(t *testing.T) {
	b, err := fs.ReadFile(Skill(), "SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, "---\n") {
		t.Fatal("SKILL.md does not start with frontmatter")
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		t.Fatal("frontmatter is not closed")
	}
	var fm struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Name != "docket" {
		t.Errorf("name = %q", fm.Name)
	}
	if fm.Description == "" || len(fm.Description) > 1024 {
		t.Errorf("description length %d", len(fm.Description))
	}
}

func TestBlock(t *testing.T) {
	b := Block()
	if !strings.HasPrefix(b, "<!-- docket:start -->\n") {
		t.Errorf("bad start: %q", b)
	}
	if !strings.HasSuffix(b, "\n<!-- docket:end -->") {
		t.Errorf("bad end: %q", b)
	}
	for _, m := range []string{"<!-- docket:start -->", "<!-- docket:end -->"} {
		if n := strings.Count(b, m); n != 1 {
			t.Errorf("%s appears %d times", m, n)
		}
	}
}

func TestPlanTemplateReviewLine(t *testing.T) {
	b, err := fs.ReadFile(Schema(), "templates/plan.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "**Review:** <checks|judgment>") {
		t.Error("plan template lacks the **Review:** line")
	}
}

func TestInstalledSkillMatchesShipped(t *testing.T) {
	err := fs.WalkDir(Skill(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want, err := fs.ReadFile(Skill(), p)
		if err != nil {
			return err
		}
		got, err := os.ReadFile(filepath.Join(".claude", "skills", "docket", filepath.FromSlash(p)))
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return nil
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from the shipped copy", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
