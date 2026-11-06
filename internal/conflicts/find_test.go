package conflicts

import (
	"reflect"
	"testing"
)

func p(slug string, sources []string, touches ...Touch) Parsed {
	return Parsed{Copy: Copy{Slug: slug, Sources: sources}, Touches: touches}
}

func tc(capability, req, op string) Touch { return Touch{capability, req, op} }

func TestFind(t *testing.T) {
	main := []string{"main"}
	tests := []struct {
		name  string
		input []Parsed
		want  []Overlap
	}{
		{"both modify", []Parsed{
			p("add-x", main, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-y", main, tc("booking", "Cancel booking", "MODIFIED")),
		}, []Overlap{{"booking", "Cancel booking", []Entry{
			{"add-x", []string{"MODIFIED"}, main}, {"add-y", []string{"MODIFIED"}, main}}}}},
		{"rename against modify", []Parsed{
			p("add-x", main, tc("booking", "Cancel booking", "RENAMED-FROM"), tc("booking", "Cancel a booking", "RENAMED-TO")),
			p("add-y", main, tc("booking", "Cancel booking", "MODIFIED")),
		}, []Overlap{{"booking", "Cancel booking", []Entry{
			{"add-x", []string{"RENAMED-FROM"}, main}, {"add-y", []string{"MODIFIED"}, main}}}}},
		{"different requirements", []Parsed{
			p("add-x", main, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-y", main, tc("booking", "Refund points", "ADDED")),
		}, []Overlap{}},
		{"both add", []Parsed{
			p("add-x", main, tc("booking", "Waitlist cap", "ADDED")),
			p("add-y", main, tc("booking", "Waitlist cap", "ADDED")),
		}, []Overlap{{"booking", "Waitlist cap", []Entry{
			{"add-x", []string{"ADDED"}, main}, {"add-y", []string{"ADDED"}, main}}}}},
		{"case and spaces", []Parsed{
			p("add-x", main, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-y", main, tc("booking", "Cancel  Booking", "MODIFIED")),
		}, []Overlap{{"booking", "Cancel booking", []Entry{
			{"add-x", []string{"MODIFIED"}, main}, {"add-y", []string{"MODIFIED"}, main}}}}},
		{"copies that differ", []Parsed{
			p("add-x", []string{"branch"}, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-x", []string{"worktree"}, tc("booking", "Refund points", "MODIFIED")),
			p("add-y", main, tc("booking", "Refund points", "MODIFIED")),
		}, []Overlap{{"booking", "Refund points", []Entry{
			{"add-x", []string{"MODIFIED"}, []string{"worktree"}}, {"add-y", []string{"MODIFIED"}, main}}}}},
		{"own copies", []Parsed{
			p("add-x", []string{"branch"}, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-x", []string{"worktree"}, tc("booking", "Cancel booking", "MODIFIED")),
		}, []Overlap{}},
		{"sources listed", []Parsed{
			p("add-x", main, tc("booking", "Cancel booking", "MODIFIED")),
			p("add-y", []string{"worktree B", "refs/remotes/origin/add-y"}, tc("booking", "Cancel booking", "MODIFIED")),
		}, []Overlap{{"booking", "Cancel booking", []Entry{
			{"add-x", []string{"MODIFIED"}, main},
			{"add-y", []string{"MODIFIED"}, []string{"worktree B", "refs/remotes/origin/add-y"}}}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Find(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Find = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFindOrder(t *testing.T) {
	m := []string{"main"}
	got := Find([]Parsed{
		p("add-z", m, tc("booking", "Cancel booking", "MODIFIED"), tc("accounts", "Login", "MODIFIED")),
		p("add-y", m, tc("booking", "cancel BOOKING", "MODIFIED"), tc("accounts", "Login", "MODIFIED")),
		p("add-x", m, tc("booking", "Cancel booking", "MODIFIED")),
	})
	if len(got) != 2 || got[0].Capability != "accounts" || got[1].Capability != "booking" {
		t.Fatalf("order = %+v", got)
	}
	if got[1].Requirement != "Cancel booking" {
		t.Errorf("requirement = %q", got[1].Requirement)
	}
	var slugs []string
	for _, e := range got[1].Changes {
		slugs = append(slugs, e.Slug)
	}
	if !reflect.DeepEqual(slugs, []string{"add-x", "add-y", "add-z"}) {
		t.Errorf("entries = %v", slugs)
	}
}

func TestFindMergesOperations(t *testing.T) {
	m := []string{"main"}
	got := Find([]Parsed{
		p("add-x", []string{"a"}, tc("c", "R", "MODIFIED"), tc("c", "R", "RENAMED-TO")),
		p("add-x", []string{"b"}, tc("c", "R", "REMOVED"), tc("c", "R", "MODIFIED")),
		p("add-y", m, tc("c", "R", "ADDED")),
	})
	want := []string{"MODIFIED", "REMOVED", "RENAMED-TO"}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Changes[0].Operations, want) {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(got[0].Changes[0].Sources, []string{"a", "b"}) {
		t.Errorf("sources = %v", got[0].Changes[0].Sources)
	}
}

func TestInvolving(t *testing.T) {
	a := Overlap{Capability: "a", Changes: []Entry{{Slug: "add-x"}, {Slug: "add-y"}}}
	b := Overlap{Capability: "b", Changes: []Entry{{Slug: "add-y"}, {Slug: "add-z"}}}
	if got := Involving([]Overlap{a, b}, "add-x"); !reflect.DeepEqual(got, []Overlap{a}) {
		t.Errorf("got %+v", got)
	}
	if got := Involving([]Overlap{a, b}, "nope"); got == nil || len(got) != 0 {
		t.Errorf("got %#v", got)
	}
}
