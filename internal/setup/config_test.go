package setup

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

const testOps = "operations:\n  archive:\n    guidance:\n      - \"x\"\n"

func plan(t *testing.T, in string) (string, ConfigState, error) {
	t.Helper()
	out, st, err := planConfig([]byte(in), []byte(testOps))
	return string(out), st, err
}

func TestPlanConfigOpenSpec170Fixture(t *testing.T) {
	b, err := os.ReadFile("../../.superpowers/sdd/add-init-and-inspect/openspec-1.7.0-config.yaml")
	if err != nil {
		t.Skip("fixture not present")
	}
	in := string(b)
	out, st, err := plan(t, in)
	if err != nil {
		t.Fatal(err)
	}
	want := "schema: docket" + strings.TrimPrefix(in, "schema: spec-driven")
	want = strings.TrimRight(want, "\n") + "\n\n" + testOps
	if out != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
	if st != (ConfigState{Schema: "default", Guidance: "missing"}) {
		t.Errorf("state %+v", st)
	}
}

func TestPlanConfigTable(t *testing.T) {
	tests := []struct {
		name, in, out string
		st            ConfigState
	}{
		{"quoted default", "schema: \"spec-driven\"\n", "schema: \"docket\"\n\n" + testOps, ConfigState{"default", "missing"}},
		{"single quoted", "schema: 'spec-driven'\n", "schema: 'docket'\n\n" + testOps, ConfigState{"default", "missing"}},
		{"comment kept", "schema: spec-driven # default\n", "schema: docket # default\n\n" + testOps, ConfigState{"default", "missing"}},
		{"crlf kept", "schema: spec-driven\r\n", "schema: docket\r\n\n" + testOps, ConfigState{"default", "missing"}},
		{"no schema key", "# c\n", "schema: docket\n# c\n\n" + testOps, ConfigState{"default", "missing"}},
		{"already docket", "schema: docket\n\n" + testOps, "schema: docket\n\n" + testOps, ConfigState{"docket", "same"}},
		{"other kept", "schema: team-flow\n", "schema: team-flow\n\n" + testOps, ConfigState{"other", "missing"}},
		{"no final newline", "schema: docket", "schema: docket\n\n" + testOps, ConfigState{"docket", "missing"}},
		{"ends with blank line", "schema: docket\n\n", "schema: docket\n\n" + testOps, ConfigState{"docket", "missing"}},
		{"edited guidance kept", "schema: docket\noperations:\n  apply: {}\n", "schema: docket\noperations:\n  apply: {}\n", ConfigState{"docket", "differs"}},
		{"operations indented is not top-level", "schema: docket\nx:\n  operations:\n    a: 1\n", "schema: docket\nx:\n  operations:\n    a: 1\n\n" + testOps, ConfigState{"docket", "missing"}},
		{"comment ends section, tail kept", "schema: docket\n" + testOps + "\n# tail\nfoo: 1\n", "schema: docket\n" + testOps + "\n# tail\nfoo: 1\n", ConfigState{"docket", "same"}},
		{"section equals shipped without final newline", "schema: docket\n" + strings.TrimSuffix(testOps, "\n"), "schema: docket\n" + strings.TrimSuffix(testOps, "\n"), ConfigState{"docket", "same"}},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			out, st, err := plan(t, c.in)
			if err != nil {
				t.Fatal(err)
			}
			if out != c.out || st != c.st {
				t.Errorf("out %q state %+v\nwant %q %+v", out, st, c.out, c.st)
			}
		})
	}
}

func TestPlanConfigEarlierSectionReplaced(t *testing.T) {
	old := "operations:\n  archive:\n    guidance:\n      - old\n"
	sum := sha256.Sum256([]byte(old))
	saved := earlier[opsKey]
	earlier[opsKey] = []string{hex.EncodeToString(sum[:])}
	defer func() { earlier[opsKey] = saved }()
	in := "schema: docket\n" + old + "\n# tail\n"
	out, st, err := plan(t, in)
	if err != nil {
		t.Fatal(err)
	}
	if want := "schema: docket\n" + testOps + "\n# tail\n"; out != want {
		t.Errorf("got %q want %q", out, want)
	}
	if st.Guidance != "outdated" {
		t.Errorf("state %+v", st)
	}
}

func TestPlanConfigUnreadable(t *testing.T) {
	for name, in := range map[string]string{
		"schema without value":     "schema:\n  spec-driven\n",
		"schema comment only":      "schema: # x\n",
		"two schema":               "schema: a\nschema: b\n",
		"two operations":           "operations:\n  a: 1\noperations:\n  b: 2\n",
		"operations with value":    "operations: {}\n",
		"flow mapping":             "{schema: a}\n",
		"flow sequence":            "[a]\n",
		"quoted key":               "\"schema\": a\n",
		"single quoted key":        "'schema': a\n",
		"schema space colon":       "schema : a\n",
		"operations space colon":   "operations : a\n",
		"complex key":              "? schema\n",
		"document start":           "---\nschema: a\n",
		"unterminated quote":       "schema: \"a\n",
		"schema followed by words": "schema: a b\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := plan(t, in); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestDecideConfig(t *testing.T) {
	root := t.TempDir()
	ds, _ := decideT(t, root, false)
	d := find(ds, "openspec/config.yaml")
	if d == nil || d.Kind != KindConfig || d.Status != StatusMissing || d.Action != ActionCreate ||
		d.Config == nil || *d.Config != (ConfigState{"default", "missing"}) {
		t.Fatalf("%+v", d)
	}
	ci, si, ki := -1, -1, -1
	for i, x := range ds {
		switch x.Kind {
		case KindConfig:
			ci = i
		case KindSchema:
			si = i
		case KindSkill:
			if ki < 0 {
				ki = i
			}
		}
	}
	if !(si < ci && ci < ki) {
		t.Errorf("order: schema %d config %d skill %d", si, ci, ki)
	}
	if find(ds, "openspec/schemas/docket/schema.yaml").Config != nil {
		t.Error("Config set on a schema")
	}
	if err := Write(root, ds); err != nil {
		t.Fatal(err)
	}
	if got := get(t, root, "openspec/config.yaml"); !strings.HasPrefix(got, "schema: docket\n\noperations:") {
		t.Errorf("%q", got)
	}
	ds, _ = decideT(t, root, false)
	if d := find(ds, "openspec/config.yaml"); d.Status != StatusSame || d.Action != ActionNone || *d.Config != (ConfigState{"docket", "same"}) {
		t.Errorf("%+v", d)
	}
}

func TestDecideConfigStatuses(t *testing.T) {
	ops := string(mustShippedOps(t))
	cases := []struct {
		name, in string
		force    bool
		status   Status
		action   Action
	}{
		{"default schema and section", "schema: spec-driven\n\n" + ops, false, StatusOutdated, ActionUpdate},
		{"other schema, section missing", "schema: team\n", false, StatusDiffers, ActionUpdate},
		{"other schema, section there", "schema: team\n\n" + ops, true, StatusDiffers, ActionKeep},
		{"docket, differing guidance", "schema: docket\noperations:\n  a: 1\n", true, StatusDiffers, ActionKeep},
		{"default, differing guidance", "operations:\n  a: 1\n", false, StatusDiffers, ActionUpdate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			put(t, root, "openspec/config.yaml", c.in)
			ds, ps := decideT(t, root, c.force)
			if len(ps) != 0 {
				t.Fatal(ps)
			}
			d := find(ds, "openspec/config.yaml")
			if d.Status != c.status || d.Action != c.action {
				t.Errorf("%s/%s", d.Status, d.Action)
			}
			before := get(t, root, "openspec/config.yaml")
			if err := Write(root, ds); err != nil {
				t.Fatal(err)
			}
			after := get(t, root, "openspec/config.yaml")
			if (c.action == ActionKeep) != (before == after) {
				t.Errorf("after %q", after)
			}
			if strings.Contains(c.in, "team") && !strings.Contains(after, "schema: team") {
				t.Errorf("other schema changed: %q", after)
			}
			if strings.Contains(c.in, "a: 1") && !strings.Contains(after, "a: 1") {
				t.Errorf("guidance changed: %q", after)
			}
		})
	}
}

func TestDecideConfigUnreadableIsAProblem(t *testing.T) {
	root := t.TempDir()
	put(t, root, "openspec/config.yaml", "schema:\n  x\n")
	ds, ps := decideT(t, root, false)
	if find(ds, "openspec/config.yaml") != nil {
		t.Error("decision for an unreadable file")
	}
	if len(ps) != 1 || ps[0].Path != "openspec/config.yaml" || ps[0].Kind != ProblemUnreadable || ps[0].Err == nil {
		t.Errorf("%+v", ps)
	}
}

func mustShippedOps(t *testing.T) []byte {
	t.Helper()
	b, err := readShippedOps(shipped(t))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
