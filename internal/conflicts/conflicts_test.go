package conflicts

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wake/docket/internal/delta"
)

func req(name string) string {
	return "### Requirement: " + name + "\nThe system SHALL work.\n\n#### Scenario: s\n- **WHEN** a\n- **THEN** b\n\n"
}

func section(kind string, names ...string) string {
	var b strings.Builder
	b.WriteString("## " + kind + " Requirements\n\n")
	for _, n := range names {
		if kind == "REMOVED" {
			b.WriteString("### Requirement: " + n + "\n\n")
		} else {
			b.WriteString(req(n))
		}
	}
	return b.String()
}

func rename(from, to string) string {
	return "## RENAMED Requirements\n\n- FROM: `### Requirement: " + from + "`\n- TO: `### Requirement: " + to + "`\n\n"
}

func chg(slug string, caps map[string]string) Change {
	d := map[string]delta.Delta{}
	for k, v := range caps {
		d[k] = delta.Parse([]byte(v))
	}
	return Change{Slug: slug, Deltas: d}
}

func one(slug, text string) Change { return chg(slug, map[string]string{"booking": text}) }

func ops(o Overlap, slug string) []string {
	for _, e := range o.Changes {
		if e.Slug == slug {
			return e.Operations
		}
	}
	return nil
}

func slugs(o Overlap) []string {
	r := []string{}
	for _, e := range o.Changes {
		r = append(r, e.Slug)
	}
	return r
}

func TestBothModifyOneRequirement(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("MODIFIED", "Cancel booking")),
		one("add-y", section("MODIFIED", "Cancel booking")),
	})
	if len(got) != 1 || got[0].Capability != "booking" || got[0].Requirement != "Cancel booking" {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(slugs(got[0]), []string{"add-x", "add-y"}) ||
		!reflect.DeepEqual(ops(got[0], "add-x"), []string{Modified}) ||
		!reflect.DeepEqual(ops(got[0], "add-y"), []string{Modified}) {
		t.Fatalf("entries %+v", got[0].Changes)
	}
}

func TestRenameAgainstModify(t *testing.T) {
	got := Find([]Change{
		one("add-x", rename("Cancel booking", "Cancel a booking")),
		one("add-y", section("MODIFIED", "Cancel booking")),
	})
	if len(got) != 1 || got[0].Requirement != "Cancel booking" {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(ops(got[0], "add-x"), []string{RenamedFrom}) ||
		!reflect.DeepEqual(ops(got[0], "add-y"), []string{Modified}) {
		t.Fatalf("entries %+v", got[0].Changes)
	}
}

func TestSameCapabilityDifferentRequirements(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("MODIFIED", "Cancel booking")),
		one("add-y", section("ADDED", "Refund points")),
	})
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestBothAddTheSameName(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("ADDED", "Waitlist cap")),
		one("add-y", section("ADDED", "Waitlist cap")),
	})
	if len(got) != 1 || !reflect.DeepEqual(ops(got[0], "add-x"), []string{Added}) {
		t.Fatalf("got %+v", got)
	}
}

func TestRemoveAgainstModify(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("REMOVED", "Cancel booking")),
		one("add-y", section("MODIFIED", "Cancel booking")),
	})
	if len(got) != 1 || !reflect.DeepEqual(ops(got[0], "add-x"), []string{Removed}) {
		t.Fatalf("got %+v", got)
	}
}

func TestOperationOrder(t *testing.T) {
	text := rename("W", "X") + rename("X", "Z") + section("REMOVED", "X") + section("MODIFIED", "X") + section("ADDED", "X")
	got := Find([]Change{one("add-x", text), one("add-y", section("MODIFIED", "X"))})
	var o *Overlap
	for i := range got {
		if got[i].Requirement == "X" {
			o = &got[i]
		}
	}
	if o == nil {
		t.Fatalf("got %+v", got)
	}
	want := []string{Added, Modified, Removed, RenamedFrom, RenamedTo}
	if !reflect.DeepEqual(ops(*o, "add-x"), want) {
		t.Fatalf("ops %v", ops(*o, "add-x"))
	}
}

func TestNoOverlaps(t *testing.T) {
	got := Find([]Change{
		one("a", section("MODIFIED", "One")),
		one("b", section("MODIFIED", "Two")),
		one("c", section("ADDED", "Three")),
	})
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestRenameThenModifyInOneChange(t *testing.T) {
	got := Find([]Change{
		one("add-x", rename("A", "B")+section("MODIFIED", "B")),
		one("add-y", section("MODIFIED", "B")),
	})
	if len(got) != 1 || got[0].Requirement != "B" {
		t.Fatalf("got %+v", got)
	}
	if !reflect.DeepEqual(ops(got[0], "add-x"), []string{Modified, RenamedTo}) {
		t.Fatalf("ops %v", ops(got[0], "add-x"))
	}
}

func TestNamesMatchAsOpenSpecDoes(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("MODIFIED", "Cancel booking  ")),
		one("add-y", section("MODIFIED", "Cancel booking")),
	})
	if len(got) != 1 {
		t.Fatalf("trailing space: %+v", got)
	}
	got = Find([]Change{
		one("add-x", section("MODIFIED", "cancel booking")),
		one("add-y", section("MODIFIED", "Cancel booking")),
	})
	if len(got) != 0 {
		t.Fatalf("case: %+v", got)
	}
}

func TestNestedCapabilities(t *testing.T) {
	got := Find([]Change{
		chg("add-x", map[string]string{"a": section("MODIFIED", "R")}),
		chg("add-y", map[string]string{"a/b": section("MODIFIED", "R")}),
	})
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestThreeChangesSorted(t *testing.T) {
	got := Find([]Change{
		chg("c", map[string]string{"zeta": section("MODIFIED", "R"), "alpha": section("MODIFIED", "S")}),
		chg("a", map[string]string{"zeta": section("MODIFIED", "R"), "alpha": section("MODIFIED", "Q")}),
		chg("b", map[string]string{"zeta": section("MODIFIED", "R"), "alpha": section("MODIFIED", "Q")}),
		chg("d", map[string]string{"alpha": section("MODIFIED", "S")}),
	})
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Capability != "alpha" || got[0].Requirement != "Q" ||
		got[1].Capability != "alpha" || got[1].Requirement != "S" ||
		got[2].Capability != "zeta" {
		t.Fatalf("order %+v", got)
	}
	if !reflect.DeepEqual(slugs(got[2]), []string{"a", "b", "c"}) {
		t.Fatalf("slugs %v", slugs(got[2]))
	}
}

func TestDuplicateSlugCountsOnce(t *testing.T) {
	got := Find([]Change{
		one("add-x", section("MODIFIED", "R")),
		one("add-x", section("MODIFIED", "R")),
	})
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestInvolving(t *testing.T) {
	all := Find([]Change{
		chg("a", map[string]string{"c1": section("MODIFIED", "R"), "c2": section("MODIFIED", "S")}),
		chg("b", map[string]string{"c1": section("MODIFIED", "R")}),
		chg("c", map[string]string{"c2": section("MODIFIED", "S")}),
	})
	if got := Involving(all, "b"); len(got) != 1 || got[0].Capability != "c1" {
		t.Fatalf("got %+v", got)
	}
	if got := Involving(all, "a"); len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got := Involving(all, "nobody"); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
