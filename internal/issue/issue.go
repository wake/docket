// Package issue holds what Docket knows about an issue: where it lives and its
// sections.
package issue

import "github.com/wake/docket/internal/record"

// Dir is the folder holding the issues, relative to the repository root.
const Dir = "docket/issues"

// Sections lists the sections of an issue, in order.
var Sections = []string{"Current", "Expected", "Unchanged", "Evidence"}

// BodyLimit is the most the sections of an issue may hold, in bytes (see change.Size).
const BodyLimit = record.BodyLimit

// Path is the issue file of slug, relative to the repository root.
func Path(slug string) string { return Dir + "/" + slug + ".md" }
