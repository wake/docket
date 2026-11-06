package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/wake/docket/internal/testutil"
)

type domainsOut struct {
	Findings   []Finding
	Unreadable []Unreadable
	Result     *domainsResult
}

// dmRun runs dkt domains in dir with no OpenSpec on the machine; the text
// output is returned when json is false.
func dmRun(t *testing.T, r *testutil.Repo, dir string, json_ bool, args ...string) (int, domainsOut, string) {
	t.Helper()
	env := append(slices.Clone(r.Env), "DOCKET_OPENSPEC="+filepath.Join(t.TempDir(), "nope"))
	a := append([]string{"domains"}, args...)
	if json_ {
		a = append(a, "--json")
	}
	var out, errb bytes.Buffer
	code := Run(Options{Args: a, Dir: dir, Env: env, Stdout: &out, Stderr: &errb})
	var m domainsOut
	if json_ && out.Len() > 0 {
		if err := json.Unmarshal(out.Bytes(), &m); err != nil {
			t.Fatalf("%q: %v", out.String(), err)
		}
	}
	return code, m, out.String() + errb.String()
}

func dmSpec(name string) string { return "# " + name + " Specification\n" }

func dmRepo(t *testing.T, files map[string]string) *testutil.Repo {
	t.Helper()
	r := testutil.NewRepo(t)
	if files == nil {
		files = map[string]string{}
	}
	files["README.md"] = "x"
	r.Commit("base", files)
	return r
}

func capPaths(cs []domainCapability) []string {
	out := []string{}
	for _, c := range cs {
		out = append(out, c.Path)
	}
	return out
}

func TestDomainsTree(t *testing.T) {
	r := dmRepo(t, map[string]string{
		"docket/domains.md":                      "# Areas\n- 付款\n  - `billing/invoice`\n- 帳號\n",
		"openspec/specs/billing/invoice/spec.md": dmSpec("Invoice"),
		"openspec/specs/other/spec.md":           dmSpec("Other"),
	})
	r.Write("docket/domains.md", "# Areas\n- 付款\n  - `billing/invoice`\n- 帳號\n- 新增\n")
	before := r.Git("status", "--porcelain")
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 0 || m.Result == nil {
		t.Fatalf("%d %s", code, raw)
	}
	inv := domainCapability{Path: "billing/invoice", Title: "Invoice", Main: true, Changes: []string{}}
	want := &domainsResult{
		Found: true,
		Nodes: []domainNode{
			{Name: "付款", Capabilities: []domainCapability{inv}, Children: []domainNode{}},
			{Name: "帳號", Capabilities: []domainCapability{}, Children: []domainNode{}},
			{Name: "新增", Capabilities: []domainCapability{}, Children: []domainNode{}},
		},
		Unclassified: []domainCapability{{Path: "other", Title: "Other", Main: true, Changes: []string{}}},
	}
	if !reflect.DeepEqual(m.Result, want) {
		t.Errorf("%+v", m.Result)
	}
	var env struct{ Result json.RawMessage }
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatal(err)
	}
	if s := string(env.Result); !strings.Contains(s, `"children":[]`) || !strings.Contains(s, `"changes":[]`) || strings.Contains(s, "null") {
		t.Errorf("%s", s)
	}
	if after := r.Git("status", "--porcelain"); after != before {
		t.Errorf("status changed: %q -> %q", before, after)
	}
}

func TestDomainsNoTree(t *testing.T) {
	r := dmRepo(t, map[string]string{"openspec/specs/refund/spec.md": dmSpec("Refund")})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 0 || m.Result == nil || m.Result.Found || len(m.Result.Nodes) != 0 || !slices.Equal(capPaths(m.Result.Unclassified), []string{"refund"}) {
		t.Errorf("%d %s", code, raw)
	}
	if !strings.Contains(raw, `"nodes":[]`) {
		t.Errorf("%s", raw)
	}
}

func TestDomainsFormat(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/domains.md": "- A\nUsers\n"})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 1 || m.Result != nil || len(m.Findings) != 1 || m.Findings[0].Rule != "domain-format" || m.Findings[0].File != "docket/domains.md" || m.Findings[0].Line != 2 {
		t.Errorf("%d %s", code, raw)
	}
}

func TestDomainsDuplicate(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/domains.md": "- A\n- A\n"})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 1 || m.Result != nil || len(m.Findings) != 1 || m.Findings[0].Rule != "domain-duplicate" || m.Findings[0].Line != 2 {
		t.Errorf("%d %s", code, raw)
	}
}

func TestDomainsUnknown(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/domains.md": "- A\n  - `refund`\n"})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 1 || len(m.Findings) != 1 || m.Findings[0].Rule != "domain-unknown" || m.Findings[0].Line != 2 || m.Result == nil {
		t.Fatalf("%d %s", code, raw)
	}
	want := []domainCapability{{Path: "refund", Changes: []string{}}}
	if !reflect.DeepEqual(m.Result.Nodes[0].Capabilities, want) {
		t.Errorf("%+v", m.Result.Nodes[0].Capabilities)
	}
}

func TestDomainsLinkedTree(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/other.md": "x"})
	if err := os.Symlink("other.md", filepath.Join(r.Dir, "docket/domains.md")); err != nil {
		t.Fatal(err)
	}
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 3 || m.Result != nil || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "docket/domains.md" {
		t.Errorf("%d %s", code, raw)
	}
	if !strings.Contains(raw, `"result":null`) {
		t.Errorf("%s", raw)
	}
}

func skipRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
}

func TestDomainsUnreadableSpec(t *testing.T) {
	skipRoot(t)
	r := dmRepo(t, map[string]string{
		"docket/domains.md":             "- A\n  - `gone`\n",
		"openspec/specs/refund/spec.md": dmSpec("Refund"),
	})
	p := filepath.Join(r.Dir, "openspec/specs/refund/spec.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 3 || len(m.Findings) != 0 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/specs/refund/spec.md" || m.Result == nil || len(m.Result.Nodes) != 1 {
		t.Errorf("%d %s", code, raw)
	}
	if want := []domainCapability{{Path: "refund", Main: true, Changes: []string{}}}; !reflect.DeepEqual(m.Result.Unclassified, want) {
		t.Errorf("%+v", m.Result.Unclassified)
	}
}

func TestDomainsUnreadableProposal(t *testing.T) {
	skipRoot(t)
	r := dmRepo(t, map[string]string{
		"docket/domains.md":                           "- A\n  - `gone`\n",
		"openspec/changes/add-x/specs/refund/spec.md": "x",
		"openspec/changes/add-x/proposal.md":          "---\ntitle: X\nspecs:\n  refund: Refund\n---\n",
	})
	p := filepath.Join(r.Dir, "openspec/changes/add-x/proposal.md")
	if err := os.Chmod(p, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(p, 0o644) })
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 3 || len(m.Findings) != 0 || len(m.Unreadable) != 1 || m.Unreadable[0].Source != "openspec/changes/add-x/proposal.md" || m.Result == nil {
		t.Fatalf("%d %s", code, raw)
	}
	want := []domainCapability{{Path: "refund", Changes: []string{"add-x"}}}
	if !reflect.DeepEqual(m.Result.Unclassified, want) {
		t.Errorf("%+v", m.Result.Unclassified)
	}
}

func TestDomainsDocketIsAFile(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket": "x"})
	code, m, raw := dmRun(t, r, r.Dir, true)
	if code != 0 || m.Result == nil || m.Result.Found {
		t.Errorf("%d %s", code, raw)
	}
}

func TestDomainsText(t *testing.T) {
	r := dmRepo(t, map[string]string{
		"docket/domains.md":              "- 付款\n  - `refund`\n",
		"openspec/specs/refund/spec.md":  dmSpec("退款"),
		"openspec/specs/invoice/spec.md": dmSpec("invoice"),
	})
	code, _, out := dmRun(t, r, r.Dir, false)
	if want := "付款\n  refund: 退款\nUnclassified:\n  invoice\n"; code != 0 || out != want {
		t.Errorf("%d %q", code, out)
	}
}

func TestDomainsTextNoTree(t *testing.T) {
	r := dmRepo(t, nil)
	code, _, out := dmRun(t, r, r.Dir, false)
	if code != 0 || !strings.Contains(out, "docket/domains.md does not exist") {
		t.Errorf("%d %q", code, out)
	}
}

func TestDomainsArgument(t *testing.T) {
	r := dmRepo(t, nil)
	if code, _, raw := dmRun(t, r, r.Dir, true, "x"); code != 2 {
		t.Errorf("%d %s", code, raw)
	}
}

func TestDomainsOutsideRepository(t *testing.T) { // Domains outside a repository
	var out bytes.Buffer
	code := Run(Options{Args: []string{"domains", "--json"}, Dir: t.TempDir(), Env: testutil.Env(t), Stdout: &out, Stderr: &bytes.Buffer{}})
	var m domainsOut
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("%q: %v", out.String(), err)
	}
	if code != 4 || len(m.Findings) != 1 || m.Findings[0].Rule != "no-repository" {
		t.Errorf("code=%d out=%q", code, out.String())
	}
}

func TestDomainsFromSubdirectory(t *testing.T) {
	r := dmRepo(t, map[string]string{"docket/domains.md": "- A\n", "sub/f": "x"})
	code, m, raw := dmRun(t, r, filepath.Join(r.Dir, "sub"), true)
	if code != 0 || m.Result == nil || !m.Result.Found || len(m.Result.Nodes) != 1 {
		t.Errorf("%d %s", code, raw)
	}
}
