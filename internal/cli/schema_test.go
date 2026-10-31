package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

func schemaContext(t *testing.T, root, data string) *Context {
	t.Helper()
	return &Context{Repo: &Repo{Root: root}, Env: []string{"HOME=" + t.TempDir(), "XDG_DATA_HOME=" + data}}
}

func TestFindSchemaEditedProjectCopy(t *testing.T) {
	root := t.TempDir()
	testutil.InstallSchema(t, root)
	f := filepath.Join(root, "openspec/schemas/docket/schema.yaml")
	if err := os.WriteFile(f, []byte("name: docket\n# mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, lines, err := findSchema(schemaContext(t, root, t.TempDir()))
	if err != nil || st == nil || st.Matches || len(lines) != 1 ||
		!strings.Contains(lines[0], "differs from every shipped schema") {
		t.Fatalf("%v %v %v", st, lines, err)
	}
}

func TestFindSchemaNothing(t *testing.T) {
	st, lines, err := findSchema(schemaContext(t, t.TempDir(), t.TempDir()))
	if st != nil || lines != nil || err != nil {
		t.Fatalf("%v %v %v", st, lines, err)
	}
}
