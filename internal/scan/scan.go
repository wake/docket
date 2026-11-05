// Package scan finds the changes in every worktree and branch of a repository.
package scan

import "github.com/wake/docket/internal/gitx"

// Use is one change folder found in one source.
type Use struct {
	Slug     string `json:"slug"`
	Archived bool   `json:"archived"`
	Path     string `json:"path"`   // the folder, relative to the repository root, slash-separated
	Source   string `json:"source"` // a worktree's path, or a ref's full name
}

// Unreadable is a source the scan could not read.
type Unreadable struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Inventory is what a scan found.
type Inventory struct {
	Uses       []Use
	Unreadable []Unreadable
}

// Changes scans every worktree, local branch and remote-tracking branch.
func Changes(g gitx.Git) (Inventory, error) { return Inventory{}, nil }
