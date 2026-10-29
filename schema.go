// Package docket embeds the assets the dkt command ships: the OpenSpec
// schema, the skill and the Docket block of AGENTS.md.
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

// Schema returns the files of openspec/schemas/docket, with paths relative
// to that directory (schema.yaml, templates/proposal.md, ...).
func Schema() fs.FS {
	sub, err := fs.Sub(schemaFS, "openspec/schemas/docket")
	if err != nil {
		panic(err)
	}
	return sub
}

//go:embed .agents/skills/docket
var skillFS embed.FS

// Skill returns the files of .agents/skills/docket, with paths relative to
// that directory (SKILL.md, ...).
func Skill() fs.FS {
	sub, err := fs.Sub(skillFS, ".agents/skills/docket")
	if err != nil {
		panic(err)
	}
	return sub
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
