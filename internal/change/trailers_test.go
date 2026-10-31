package change

import (
	"testing"

	"github.com/wake/docket/internal/gitx"
)

func TestNamedTasks(t *testing.T) {
	p := Plan{
		Tasks: []Task{
			{Number: 1, Name: "one"},
			{Number: 2, Name: "first two"},
			{Number: 2, Name: "second two"},
			{Number: 3, Name: "three"},
		},
		Cancelled: []Cancelled{{Number: 4, Name: "four", Reason: "dropped", OK: true}},
	}
	other := gitx.Commit{SHA: "a1", Change: []string{"other"}, Task: []string{"1"}}
	two := gitx.Commit{SHA: "b2", Change: []string{"x"}, Task: []string{"2", "2", "9"}}
	zero := gitx.Commit{SHA: "c3", Change: []string{"x"}, Task: []string{"01"}}
	cancelled := gitx.Commit{SHA: "d4", Change: []string{"x"}, Task: []string{"4"}}
	one := gitx.Commit{SHA: "e5", Change: []string{"x"}, Task: []string{"1"}}

	named, unknown, anyTrailer := NamedTasks(p, "x", []gitx.Commit{other, two, zero, cancelled, one})
	if len(named) != 2 || named[0].Task.Name != "one" || named[1].Task.Name != "first two" {
		t.Fatalf("named = %+v", named)
	}
	if len(named[0].Commits) != 1 || named[0].Commits[0].SHA != "e5" {
		t.Errorf("task 1 commits = %+v", named[0].Commits)
	}
	if len(named[1].Commits) != 1 || named[1].Commits[0].SHA != "b2" {
		t.Errorf("task 2 commits = %+v", named[1].Commits)
	}
	want := []UnknownTrailer{{two, "9"}, {zero, "01"}, {cancelled, "4"}}
	if len(unknown) != len(want) {
		t.Fatalf("unknown = %+v", unknown)
	}
	for i, u := range unknown {
		if u.Value != want[i].Value || u.Commit.SHA != want[i].Commit.SHA {
			t.Errorf("unknown[%d] = %s on %s, want %s on %s", i, u.Value, u.Commit.SHA, want[i].Value, want[i].Commit.SHA)
		}
	}
	if !anyTrailer {
		t.Error("anyTrailer = false")
	}

	named, unknown, anyTrailer = NamedTasks(p, "none", []gitx.Commit{other, two})
	if len(named) != 0 || len(unknown) != 0 || anyTrailer {
		t.Errorf("no commit for slug: %v %v %v", named, unknown, anyTrailer)
	}
}

func TestMatchTasksTakesEveryCommit(t *testing.T) {
	p := Plan{Tasks: []Task{{Number: 1, Name: "one"}}}
	named, unknown, anyTrailer := MatchTasks(p, []gitx.Commit{{SHA: "a1", Task: []string{"1"}}})
	if len(named) != 1 || named[0].Task.Name != "one" || len(unknown) != 0 || !anyTrailer {
		t.Fatalf("named = %+v, unknown = %+v, anyTrailer = %v", named, unknown, anyTrailer)
	}
}
