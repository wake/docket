package specdoc

import (
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

func TestRequirements(t *testing.T) {
	in := "### Requirement: A\n\n```\n### Requirement: C\n```\n\n### Requirement: B\n"
	if got := Requirements([]byte(in)); !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Fatalf("got %v", got)
	}
	if got := Requirements(nil); got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
