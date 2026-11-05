package openspec

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Delta is one change to a capability in a change, as `openspec show` reports it.
type Delta struct {
	Spec, Operation string
	Requirements    []Requirement
	Rename          *Rename // set for RENAMED
}

// Rename is a requirement renamed by a delta.
type Rename struct{ From, To string }

// Requirement is a requirement with its scenario names.
type Requirement struct {
	Name      string
	Scenarios []Scenario
}

// Scenario is a scenario of a requirement.
type Scenario struct{ Name string }

// Issue is one finding of `openspec validate`; Line is 0 when OpenSpec gives none.
type Issue struct {
	Level, Path, Message string
	Line                 int
}

type wireReq struct {
	Name      string `json:"name"`
	Scenarios []struct {
		Name string `json:"name"`
	} `json:"scenarios"`
}

func (w wireReq) req() Requirement {
	r := Requirement{Name: w.Name}
	for _, s := range w.Scenarios {
		r.Scenarios = append(r.Scenarios, Scenario{Name: s.Name})
	}
	return r
}

func reqs(ws []wireReq) []Requirement {
	var out []Requirement
	for _, w := range ws {
		out = append(out, w.req())
	}
	return out
}

// runJSON runs args and, on a FailedError, fills in Messages from a {"status":[...]} on stdout.
func (r *Runner) runJSON(args ...string) (string, error) {
	out, err := r.Run(args...)
	var fe *FailedError
	if errors.As(err, &fe) {
		var st struct {
			Status []struct {
				Message string `json:"message"`
			} `json:"status"`
		}
		if json.Unmarshal([]byte(out), &st) == nil {
			for _, s := range st.Status {
				fe.Messages = append(fe.Messages, s.Message)
			}
		}
	}
	return out, err
}

func decodeErr(args []string, err error) error {
	return fmt.Errorf("openspec %s: cannot read its JSON: %w", strings.Join(args, " "), err)
}

// ShowChange returns the deltas of the change slug.
func (r *Runner) ShowChange(slug string) ([]Delta, error) {
	args := []string{"show", slug, "--type", "change", "--json", "--deltas-only"}
	out, err := r.runJSON(args...)
	if err != nil {
		return nil, err
	}
	var w struct {
		Deltas []struct {
			Spec         string    `json:"spec"`
			Operation    string    `json:"operation"`
			Requirements []wireReq `json:"requirements"`
			Rename       *struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"rename"`
		} `json:"deltas"`
	}
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		return nil, decodeErr(args, err)
	}
	if w.Deltas == nil {
		return nil, decodeErr(args, errors.New("no deltas"))
	}
	var ds []Delta
	for _, d := range w.Deltas {
		x := Delta{Spec: d.Spec, Operation: d.Operation, Requirements: reqs(d.Requirements)}
		if d.Rename != nil {
			x.Rename = &Rename{From: d.Rename.From, To: d.Rename.To}
		}
		ds = append(ds, x)
	}
	return ds, nil
}

// ShowSpec returns the requirements of the capability.
func (r *Runner) ShowSpec(capability string) ([]Requirement, error) {
	args := []string{"show", capability, "--type", "spec", "--json"}
	out, err := r.runJSON(args...)
	if err != nil {
		return nil, err
	}
	var w struct {
		Requirements []wireReq `json:"requirements"`
	}
	if err := json.Unmarshal([]byte(out), &w); err != nil {
		return nil, decodeErr(args, err)
	}
	if w.Requirements == nil {
		return nil, decodeErr(args, errors.New("no requirements"))
	}
	return reqs(w.Requirements), nil
}

// Validate returns the issues OpenSpec finds in the change slug, whatever its exit status
// when it printed a report.
func (r *Runner) Validate(slug string) ([]Issue, error) {
	args := []string{"validate", slug, "--type", "change", "--json"}
	out, err := r.runJSON(args...)
	var w struct {
		Items []struct {
			ID     string `json:"id"`
			Valid  *bool  `json:"valid"`
			Issues []struct {
				Level   string `json:"level"`
				Path    string `json:"path"`
				Message string `json:"message"`
				Line    int    `json:"line"`
			} `json:"issues"`
		} `json:"items"`
	}
	if json.Unmarshal([]byte(out), &w) != nil || w.Items == nil {
		if err == nil {
			err = fmt.Errorf("openspec %s: printed no validation report", strings.Join(args, " "))
		}
		return nil, err
	}
	for _, it := range w.Items {
		if it.ID != slug {
			continue
		}
		var issues []Issue
		for _, i := range it.Issues {
			issues = append(issues, Issue{Level: i.Level, Path: i.Path, Message: i.Message, Line: i.Line})
		}
		if it.Valid != nil && !*it.Valid && !hasError(issues) {
			if err == nil {
				err = fmt.Errorf("openspec %s: the report marks %q invalid but lists no error", strings.Join(args, " "), slug)
			}
			return nil, err
		}
		return issues, nil
	}
	if err == nil {
		err = fmt.Errorf("openspec %s: the report has no entry for %q", strings.Join(args, " "), slug)
	}
	return nil, err
}

func hasError(issues []Issue) bool {
	for _, i := range issues {
		if strings.EqualFold(i.Level, "ERROR") {
			return true
		}
	}
	return false
}
