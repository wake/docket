package specdoc

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const delta = `## ADDED Requirements

### Requirement: Join when full

#### Scenario: Full slot

#### Scenario: Slot frees up

## MODIFIED Requirements

### Requirement: Cancel

#### Scenario: Cancel early

## REMOVED Requirements

### Requirement: Walk-in mark

**Reason**: Walk-ins moved to paper.

## RENAMED Requirements

- FROM: ` + "`### Requirement: Party limit`" + `
- TO: ` + "`### Requirement: Group size limit`" + `
`

var want = Delta{
	Added:    []Requirement{{Name: "Join when full", Scenarios: []string{"Full slot", "Slot frees up"}}},
	Modified: []Requirement{{Name: "Cancel", Scenarios: []string{"Cancel early"}}},
	Removed:  []Removed{{Name: "Walk-in mark", Reason: "Walk-ins moved to paper."}},
	Renamed:  []Renamed{{From: "Party limit", To: "Group size limit"}},
}

func TestParseDelta(t *testing.T) {
	if got := ParseDelta([]byte(delta)); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseDeltaEdges(t *testing.T) {
	crlf := "\xef\xbb\xbf" + strings.ReplaceAll(delta, "\n", "\r\n")
	if got := ParseDelta([]byte(crlf)); !reflect.DeepEqual(got, want) {
		t.Fatalf("crlf: got %+v", got)
	}
	in := "#### Scenario: Orphan\n\n### Requirement: No op\n\n## ADDED Requirements\n\n#### Scenario: Early\n\n" +
		"```\n### Requirement: Example\n```\n\n<!--\n### Requirement: Example\n-->\n\n" +
		"###   Requirement:   Join when full ###  \n\n#### Scenario: One\n"
	got := ParseDelta([]byte(in))
	exp := Delta{Added: []Requirement{{Name: "Join when full", Scenarios: []string{"One"}}},
		Modified: []Requirement{}, Removed: []Removed{}, Renamed: []Renamed{}}
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseDeltaJSONNeverNull(t *testing.T) {
	d := ParseDelta([]byte("## REMOVED Requirements\n\n### Requirement: X\n\n**Reason**: r\n"))
	b, _ := json.Marshal(d)
	if got := string(b); got != `{"added":[],"modified":[],"removed":[{"name":"X","reason":"r"}],"renamed":[]}` {
		t.Fatalf("got %s", got)
	}
	d = ParseDelta([]byte("## ADDED Requirements\n\n### Requirement: Y\n"))
	b, _ = json.Marshal(d.Added)
	if got := string(b); got != `[{"name":"Y","scenarios":[]}]` {
		t.Fatalf("got %s", got)
	}
}

func TestParseDeltaRenameEdges(t *testing.T) {
	r := func(n string) string { return "`### Requirement: " + n + "`" }
	in := "## RENAMED Requirements\n\n```\n- FROM: " + r("F0") + "\n- TO: " + r("T0") + "\n```\n\n" +
		"- TO: " + r("Lone") + "\n" +
		"- FROM: " + r("A") + "\n- TO: " + r("B") + "\n- TO: " + r("Extra") + "\n"
	got := ParseDelta([]byte(in)).Renamed
	if !reflect.DeepEqual(got, []Renamed{{From: "A", To: "B"}}) {
		t.Fatalf("got %+v", got)
	}
}

func TestParseDeltaReason(t *testing.T) {
	in := "## REMOVED Requirements\n\n### Requirement: X\n\n```\n**Reason**: fenced\n```\n\n  **Reason**: first\n**Reason**: second\n"
	got := ParseDelta([]byte(in)).Removed
	if !reflect.DeepEqual(got, []Removed{{Name: "X", Reason: "first"}}) {
		t.Fatalf("got %+v", got)
	}
}

func TestRequirements(t *testing.T) {
	in := "### Requirement: A\n\n```\n### Requirement: C\n```\n\n### Requirement: B\n"
	if got := Requirements([]byte(in)); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("got %v", got)
	}
	if got := Requirements(nil); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
