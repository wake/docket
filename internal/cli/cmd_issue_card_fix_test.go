package cli

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestIssueCardBrokenFront(t *testing.T) {
	r := recordRepo(t)
	const bad = "docket/issues/2026-10-29-login.md"
	r.Write(bad, "---\nexternal: \""+cardURL+"\"\ntitle: \"T\n---\n")
	code, m, raw := doIssue(t, r, r.Dir, "login-loop", "--title", "X", "--source", "x", "--external", cardURL, "--external-id", "#92")
	if code != 3 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != bad {
		t.Errorf("code=%d %s", code, raw)
	}
	gone(t, r, "docket/issues/2026-10-30-login-loop.md")
}

func TestIssueCardBlankFlags(t *testing.T) {
	base := []string{"login-loop", "--title", "X", "--source", "x"}
	for name, extra := range map[string][]string{
		"external":     {"--external", ""},
		"both":         {"--external", "", "--external-id", ""},
		"external-id":  {"--external-id", ""},
		"id with addr": {"--external", cardURL, "--external-id", ""},
		"addr with id": {"--external", "", "--external-id", "#92"},
	} {
		t.Run(name, func(t *testing.T) {
			r := recordRepo(t)
			code, _, raw := doIssue(t, r, r.Dir, append(append([]string{}, base...), extra...)...)
			if code != 2 {
				t.Errorf("code=%d %s", code, raw)
			}
			gone(t, r, "docket")
		})
	}
}

func TestIssueCardConcurrent(t *testing.T) {
	r := recordRepo(t)
	var wg sync.WaitGroup
	codes := make([]int, 2)
	taken := make([]bool, 2)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, m, _ := doIssue(t, r, r.Dir, "login-loop", "--title", "X", "--source", "x", "--external", cardURL, "--external-id", "#92")
			codes[i], taken[i] = c, hasRuleIn(m.Findings, "external-taken")
		}()
	}
	wg.Wait()
	entries, _ := os.ReadDir(filepath.Join(r.Dir, "docket/issues"))
	if codes[0]+codes[1] != 1 || taken[0] == taken[1] || len(entries) != 1 {
		t.Fatalf("codes %v taken %v files %d", codes, taken, len(entries))
	}
}

func TestIssueCardNoDocketFolder(t *testing.T) {
	r := recordRepo(t)
	code, _, raw := doIssue(t, r, r.Dir, "x", "--title", "X", "--source", "x", "--external", cardURL, "--external-id", "#1")
	if code != 0 {
		t.Fatalf("code=%d %s", code, raw)
	}
	entries, _ := os.ReadDir(filepath.Join(r.Dir, "docket/issues"))
	if len(entries) != 1 {
		t.Errorf("files %d", len(entries))
	}
}

func TestIssueCardTakenWritesNothing(t *testing.T) {
	r := recordRepo(t)
	r.Write("docket/issues/2026-10-29-login.md", "---\ntitle: \"T\"\nexternal: \""+cardURL+"\"\n---\n")
	list := func() []string {
		var names []string
		filepath.WalkDir(r.Dir, func(p string, d os.DirEntry, _ error) error {
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}
			names = append(names, p)
			return nil
		})
		return names
	}
	before := list()
	code, m, raw := doIssue(t, r, r.Dir, "x", "--title", "X", "--source", "x", "--external", cardURL, "--external-id", "#1")
	after := list()
	if code != 1 || !hasRuleIn(m.Findings, "external-taken") || len(before) != len(after) {
		t.Errorf("code=%d %s before %v after %v", code, raw, before, after)
	}
}
