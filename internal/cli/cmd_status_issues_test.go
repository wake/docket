package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	stIssue     = "docket/issues/2026-11-01-a.md"
	stIssueText = "---\ntitle: A thing\nsource: x\n---\n\n## Current\n\nx\n"
)

func statusIssues(t *testing.T, so string) []any {
	t.Helper()
	res, _ := decode(t, so)["result"].(map[string]any)
	if res == nil {
		t.Fatalf("result is null\n%s", so)
	}
	is, ok := res["issues"].([]any)
	if !ok {
		t.Fatalf("issues is not an array\n%s", so)
	}
	return is
}

func TestStatusListsPendingIssue(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("issue", map[string]string{stIssue: stIssueText})
	code, so, se := runStatus(t, r.Dir, true)
	is := statusIssues(t, so)
	if code != 0 || len(is) != 1 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	i := is[0].(map[string]any)
	if i["slug"] != "2026-11-01-a" || i["state"] != "pending" || i["title"] != "A thing" || len(i["handlers"].([]any)) != 0 {
		t.Fatalf("issue = %v", i)
	}
	_, hs, _ := runStatus(t, r.Dir, false)
	if !strings.Contains(hs, "issue 2026-11-01-a pending \"A thing\"\n") {
		t.Fatalf("human:\n%s", hs)
	}
}

func TestStatusIssueInProgress(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("issue", map[string]string{stIssue: stIssueText})
	w := r.AddWorktree(filepath.Join(t.TempDir(), "wt-b"), "wt-b")
	w.Write("docket/records/2026-11-02-fix-a.md", "---\ntitle: R\nkind: fix\nsource: "+stIssue+"\n---\n\n## What\n\nx\n")
	_, so, _ := runStatus(t, r.Dir, true)
	is := statusIssues(t, so)
	if len(is) != 1 || is[0].(map[string]any)["state"] != "in-progress" {
		t.Fatalf("issues = %v", is)
	}
	h := is[0].(map[string]any)["handlers"].([]any)[0].(map[string]any)
	if h["kind"] != "record" || h["slug"] != "2026-11-02-fix-a" || h["merged"] != false {
		t.Fatalf("handler = %v", h)
	}
	_, hs, _ := runStatus(t, r.Dir, false)
	if !strings.Contains(hs, "issue 2026-11-01-a in-progress \"A thing\" (record 2026-11-02-fix-a)\n") {
		t.Fatalf("human:\n%s", hs)
	}
}

func TestStatusHidesHandledDroppedAndMovedIssues(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("issues", map[string]string{
		stIssue:                              stIssueText,
		"docket/records/2026-11-02-fix-a.md": "---\ntitle: R\nkind: fix\nsource: " + stIssue + "\n---\n\n## What\n\nx\n",
		"docket/issues/2026-11-01-b.md":      "---\ntitle: B\nsource: x\ndropped: true\n---\n\nx\n",
		"docket/issues/2026-11-01-c.md":      "---\ntitle: C\nsource: x\nidea: docket/ideas/c.md\n---\n\nx\n",
	})
	_, so, _ := runStatus(t, r.Dir, true)
	if is := statusIssues(t, so); len(is) != 0 {
		t.Fatalf("issues = %v", is)
	}
}

func TestStatusSlugsListNoIssues(t *testing.T) {
	r := cleanRepo(t)
	r.Commit("issue", map[string]string{stIssue: stIssueText})
	_, so, _ := runStatus(t, r.Dir, true, "add-x")
	if is := statusIssues(t, so); len(is) != 0 {
		t.Fatalf("issues = %v", is)
	}
}
