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

// osReport is the part of `openspec show --type change --json --deltas-only`
// that CrossCheck reads; unknown fields are ignored.
type osReport struct {
	Deltas []struct {
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
	} `json:"deltas"`
}

func reqLine(capability, op, name string, scenarios []string) string {
	return fmt.Sprintf("%s: %s|%s|%q", capability, op, name, scenarios)
}

// CrossCheck compares Docket's reading of a change's delta specs (deltas,
// keyed by capability path) with show, the output of `openspec show <slug>
// --type change --json --deltas-only`. The bool is true when both readings
// produce the same lines; the error is for output that is not that JSON.
func CrossCheck(deltas map[string]delta.Delta, show string) (Readings, bool, error) {
	var rep osReport
	if err := json.Unmarshal([]byte(show), &rep); err != nil {
		return Readings{}, false, fmt.Errorf("parse OpenSpec output: %w", err)
	}
	r := Readings{Docket: []string{}, OpenSpec: []string{}}

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
				r.Docket = append(r.Docket, reqLine(c, op, q.Name, names))
			}
		}
		add("ADDED", d.Added, true)
		add("MODIFIED", d.Modified, true)
		add("REMOVED", d.Removed, false)
		for _, rn := range d.Renamed {
			r.Docket = append(r.Docket, fmt.Sprintf("%s: RENAMED|%s|%s", c, rn.From, rn.To))
		}
	}

	bySpec := map[string][]string{}
	for _, d := range rep.Deltas {
		switch {
		case d.Rename != nil:
			bySpec[d.Spec] = append(bySpec[d.Spec], fmt.Sprintf("%s: %s|%s|%s", d.Spec, d.Operation, d.Rename.From, d.Rename.To))
		case d.Requirement != nil:
			names := []string{}
			for _, s := range d.Requirement.Scenarios {
				names = append(names, s.Name)
			}
			bySpec[d.Spec] = append(bySpec[d.Spec], reqLine(d.Spec, d.Operation, d.Requirement.Name, names))
		}
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
