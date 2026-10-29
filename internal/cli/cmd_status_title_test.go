package cli

import (
	"strings"
	"testing"
)

func TestStatusLineTitle(t *testing.T) {
	r := cleanRepo(t)
	w := statusChange(t, r, "add-x")
	w.Write("openspec/changes/add-x/docket.yaml", "title: Booking waitlist\nkind: fix\nsource: issue 7\nbase: main\n")
	code, so, se := runStatus(t, r.Dir, false)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, so, se)
	}
	if want := "add-x  \"Booking waitlist\"  ready  0/2 tasks"; !strings.Contains(so, want) {
		t.Errorf("missing %q in:\n%s", want, so)
	}
}
