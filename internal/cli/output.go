package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

type envelope struct {
	Tool       string       `json:"tool"`
	Version    string       `json:"version"`
	Command    string       `json:"command"`
	Repo       *Repo        `json:"repo"`
	OK         bool         `json:"ok"`
	Findings   []Finding    `json:"findings"`
	Unreadable []Unreadable `json:"unreadable"`
	Result     any          `json:"result"`
}

// errorFinding turns a command error into a finding.
func errorFinding(err error) Finding {
	rule := "internal"
	switch {
	case asUsage(err):
		rule = "usage"
	case asEnv(err):
		rule = "environment"
	}
	return Finding{Rule: rule, Message: err.Error()}
}

func writeJSON(w io.Writer, name string, repo *Repo, o Outcome, code int) error {
	e := envelope{
		Tool: "dkt", Version: Version, Command: name, Repo: repo,
		OK: code == 0, Findings: o.Findings, Unreadable: o.Unreadable, Result: o.Result,
	}
	if e.Findings == nil {
		e.Findings = []Finding{}
	}
	if e.Unreadable == nil {
		e.Unreadable = []Unreadable{}
	}
	b, err := json.Marshal(e)
	if err != nil {
		// Keep stdout one JSON object: report the failure in the envelope.
		cause := err
		e.OK, e.Result = false, nil
		e.Findings = append(append([]Finding{}, e.Findings...), Finding{Rule: "internal", Message: cause.Error()})
		if b, err = json.Marshal(e); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(w, "%s\n", b); err != nil {
			return err
		}
		return fmt.Errorf("encoding result: %w", cause)
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

func writeText(w io.Writer, name string, o Outcome, code int) error {
	for _, l := range o.Human {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	for _, f := range o.Findings {
		line := f.Rule + ": " + f.Message
		if f.File != "" {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			line += " (" + loc + ")"
		}
		if f.Change != "" {
			line += " [" + f.Change + "]"
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	for _, u := range o.Unreadable {
		if _, err := fmt.Fprintf(w, "unreadable: %s: %s\n", u.Source, u.Reason); err != nil {
			return err
		}
	}
	if code == 0 {
		_, err := fmt.Fprintf(w, "dkt %s: ok\n", name)
		return err
	}
	return nil
}
