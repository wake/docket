package archive

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/wake/docket/internal/delta"
)

// Readings are the two line lists CrossCheck compares: Docket's reading of a
// change's delta specs, and OpenSpec's reading of the same change.
type Readings struct {
	Docket   []string `json:"docket"`
	OpenSpec []string `json:"openspec"`
}

// osDelta is one entry of `openspec show --type change --json
// --deltas-only`'s deltas array; unknown fields are ignored.
type osDelta struct {
	Spec        string `json:"spec"`
	Operation   string `json:"operation"`
	Requirement *struct {
		Name      string `json:"name"`
		Scenarios []struct {
			Name string `json:"name"`
		} `json:"scenarios"`
	} `json:"requirement"`
	Rename *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rename"`
}

// osReport is the part of that output CrossCheck reads. Deltas is nil when
// the key is absent or null.
type osReport struct {
	Deltas []osDelta `json:"deltas"`
}

// Every field is quoted, so a name with a bar or a colon cannot make two
// different readings the same line.
func reqLine(capability, op, name string, scenarios []string) string {
	return fmt.Sprintf("%q: %s|%q|%q", capability, op, name, scenarios)
}

func renameLine(capability, from, to string) string {
	return fmt.Sprintf("%q: RENAMED|%q|%q", capability, from, to)
}

// check reports why d is not an entry OpenSpec's output can contain: a
// spec, a known operation, and the one payload that fits it.
func (d osDelta) check() error {
	if d.Spec == "" {
		return fmt.Errorf("no spec")
	}
	switch d.Operation {
	case "RENAMED":
		switch {
		case d.Requirement != nil:
			return fmt.Errorf("RENAMED carries a requirement")
		case d.Rename == nil || d.Rename.From == "" || d.Rename.To == "":
			return fmt.Errorf("RENAMED needs a rename with from and to")
		}
	case "ADDED", "MODIFIED", "REMOVED":
		switch {
		case d.Rename != nil:
			return fmt.Errorf("%s carries a rename", d.Operation)
		case d.Requirement == nil || d.Requirement.Name == "":
			return fmt.Errorf("%s needs a requirement with a name", d.Operation)
		}
	default:
		return fmt.Errorf("unknown operation %q", d.Operation)
	}
	return nil
}

// docketLines is Docket's reading of deltas, keyed by capability path.
func docketLines(deltas map[string]delta.Delta) []string {
	lines := []string{}
	caps := make([]string, 0, len(deltas))
	for c := range deltas {
		caps = append(caps, c)
	}
	sort.Strings(caps)
	for _, c := range caps {
		d := deltas[c]
		add := func(op string, rs []delta.Requirement, withScenarios bool) {
			for _, q := range rs {
				names := []string{}
				if withScenarios {
					for _, s := range q.Scenarios {
						names = append(names, s.Name)
					}
				}
				lines = append(lines, reqLine(c, op, q.Name, names))
			}
		}
		add("ADDED", d.Added, true)
		add("MODIFIED", d.Modified, true)
		add("REMOVED", d.Removed, false)
		for _, rn := range d.Renamed {
			lines = append(lines, renameLine(c, rn.From, rn.To))
		}
	}
	return lines
}

// CrossCheck compares Docket's reading of a change's delta specs (deltas,
// keyed by capability path) with show, the output of `openspec show <slug>
// --type change --json --deltas-only`. The bool is true when both readings
// produce the same lines. The error is for output that is not that JSON
// shape; Readings.Docket is still Docket's reading then and OpenSpec is
// empty.
func CrossCheck(deltas map[string]delta.Delta, show string) (Readings, bool, error) {
	r := Readings{Docket: docketLines(deltas), OpenSpec: []string{}}
	var rep osReport
	if err := json.Unmarshal([]byte(show), &rep); err != nil {
		return r, false, fmt.Errorf("parse OpenSpec output: %w", err)
	}
	if rep.Deltas == nil {
		return r, false, fmt.Errorf("OpenSpec output has no deltas array")
	}

	bySpec := map[string][]string{}
	for i, d := range rep.Deltas {
		if err := d.check(); err != nil {
			return r, false, fmt.Errorf("OpenSpec output deltas[%d]: %w", i, err)
		}
		if d.Rename != nil {
			bySpec[d.Spec] = append(bySpec[d.Spec], renameLine(d.Spec, d.Rename.From, d.Rename.To))
			continue
		}
		names := []string{}
		for _, s := range d.Requirement.Scenarios {
			names = append(names, s.Name)
		}
		bySpec[d.Spec] = append(bySpec[d.Spec], reqLine(d.Spec, d.Operation, d.Requirement.Name, names))
	}
	specs := make([]string, 0, len(bySpec))
	for s := range bySpec {
		specs = append(specs, s)
	}
	sort.Strings(specs)
	for _, s := range specs {
		r.OpenSpec = append(r.OpenSpec, bySpec[s]...)
	}
	return r, reflect.DeepEqual(r.Docket, r.OpenSpec), nil
}
