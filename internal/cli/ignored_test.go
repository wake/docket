package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/setup"
	"github.com/wake/docket/internal/testutil"
)

// installIgnored lists the install-ignored findings as file -> message.
func installIgnored(t *testing.T, so string) map[string]string {
	t.Helper()
	got := map[string]string{}
	fnds, _ := decode(t, so)["findings"].([]any)
	for _, f := range fnds {
		if m := f.(map[string]any); m["rule"] == "install-ignored" {
			got[m["file"].(string)] = m["message"].(string)
		}
	}
	return got
}

func TestInitIgnoredSkill(t *testing.T) {
	r := committedRepo(t)
	const ign = "# local\n\n.claude/\n"
	r.Commit("ignore", map[string]string{".gitignore": ign})
	code, so, se := initIn(t, r, nil)
	if code != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	got := installIgnored(t, so)
	n := 0
	err := fs.WalkDir(os.DirFS(r.Dir), ".claude/skills", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		if got[p] != "ignored by .gitignore:3: .claude/" {
			t.Errorf("%s: %q", p, got[p])
		}
		return nil
	})
	if err != nil || n == 0 || len(got) != n {
		t.Fatalf("walk: %v, %d files, %d findings\n%s", err, n, len(got), so)
	}
	if b, _ := os.ReadFile(filepath.Join(r.Dir, ".gitignore")); string(b) != ign {
		t.Errorf(".gitignore changed: %q", b)
	}
}

func TestInitTrackedNotReported(t *testing.T) {
	r := committedRepo(t)
	r.Commit("ignore", map[string]string{".gitignore": "AGENTS.md\n"})
	r.Write("AGENTS.md", "mine\n")
	r.Git("add", "-f", "AGENTS.md")
	r.Git("commit", "-q", "-m", "agents")
	_, so, se := initIn(t, r, nil)
	if got := installIgnored(t, so); len(got) != 0 {
		t.Errorf("reported %v\n%s", got, se)
	}
}

func TestInitNegatedRuleNotReported(t *testing.T) {
	r := committedRepo(t)
	r.Commit("ignore", map[string]string{".gitignore": ".claude/*\n!.claude/skills/\n"})
	_, so, se := initIn(t, r, nil)
	if got := installIgnored(t, so); len(got) != 0 {
		t.Errorf("reported %v\n%s", got, se)
	}
}

func TestInspectIgnoredFile(t *testing.T) {
	r := committedRepo(t)
	const excl = "# local\n\n\n\n\n\nopenspec/\n"
	p := filepath.Join(r.Dir, ".git/info/exclude")
	if err := os.WriteFile(p, []byte(excl), 0o644); err != nil {
		t.Fatal(err)
	}
	env, _ := inspectEnv(t)
	_, so, _ := runInspect(t, r.Dir, env, "--json")
	if got := installIgnored(t, so)["openspec/config.yaml"]; got != "ignored by .git/info/exclude:7: openspec/" {
		t.Errorf("message %q\n%s", got, so)
	}
	if b, _ := os.ReadFile(p); string(b) != excl {
		t.Error("exclude file changed")
	}
}

func TestIgnoredFindingsGitFails(t *testing.T) {
	c := &Context{Git: gitx.Git{Dir: t.TempDir(), Env: testutil.Env(t)}}
	fnds, un := ignoredFindings(c, []setup.File{
		{Path: "AGENTS.md"}, {Path: ".claude/skills/docket/SKILL.md"}, {Path: "/abs/schema.yaml"},
	})
	if len(fnds) != 0 || len(un) != 2 || un[0].Source != "AGENTS.md" || un[1].Source != ".claude/skills/docket/SKILL.md" || un[0].Reason == "" {
		t.Fatalf("findings %v, unreadable %v", fnds, un)
	}
}
