package cli

import (
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestLintMissingNamesChange(t *testing.T) {
	r := lintRepo(t, "spec-driven")
	cmd, _ := testutil.FakeOpenSpec(t, "1.14.0")
	env := append(append([]string{}, r.Env...), "DOCKET_OPENSPEC="+cmd)
	_, out, _ := runIn(t, r.Dir, env, "lint", "add-z", "--json")
	if !strings.Contains(out, `"change":"add-z"`) {
		t.Fatalf("%s", out)
	}
}
