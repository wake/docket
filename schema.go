// Package docket embeds the assets the dkt command ships: the OpenSpec
// schema, the skills, the Docket block of AGENTS.md and the workflow
// template.
package docket

import (
	"embed"
	"io/fs"

	"github.com/wake/docket/internal/setup"
)

//go:embed openspec/schemas/docket
var schemaFS embed.FS

//go:embed AGENTS.md
var agentsMD string

//go:embed workflow/template.md
var workflowTemplate string

// Template returns the workflow template, workflow/template.md: the review
// and verification steps for each path, worded in capabilities.
func Template() string { return workflowTemplate }

//go:embed record/template.md
var recordTemplate string

// RecordTemplate returns the body of a new record, record/template.md: the
// four sections, each holding a comment that says what to write.
func RecordTemplate() string { return recordTemplate }

// Schema returns the files of openspec/schemas/docket, with paths relative
// to that directory (schema.yaml, templates/proposal.md, ...).
func Schema() fs.FS {
	sub, err := fs.Sub(schemaFS, "openspec/schemas/docket")
	if err != nil {
		panic(err)
	}
	return sub
}

//go:embed .agents/skills/docket .agents/skills/docket-setup .agents/skills/docket-doctor
var skillFS embed.FS

// Skills returns the skills under .agents/skills, each with paths relative
// to its folder (SKILL.md, ...).
func Skills() []setup.Skill {
	var out []setup.Skill
	for _, name := range []string{"docket", "docket-setup", "docket-doctor"} {
		sub, err := fs.Sub(skillFS, ".agents/skills/"+name)
		if err != nil {
			panic(err)
		}
		out = append(out, setup.Skill{Name: name, Files: sub})
	}
	return out
}

// Block returns the Docket block of AGENTS.md: from the start of the line
// holding the start marker through the end of the end marker, without the
// newline after it. It panics unless AGENTS.md holds exactly one block.
func Block() string {
	from, to, found, err := setup.FindBlock([]byte(agentsMD))
	if err != nil {
		panic("AGENTS.md: " + err.Error())
	}
	if !found {
		panic("AGENTS.md has no docket block")
	}
	return agentsMD[from:to]
}
