// Package docket embeds the assets the dkt command ships: the OpenSpec
// schema and the routing section of AGENTS.md.
package docket

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed openspec/schemas/docket
var schemaFS embed.FS

//go:embed AGENTS.md
var agentsMD string

const agentsHeading = "## Change workflow (superpowers × OpenSpec)"

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

const (
	blockStart = "<!-- docket:start -->"
	blockEnd   = "<!-- docket:end -->"
)

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
	lines := strings.SplitAfter(agentsMD, "\n")
	start, end, offset := -1, -1, 0
	startOff, endOff := 0, 0
	for _, l := range lines {
		switch strings.TrimRight(l, "\r\n") {
		case blockStart:
			if start >= 0 {
				panic("AGENTS.md has more than one " + blockStart)
			}
			start, startOff = 1, offset
		case blockEnd:
			if start < 0 {
				panic("AGENTS.md has " + blockEnd + " before " + blockStart)
			}
			if end >= 0 {
				panic("AGENTS.md has more than one " + blockEnd)
			}
			end, endOff = 1, offset+len(strings.TrimRight(l, "\r\n"))
		}
		offset += len(l)
	}
	if start < 0 || end < 0 {
		panic("AGENTS.md has no complete docket block")
	}
	return agentsMD[startOff:endOff]
}

// AgentsSection returns the AGENTS.md section that starts at the change
// workflow heading and ends before the next "## " line, trailing whitespace
// trimmed. It panics if the heading is missing.
func AgentsSection() string {
	lines := strings.Split(agentsMD, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimRight(l, " \t\r") == agentsHeading {
			start = i
			break
		}
	}
	if start < 0 {
		panic("AGENTS.md has no section " + agentsHeading)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.TrimRight(strings.Join(lines[start:end], "\n"), " \t\r\n")
}
