// Package archive closes changes: Sync ticks plan.md from commit trailers,
// Archive checks and archives a change, and Rebase moves a change's base.
package archive

import (
	"github.com/wake/docket/internal/gitx"
	"github.com/wake/docket/internal/scan"
)

// Finding is one problem found in a change. Change and File are empty and
// Line is 0 when they do not apply; File is relative to the repository root.
type Finding struct {
	Rule    string
	Change  string
	File    string
	Line    int
	Message string
}

// Unreadable is a source that was needed but could not be read.
type Unreadable struct {
	Source string
	Reason string
}

// Env is what the operations run against. Root is the repository root and
// Git runs there; Default is used by Archive and Rebase only.
type Env struct {
	Git     gitx.Git
	Root    string
	Default scan.DefaultBranch
}

// TaskTicks says how many steps of a task were ticked, or would be.
type TaskTicks struct {
	Task  int `json:"task"`
	Steps int `json:"steps"`
}

// SyncReport is the outcome of Sync.
type SyncReport struct {
	Change     string
	Ticked     []TaskTicks
	Findings   []Finding
	Unreadable []Unreadable
}
