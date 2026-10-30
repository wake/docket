package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/workflow"
)

// WorkflowResult is the result object of dkt workflow.
type WorkflowResult struct {
	Path            string          `json:"path"`
	Status          workflow.Status `json:"status"`
	Template        string          `json:"template"`
	Recorded        *string         `json:"recorded"`
	MissingSections []string        `json:"missing_sections"`
}

// TemplateResult is the result object of dkt workflow template.
type TemplateResult struct {
	Template string `json:"template"`
	Text     string `json:"text"`
}

func workflowCommand() Command {
	return Command{
		Name:  "workflow",
		Usage: "workflow [template]",
		Run: func(c *Context, args []string) (Outcome, error) {
			switch {
			case len(args) == 0:
				return workflowReport(c)
			case len(args) == 1 && args[0] == "template":
				text := docket.Template()
				lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
				return Outcome{Human: lines, Result: TemplateResult{Template: workflow.Hash(text), Text: text}}, nil
			}
			return Outcome{}, UsageError{Msg: "workflow takes no argument except template"}
		},
	}
}

func workflowReport(c *Context) (Outcome, error) {
	path, err := workflow.Locate(c.Env, c.UserHome)
	if err != nil {
		if errors.Is(err, workflow.ErrNoHome) {
			return Outcome{}, EnvError{Msg: err.Error()}
		}
		return Outcome{}, err
	}
	b, found, err := workflow.Read(path)
	if err != nil {
		return Outcome{Unreadable: []Unreadable{{Source: path, Reason: err.Error()}}}, nil
	}
	shipped := workflow.Hash(docket.Template())
	rep := workflow.Report{Status: workflow.Missing, MissingSections: []string{}}
	if found {
		rep = workflow.Inspect(b, shipped)
	}
	var out Outcome
	switch rep.Status {
	case workflow.Missing:
		out.Findings = append(out.Findings, Finding{Rule: "workflow-missing", File: path,
			Message: "no workflow file; the docket skill runs setup to write it"})
	case workflow.Incomplete:
		for _, s := range rep.MissingSections {
			out.Findings = append(out.Findings, Finding{Rule: "workflow-incomplete", File: path,
				Message: fmt.Sprintf("no %q section; the docket skill runs setup to add it", "## "+s)})
		}
	case workflow.Stale:
		msg := "no template hash recorded; the shipped template is " + shipped
		if rep.Recorded != nil {
			msg = fmt.Sprintf("made from template %s; the shipped template is %s", *rep.Recorded, shipped)
		}
		out.Findings = append(out.Findings, Finding{Rule: "workflow-stale", File: path, Message: msg})
	}
	out.Human = []string{fmt.Sprintf("%-10s %s", rep.Status, path)}
	out.Result = WorkflowResult{Path: path, Status: rep.Status, Template: shipped,
		Recorded: rep.Recorded, MissingSections: rep.MissingSections}
	return out, nil
}
