package cli

import (
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestConflictsCopyOpenSpecFails(t *testing.T) { // Copy OpenSpec cannot parse
	e := newCfEnv(t)
	put(t, e.r.Dir, spec("add-x", "x"))
	put(t, e.worktree(t, "other"), spec("add-y", "y"))
	e.branch("br", spec("add-z", "z"))
	e.answer(t, "add-x", modifyCancel)
	e.answer(t, "add-z", modifyCancel)
	testutil.OpenSpecAnswer(t, e.answ, 1, `{"status":[{"message":"bad delta in add-y"}]}`,
		"show", "add-y", "--type", "change", "--json", "--deltas-only")
	code, m, raw, _ := e.do(t)
	if code != 3 || m.Result == nil || len(m.Result.Overlaps) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	if len(m.Unreadable) != 1 || !strings.HasSuffix(m.Unreadable[0].Source, ":openspec/changes/add-y") ||
		!strings.Contains(m.Unreadable[0].Reason, "bad delta in add-y") {
		t.Errorf("%+v", m.Unreadable)
	}
}
