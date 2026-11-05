package gitx

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestIgnored(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{".gitignore": ".claude/*\n!.claude/skills/\nopenspec/\nAGENTS.md\n"})
	r.Write("AGENTS.md", "x\n")
	r.Git("add", "-f", "AGENTS.md")
	r.Git("commit", "-q", "-m", "agents")
	g := Git{Dir: r.Dir, Env: r.Env}

	got, err := g.Ignored([]string{".claude/skills/docket/SKILL.md", ".claude/settings.local.json", "openspec/config.yaml", "AGENTS.md"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Ignore{
		{Path: ".claude/settings.local.json", Source: ".gitignore", Line: 1, Pattern: ".claude/*"},
		{Path: "openspec/config.yaml", Source: ".gitignore", Line: 3, Pattern: "openspec/"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	got, err = g.Ignored([]string{"README.md"})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := g.Ignored([]string{"../outside"}); err == nil {
		t.Fatal("a path outside the repository gave no error")
	}
}

func TestIgnoredTrackedAndMissing(t *testing.T) {
	r := testutil.NewRepo(t)
	r.Commit("init", map[string]string{".gitignore": "*.log\n"})
	r.Write("kept.log", "x\n")
	r.Git("add", "-f", "kept.log")
	r.Git("commit", "-q", "-m", "kept")
	g := Git{Dir: r.Dir, Env: r.Env}

	got, err := g.Ignored([]string{"kept.log", "not/yet.log"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Ignore{{Path: "not/yet.log", Source: ".gitignore", Line: 1, Pattern: "*.log"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
