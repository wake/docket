package docket

import (
	"io/fs"
	"strings"
	"testing"
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

func TestAgentsSection(t *testing.T) {
	s := AgentsSection()
	if !strings.HasPrefix(s, "## Change workflow (superpowers × OpenSpec)") {
		t.Errorf("unexpected start: %q", s)
	}
	if !strings.Contains(s, "dkt new") {
		t.Error("section lacks dkt new")
	}
	if strings.Contains(s, "## Development") {
		t.Error("section runs past its end")
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

func TestAgentsSectionReviewAtBothEnds(t *testing.T) {
	s := AgentsSection()
	for _, want := range []string{"**Review:** judgment", "dkt conflicts", "one pull request"} {
		if !strings.Contains(s, want) {
			t.Errorf("section lacks %q", want)
		}
	}
}
