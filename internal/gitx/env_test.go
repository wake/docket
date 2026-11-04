package gitx

import (
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func TestEnvIgnoresHostGitVariables(t *testing.T) {
	t.Setenv("GIT_DIR", t.TempDir())
	t.Setenv("GIT_WORK_TREE", t.TempDir())
	t.Setenv("GIT_INDEX_FILE", "/nonexistent/index")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "commit.gpgsign")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	t.Setenv("GIT_CONFIG_PARAMETERS", "'core.bare=true'")

	env := testutil.Env(t)
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_CONFIG_COUNT",
			"GIT_CONFIG_KEY_0", "GIT_CONFIG_VALUE_0", "GIT_CONFIG_PARAMETERS":
			t.Fatalf("Env inherited %s", k)
		}
	}
	r := testutil.NewRepo(t)
	if sha := r.Commit("one", map[string]string{"a": "1"}); len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}
}
