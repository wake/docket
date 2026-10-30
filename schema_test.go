package docket

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/wake/docket/internal/setup"
)

func TestTemplateHeadings(t *testing.T) {
	lines := strings.Split(Template(), "\n")
	for _, h := range []string{"## Full review", "## Light review", "## Check", "## Direct merge", "## Reviews"} {
		n := 0
		for _, l := range lines {
			if l == h {
				n++
			}
		}
		if n != 1 {
			t.Errorf("template has %q on %d whole lines, want 1", h, n)
		}
	}
}

func TestTemplateNamesNoProduct(t *testing.T) {
	low := strings.ToLower(Template())
	for _, w := range []string{"claude", "opus", "sonnet", "haiku", "fable", "gpt", "codex", "gemini", "openai", "anthropic", "superpowers", "copilot"} {
		if strings.Contains(low, w) {
			t.Errorf("template names %q", w)
		}
	}
}

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
	for _, sk := range Skills() {
		t.Run(sk.Name, func(t *testing.T) { checkSkillFrontmatter(t, sk) })
	}
}

func checkSkillFrontmatter(t *testing.T, sk setup.Skill) {
	b, err := fs.ReadFile(sk.Files, "SKILL.md")
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
		Name                   string `yaml:"name"`
		Description            string `yaml:"description"`
		DisableModelInvocation *bool  `yaml:"disable-model-invocation"`
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Name != sk.Name {
		t.Errorf("name = %q", fm.Name)
	}
	if fm.Description == "" || len(fm.Description) > 1024 {
		t.Errorf("description length %d", len(fm.Description))
	}
	if sk.Name == "docket-doctor" {
		if fm.DisableModelInvocation == nil || !*fm.DisableModelInvocation {
			t.Error("disable-model-invocation must be true")
		}
	} else if fm.DisableModelInvocation != nil {
		t.Error("disable-model-invocation must be absent")
	}
}

func TestSkillNames(t *testing.T) {
	var got []string
	for _, sk := range Skills() {
		got = append(got, sk.Name)
	}
	if want := []string{"docket", "docket-setup", "docket-doctor"}; !slices.Equal(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func TestDoctorHiddenFromCodex(t *testing.T) {
	for _, sk := range Skills() {
		b, err := fs.ReadFile(sk.Files, "agents/openai.yaml")
		if sk.Name != "docket-doctor" {
			if err == nil {
				t.Errorf("%s has agents/openai.yaml", sk.Name)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		var y struct {
			Policy struct {
				AllowImplicitInvocation *bool `yaml:"allow_implicit_invocation"`
			} `yaml:"policy"`
		}
		if err := yaml.Unmarshal(b, &y); err != nil {
			t.Fatal(err)
		}
		if a := y.Policy.AllowImplicitInvocation; a == nil || *a {
			t.Error("policy.allow_implicit_invocation must be false")
		}
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
	for _, sk := range Skills() {
		dir := filepath.Join(".claude", "skills", sk.Name)
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return err
			}
			rel, err := filepath.Rel(dir, p)
			if err != nil {
				return err
			}
			if _, err := fs.Stat(sk.Files, filepath.ToSlash(rel)); err != nil {
				t.Errorf("%s is installed but not shipped", p)
			}
			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", dir, err)
		}
		err = fs.WalkDir(sk.Files, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			want, err := fs.ReadFile(sk.Files, p)
			if err != nil {
				return err
			}
			got, err := os.ReadFile(filepath.Join(".claude", "skills", sk.Name, filepath.FromSlash(p)))
			if err != nil {
				t.Errorf("%s/%s: %v", sk.Name, p, err)
				return nil
			}
			if string(got) != string(want) {
				t.Errorf("%s/%s differs from the shipped copy", sk.Name, p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
