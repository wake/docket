package snapshot

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
)

const dir = "openspec/changes/add-x/"

func stageIn(files map[string]string, commits []gitx.Commit, wtCommit string, def bool) WorksInput {
	f := Files{}
	for k, v := range files {
		f[k] = []byte(v)
	}
	return WorksInput{
		Changes: []scan.Use{{Slug: "add-x", Path: "openspec/changes/add-x", Source: wt}},
		Places:  map[string]Place{wt: {Kind: "worktree", Commit: wtCommit, Default: def}},
		Files:   map[string]Files{wt: f},
		Graph:   NewGraph(commits),
		Tip:     wtCommit,
	}
}

const planMd = "## Tasks\n\n### Task 1: One\n\n- [ ] **Step 1: a**\n- [x] **Step 2: b**\n\n### Task 2: Two\n\n- [x] **Step 1: a**\n- [ ] **Step 2: b**\n- [ ] **Step 3: c**\n"

func TestTasksFromReadFrom(t *testing.T) {
	in := stageIn(map[string]string{dir + "plan.md": planMd, dir + ".openspec.yaml": "schema: docket\n"},
		[]gitx.Commit{c("c2", "2", []string{"c1"}, "Change", "add-x/1"), c("c1", "1", nil)}, "c2", false)
	ws, _ := Works(in)
	ts := ws[0].Tasks
	if len(ts) != 2 || ts[0].State != "done" || !reflect.DeepEqual(ts[0].Commits, []string{"c2"}) || ts[1].State != "pending" || ts[1].Ticked != 1 || ts[1].Steps != 3 || ws[0].Progress != (Progress{1, 2}) {
		t.Fatalf("%+v %+v", ts, ws[0].Progress)
	}
	in.Places[wt] = Place{Kind: "worktree", Commit: "c1"}
	if ws, _ = Works(in); ws[0].Tasks[0].State != "pending" || len(ws[0].Tasks[0].Commits) != 0 {
		t.Fatalf("%+v", ws[0].Tasks[0])
	}
}

func TestStagesChange(t *testing.T) {
	files := map[string]string{dir + "proposal.md": "---\ntitle: X\n---\n", dir + "plan.md": planMd, dir + ".openspec.yaml": "schema: docket\n"}
	ws, _ := Works(stageIn(files, []gitx.Commit{c("c1", "1", nil, "Change", "add-x")}, "c1", false))
	if !reflect.DeepEqual(ws[0].Stages, []string{"proposal", "plan"}) || ws[0].Phase != "plan" {
		t.Fatalf("%v", ws[0].Stages)
	}
	ws, _ = Works(stageIn(files, []gitx.Commit{c("c1", "1", nil, "Change", "add-x/1")}, "c1", false))
	if !reflect.DeepEqual(ws[0].Stages, []string{"proposal", "plan", "dev"}) {
		t.Fatalf("%v", ws[0].Stages)
	}
	ws, _ = Works(stageIn(files, []gitx.Commit{c("c1", "1", nil, "Change", "add-x/a")}, "c1", false))
	if ws[0].Phase != "plan" {
		t.Fatalf("%v", ws[0].Stages)
	}
	in := stageIn(files, []gitx.Commit{c("c1", "1", nil, "Change", "add-x/1")}, "c1", true)
	in.Changes[0].Archived = true
	ws, _ = Works(in)
	if !reflect.DeepEqual(ws[0].Stages, []string{"proposal", "plan", "dev", "merged", "archived"}) || ws[0].Phase != "archived" {
		t.Fatalf("%v", ws[0].Stages)
	}
}

func TestStagesDoneNotMerged(t *testing.T) {
	files := map[string]string{dir + "plan.md": "### Task 1: One\n\n- [ ] **Step 1: a**\n", dir + ".openspec.yaml": "schema: docket\n"}
	ws, _ := Works(stageIn(files, []gitx.Commit{c("c1", "1", nil, "Change", "add-x/1")}, "c1", false))
	if ws[0].Phase != "dev" || ws[0].Progress != (Progress{1, 1}) || ws[0].Merged {
		t.Fatalf("%+v", ws[0])
	}
}

func TestStagesRecord(t *testing.T) {
	p := "docket/records/2026-11-06-r.md"
	mk := func(def bool, tr ...string) WorkEntry {
		in := WorksInput{
			Records: []scan.Use{{Slug: "2026-11-06-r", Path: p, Source: wt}},
			Places:  map[string]Place{wt: {Kind: "worktree", Commit: "c1", Default: def}},
			Files:   map[string]Files{wt: {p: []byte("---\ntitle: R\n---\n")}},
			Graph:   NewGraph([]gitx.Commit{c("c1", "1", nil, tr...)}), Tip: "c1",
		}
		ws, _ := Works(in)
		return ws[0]
	}
	if w := mk(false, "Record", "2026-11-06-r"); !reflect.DeepEqual(w.Stages, []string{"record", "dev"}) {
		t.Fatalf("%v", w.Stages)
	}
	if w := mk(true); !reflect.DeepEqual(w.Stages, []string{"record", "merged"}) || w.Merge != nil {
		t.Fatalf("%v", w.Stages)
	}
}

func TestMergeOfWork(t *testing.T) {
	files := map[string]string{dir + "proposal.md": "x", dir + ".openspec.yaml": "schema: docket\n"}
	cs := []gitx.Commit{c("m", "3", []string{"b"}), c("b", "2", []string{"a"}, "Change", "add-x"), c("a", "1", nil)}
	in := stageIn(files, cs, "m", true)
	in.Tags = []gitx.Tag{{Name: "v1", Commit: "m", Time: "T"}}
	w, _ := Works(in)
	if w[0].Merge == nil || w[0].Merge.Commit != "b" || w[0].Release == nil || w[0].Release.Tag != "v1" || w[0].LastCommit == nil || w[0].LastCommit.Commit != "b" {
		t.Fatalf("%+v", w[0])
	}
	w, _ = Works(stageIn(files, []gitx.Commit{c("m", "3", nil)}, "m", true))
	if !w[0].Merged || w[0].Merge != nil || w[0].Release != nil {
		t.Fatalf("%+v", w[0])
	}
}

func TestDocumentsAndDeltas(t *testing.T) {
	d := "## ADDED Requirements\n### Requirement: R\nbody\n#### Scenario: S\n- **WHEN** a\n"
	files := map[string]string{dir + "proposal.md": "x", dir + "design.md": "x", dir + "plan.md": "x", dir + "notes.md": "x",
		dir + "specs/booking/spec.md": d, dir + "specs/auth/spec.md": d}
	w, _ := Works(stageIn(files, nil, "", false))
	kinds := map[string]string{}
	for _, x := range w[0].Documents {
		kinds[x.Path] = x.Kind
		if x.Source != wt {
			t.Fatalf("%+v", x)
		}
	}
	want := map[string]string{dir + "proposal.md": "proposal", dir + "design.md": "design", dir + "plan.md": "plan", dir + "notes.md": "other", dir + "specs/booking/spec.md": "spec-delta", dir + "specs/auth/spec.md": "spec-delta"}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("%v", kinds)
	}
	ds := w[0].Deltas
	if len(ds) != 2 || ds[0].Capability != "auth" || ds[1].Capability != "booking" || ds[1].Path != dir+"specs/booking/spec.md" || len(ds[0].Added) != 1 || ds[0].Removed == nil {
		t.Fatalf("%+v", ds)
	}
}

const wt = "/w/app"

func in1() WorksInput {
	prop := func(title string) []byte { return []byte("---\ntitle: " + title + "\nkind: feature\nsource: s\n---\n") }
	return WorksInput{
		Changes: []scan.Use{
			{Slug: "add-x", Path: "openspec/changes/add-x", Source: wt},
			{Slug: "add-x", Path: "openspec/changes/add-x", Source: "refs/heads/b", Commit: "c1"},
			{Slug: "add-x", Path: "openspec/changes/add-x", Source: "refs/remotes/origin/b", Commit: "c1"},
		},
		Places: map[string]Place{
			wt:                      {Kind: "worktree", Branch: "b", Commit: "c1", Time: "t1"},
			"refs/heads/b":          {Kind: "branch", Branch: "b", Commit: "c1", Time: "t1"},
			"refs/remotes/origin/b": {Kind: "remote-branch", Branch: "origin/b", Commit: "c1", Time: "t1", Default: true},
		},
		Files: map[string]Files{
			wt: {
				"openspec/changes/add-x/proposal.md":    prop("Edited"),
				"openspec/changes/add-x/.openspec.yaml": []byte("schema: docket\ncreated: 2026-11-08\n"),
			},
			"refs/heads/b":          {"openspec/changes/add-x/proposal.md": prop("Old")},
			"refs/remotes/origin/b": {"openspec/changes/add-x/proposal.md": prop("Old")},
		},
	}
}

func TestWorksOneEntry(t *testing.T) {
	ws, bad := Works(in1())
	if len(ws) != 1 || len(bad) != 0 || len(ws[0].Locations) != 3 {
		t.Fatalf("%+v %v", ws, bad)
	}
	l := ws[0].Locations
	if l[0].Kind != "worktree" || l[1].Kind != "branch" || l[2].Kind != "remote-branch" || *l[2].Branch != "origin/b" || *l[0].Commit != "c1" || *l[0].Time != "t1" {
		t.Fatalf("%+v", l)
	}
}

func TestWorksCopyRead(t *testing.T) {
	in := in1()
	ws, _ := Works(in)
	if ws[0].ReadFrom != wt || *ws[0].Title != "Edited" || ws[0].Archived != nil || *ws[0].Created != "2026-11-08" {
		t.Fatalf("%+v", ws[0])
	}
	arch := "openspec/changes/archive/2026-11-08-add-x"
	in.Changes = append(in.Changes, scan.Use{Slug: "add-x", Archived: true, Path: arch, Source: "refs/remotes/origin/b", Commit: "c2"})
	in.Files["refs/remotes/origin/b"][arch+"/proposal.md"] = []byte("---\ntitle: Done\nkind: feature\nsource: s\n---\n")
	ws, _ = Works(in)
	if ws[0].ReadFrom != "refs/remotes/origin/b" || *ws[0].Archived != arch || *ws[0].Title != "Done" {
		t.Fatalf("%+v", ws[0])
	}
}

func TestWorksDetachedAndDefault(t *testing.T) {
	in := in1()
	in.Places[wt] = Place{Kind: "worktree", Commit: "c1"}
	ws, _ := Works(in)
	l := ws[0].Locations
	if l[0].Branch != nil || l[0].Default || l[1].Default || !l[2].Default {
		t.Fatalf("%+v", l)
	}
}

func TestWorksRecord(t *testing.T) {
	in := WorksInput{
		Records: []scan.Use{{Slug: "2026-11-06-fix-x", Path: "docket/records/2026-11-06-fix-x.md", Source: wt}},
		Files:   map[string]Files{wt: {"docket/records/2026-11-06-fix-x.md": []byte("---\ntitle: Fix X\nkind: fix\nsource: \"review of #40\"\n---\n\n## What\nx\n")}},
	}
	ws, bad := Works(in)
	w := ws[0]
	if len(bad) != 0 || w.Type != "record" || w.Schema != "docket" || *w.Created != "2026-11-06" || *w.Title != "Fix X" || *w.Kind != "fix" || *w.Source != "review of #40" || len(w.Deltas) != 0 || w.Deltas == nil {
		t.Fatalf("%q %q %q %q %v", *w.Title, *w.Kind, *w.Source, *w.Created, bad)
	}
}

func TestWorksUnreadableFrontMatter(t *testing.T) {
	in := in1()
	in.Files[wt]["openspec/changes/add-x/proposal.md"] = []byte("---\ntitle: X\n")
	ws, bad := Works(in)
	if ws[0].Title != nil || ws[0].Kind != nil || ws[0].Source != nil || len(bad) != 1 || bad[0].Source != wt+":openspec/changes/add-x/proposal.md" || bad[0].Reason == "" {
		t.Fatalf("%+v %v", ws[0], bad)
	}
}

func TestWorksNoFrontMatter(t *testing.T) {
	in := in1()
	in.Files[wt]["openspec/changes/add-x/proposal.md"] = []byte("# Just text\n")
	ws, bad := Works(in)
	if ws[0].Title != nil || ws[0].Kind != nil || ws[0].Source != nil || len(bad) != 0 {
		t.Fatalf("%+v %v", ws[0], bad)
	}
}

func TestWorksOtherSchema(t *testing.T) {
	in := in1()
	in.Files[wt]["openspec/changes/add-x/.openspec.yaml"] = []byte("schema: spec-driven\ncreated: 2026-11-08\n")
	ws, _ := Works(in)
	if ws[0].Schema != "spec-driven" || ws[0].Tasks != nil {
		t.Fatalf("%+v", ws[0])
	}
}

func TestWorksSorted(t *testing.T) {
	in := WorksInput{
		Changes: []scan.Use{{Slug: "b", Path: "openspec/changes/b", Source: wt}, {Slug: "a", Path: "openspec/changes/a", Source: wt}, {Slug: "c", Path: "openspec/changes/c", Source: wt}},
		Records: []scan.Use{{Slug: "2026-11-08-r", Path: "docket/records/2026-11-08-r.md", Source: wt}},
		Files: map[string]Files{wt: {
			"openspec/changes/a/.openspec.yaml": []byte("created: 2026-11-08\n"),
			"openspec/changes/b/.openspec.yaml": []byte("created: 2026-11-08\n"),
		}},
	}
	ws, _ := Works(in)
	var got []string
	for _, w := range ws {
		got = append(got, w.Slug)
	}
	if want := []string{"a", "b", "2026-11-08-r", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func keys(m map[string]any) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sorted(s ...string) []string { sort.Strings(s); return s }

func TestResultJSON(t *testing.T) {
	b, _ := json.Marshal(Result{})
	var m map[string]any
	json.Unmarshal(b, &m)
	if !reflect.DeepEqual(keys(m), sorted("default_branch", "works", "ideas", "issues", "order", "specs", "domains")) || m["default_branch"] != nil {
		t.Fatalf("%s", b)
	}
	for _, k := range []string{"works", "ideas", "issues", "order", "specs"} {
		if a, ok := m[k].([]any); !ok || len(a) != 0 {
			t.Fatalf("%s: %s", k, b)
		}
	}
	d := m["domains"].(map[string]any)
	if !reflect.DeepEqual(keys(d), sorted("found", "nodes", "unclassified")) || len(d["nodes"].([]any)) != 0 || len(d["unclassified"].([]any)) != 0 {
		t.Fatalf("%s", b)
	}

	ws, _ := Works(in1())
	other := ws[0]
	other.Schema = "spec-driven"
	other.Tasks = nil
	docket := ws[0]
	docket.Tasks = []Task{{Number: 1}}
	docket.Deltas = []DeltaEntry{{}}
	b, _ = json.Marshal(Result{Works: []WorkEntry{other, docket}, Ideas: []IdeaEntry{{}}, Issues: []IssueEntry{{}}, Specs: []Spec{{}}, Domains: Domains{Nodes: []Node{{}}}})
	var r struct {
		Works   []map[string]any
		Ideas   []map[string]any
		Issues  []map[string]any
		Specs   []map[string]any
		Domains struct{ Nodes []map[string]any }
	}
	json.Unmarshal(b, &r)
	w := r.Works[0]
	wantW := sorted("type", "slug", "title", "kind", "source", "schema", "created", "proposed", "position", "read_from", "archived", "stages", "phase", "progress", "merged", "merge", "release", "last_commit", "locations", "tasks", "documents", "deltas")
	if !reflect.DeepEqual(keys(w), wantW) || w["tasks"] != nil || w["merge"] != nil || w["position"] != nil || w["archived"] != nil {
		t.Fatalf("%s", b)
	}
	if r.Works[1]["tasks"] == nil || len(r.Works[1]["deltas"].([]any)) != 1 {
		t.Fatalf("%s", b)
	}
	for _, k := range []string{"stages", "documents", "deltas"} {
		if _, ok := w[k].([]any); !ok {
			t.Fatalf("%s: %s", k, b)
		}
	}
	loc := w["locations"].([]any)[0].(map[string]any)
	if !reflect.DeepEqual(keys(loc), sorted("source", "kind", "branch", "default", "commit", "time", "path", "archived")) {
		t.Fatalf("%s", b)
	}
	if !reflect.DeepEqual(keys(r.Ideas[0]), sorted("path", "title", "needs", "dropped", "date", "state", "waiting_on", "position", "taken_by")) ||
		!reflect.DeepEqual(keys(r.Issues[0]), sorted("slug", "path", "title", "source", "date", "state", "position", "handlers", "locations")) ||
		!reflect.DeepEqual(keys(r.Specs[0]), sorted("capability", "title", "path", "requirements")) ||
		!reflect.DeepEqual(keys(r.Domains.Nodes[0]), sorted("name", "capabilities", "children")) ||
		r.Issues[0]["handlers"] == nil || r.Ideas[0]["needs"] == nil || r.Ideas[0]["title"] != nil {
		t.Fatalf("%s", b)
	}
}

func TestEmptyPhase(t *testing.T) {
	ws, _ := Works(stageIn(map[string]string{dir + ".openspec.yaml": "schema: docket\n"}, nil, "", false))
	if len(ws[0].Stages) != 0 || ws[0].Stages == nil || ws[0].Phase != "" {
		t.Fatalf("%+v", ws[0])
	}
}

func TestDevFromOtherLocation(t *testing.T) {
	in := stageIn(map[string]string{dir + "plan.md": planMd, dir + ".openspec.yaml": "schema: docket\n"},
		[]gitx.Commit{c("c2", "2", []string{"c0"}, "Change", "add-x/1"), c("c1", "1", []string{"c0"}), c("c0", "0", nil)}, "c1", false)
	in.Changes = append(in.Changes, scan.Use{Slug: "add-x", Path: "openspec/changes/add-x", Source: "refs/heads/b", Commit: "c2"})
	in.Places["refs/heads/b"] = Place{Kind: "branch", Branch: "b", Commit: "c2"}
	in.Files["refs/heads/b"] = Files{}
	ws, _ := Works(in)
	if ws[0].ReadFrom != wt || ws[0].Tasks[0].State != "pending" || len(ws[0].Tasks[0].Commits) != 0 || ws[0].Phase != "dev" {
		t.Fatalf("%+v", ws[0])
	}
}
