package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const oldOps = "operations:\n  archive:\n    guidance:\n      - old\n"

func injectOld(t *testing.T) {
	t.Helper()
	sum := sha256.Sum256([]byte(oldOps))
	saved := Earlier[opsKey]
	Earlier[opsKey] = []string{hex.EncodeToString(sum[:])}
	t.Cleanup(func() { Earlier[opsKey] = saved })
}

func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

func TestPlanConfigCRLFSection(t *testing.T) {
	in := crlf("schema: docket\n\n" + testOps)
	out, st, err := plan(t, in)
	if err != nil || out != in || st != (ConfigState{"docket", "same"}) {
		t.Errorf("%q %+v %v", out, st, err)
	}
	injectOld(t)
	out, st, err = plan(t, crlf("schema: docket\n"+oldOps))
	if err != nil || st.Guidance != "outdated" || out != crlf("schema: docket\n")+testOps {
		t.Errorf("%q %+v %v", out, st, err)
	}
}

func TestPlanConfigMoreUnreadable(t *testing.T) {
	for name, in := range map[string]string{
		"bom team":    "\xef\xbb\xbfschema: team-flow\n",
		"bom default": "\xef\xbb\xbfschema: spec-driven\n",
		"directive":   "%YAML 1.2\nschema: a\n",
		"doc end":     "schema: a\n...\n",
		"doc end cmt": "schema: a\n... # x\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := plan(t, in); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestPlanConfigSchemaAfterOutdatedSection(t *testing.T) {
	injectOld(t)
	out, st, err := plan(t, "# top\n"+oldOps+"\nschema: spec-driven # d\nz: 1\n")
	if err != nil || st != (ConfigState{"default", "outdated"}) {
		t.Fatalf("%+v %v", st, err)
	}
	if want := "# top\n" + testOps + "\nschema: docket # d\nz: 1\n"; out != want {
		t.Errorf("%q", out)
	}
}

func TestDecideEmptyConfigFile(t *testing.T) {
	root := t.TempDir()
	put(t, root, "openspec/config.yaml", "")
	ds, _ := decideT(t, root, false)
	d := find(ds, "openspec/config.yaml")
	if d.Status != StatusOutdated || d.Action != ActionUpdate {
		t.Fatalf("%s/%s", d.Status, d.Action)
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if got := get(t, root, "openspec/config.yaml"); !strings.HasPrefix(got, "schema: docket\n\noperations:") {
		t.Errorf("%q", got)
	}
}
