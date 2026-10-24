package archive

import (
	"reflect"
	"testing"

	"github.com/wake/docket/internal/delta"
)

func dreq(name string, scenarios ...string) delta.Requirement {
	r := delta.Requirement{Name: name}
	for _, s := range scenarios {
		r.Scenarios = append(r.Scenarios, delta.Scenario{Name: s})
	}
	return r
}

const showAll = `{"id":"x","deltas":[
 {"spec":"cap/a","operation":"ADDED","description":"d","requirement":{"name":"New","scenarios":[{"name":"s1"}]}},
 {"spec":"cap/a","operation":"MODIFIED","requirement":{"name":"Mod","scenarios":[]}},
 {"spec":"cap/a","operation":"REMOVED","requirement":{"name":"Old"}},
 {"spec":"cap/a","operation":"RENAMED","rename":{"from":"A","to":"B"}},
 {"spec":"cap/a","operation":"ADDED"}
]}`

func allDeltas() map[string]delta.Delta {
	return map[string]delta.Delta{"cap/a": {
		Added:    []delta.Requirement{dreq("New", "s1")},
		Modified: []delta.Requirement{dreq("Mod")},
		Removed:  []delta.Requirement{dreq("Old")},
		Renamed:  []delta.Rename{{From: "A", To: "B"}},
	}}
}

func TestCrossCheckAgrees(t *testing.T) {
	r, ok, err := CrossCheck(allDeltas(), showAll)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v %+v", ok, err, r)
	}
	want := []string{
		`cap/a: ADDED|New|["s1"]`,
		`cap/a: MODIFIED|Mod|[]`,
		`cap/a: REMOVED|Old|[]`,
		`cap/a: RENAMED|A|B`,
	}
	if !reflect.DeepEqual(r.Docket, want) || !reflect.DeepEqual(r.OpenSpec, want) {
		t.Errorf("got %#v", r)
	}
}

func TestReadingsDisagree(t *testing.T) {
	show := `{"deltas":[{"spec":"cap/a","operation":"ADDED","requirement":{"name":"New","scenarios":[{"name":"s1"}]}}]}`
	d := map[string]delta.Delta{"cap/a": {Added: []delta.Requirement{dreq("New", "s1"), dreq("Extra")}}}
	r, ok, err := CrossCheck(d, show)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if len(r.Docket) != 2 || len(r.OpenSpec) != 1 {
		t.Errorf("got %#v", r)
	}
}

func TestCrossCheckScenarioNames(t *testing.T) {
	show := `{"deltas":[{"spec":"c","operation":"ADDED","requirement":{"name":"R","scenarios":[{"name":"one"}]}}]}`
	d := map[string]delta.Delta{"c": {Added: []delta.Requirement{dreq("R", "two")}}}
	if _, ok, err := CrossCheck(d, show); err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestCrossCheckBadJSON(t *testing.T) {
	if _, _, err := CrossCheck(nil, "not json"); err == nil {
		t.Fatal("want error")
	}
}

func TestCrossCheckEmptyIsNotNil(t *testing.T) {
	r, ok, err := CrossCheck(nil, `{"deltas":[]}`)
	if err != nil || !ok || r.Docket == nil || r.OpenSpec == nil {
		t.Fatalf("ok=%v err=%v %#v", ok, err, r)
	}
}
