package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/archive"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/openspec"
)

// ArchiveResult is the result object of dkt archive.
type ArchiveResult struct {
	Archived []archive.Archived `json:"archived"`
	Stopped  *archive.Stop      `json:"stopped"`
	Drifts   []archive.Drift    `json:"drifts"`
	Readings *archive.Readings  `json:"readings"`
}

func archiveCommand() Command {
	return Command{
		Name:  "archive",
		Usage: "archive <slug>...",
		Run: func(c *Context, args []string) (Outcome, error) {
			if len(args) == 0 {
				return Outcome{}, UsageError{Msg: "archive needs at least one slug"}
			}
			seen := map[string]bool{}
			for _, s := range args {
				if seen[s] {
					return Outcome{}, UsageError{Msg: fmt.Sprintf("slug %q is given twice", s)}
				}
				seen[s] = true
			}
			r, err := openspec.New(c.Repo.Root, c.Env)
			if err != nil {
				return Outcome{}, envOrErr(err)
			}
			if _, err := r.CheckVersion(); err != nil {
				return Outcome{}, envOrErr(err)
			}
			db, unlock, err := lockWithDefaultBranch(c)
			if err != nil {
				return Outcome{}, err
			}
			defer unlock()
			root, err := os.OpenRoot(c.Repo.Root)
			if err != nil {
				return Outcome{}, err
			}
			defer root.Close()
			added, addedErr := addedFiles(c)
			origin := ""
			if c.Repo.Origin != nil {
				origin = *c.Repo.Origin
			}
			rep, err := archive.Archive(archive.Input{
				Env:      archive.Env{Git: c.Git, Root: c.Repo.Root, Default: db},
				OpenSpec: r,
				Lint: func(slug string) lint.Report {
					return lint.Run(lint.Input{Repo: root.FS(), Templates: docket.Schema(),
						Slugs: []string{slug}, Added: added, AddedErr: addedErr})
				},
				Version: Version,
				Origin:  origin,
				Slugs:   args,
			})
			out := archiveOutcome(rep)
			if err != nil {
				// Keep what was archived before the failure.
				return out, envOrErr(err)
			}
			return out, nil
		},
	}
}

// archiveOutcome builds the result object, findings and text lines of a
// report.
func archiveOutcome(rep archive.ArchiveReport) Outcome {
	out := Outcome{Result: ArchiveResult{
		Archived: append([]archive.Archived{}, rep.Archived...),
		Stopped:  rep.Stopped,
		Drifts:   append([]archive.Drift{}, rep.Drifts...),
		Readings: rep.Readings,
	}}
	addArchiveReport(&out, rep.Findings, rep.Unreadable)
	out.Human = append(out.Human, archiveHuman(rep)...)
	return out
}

// archiveHuman builds the text-mode lines for a report.
func archiveHuman(rep archive.ArchiveReport) []string {
	var h []string
	for _, a := range rep.Archived {
		h = append(h, fmt.Sprintf("archived %s as %s", a.Change, strings.TrimPrefix(a.Folder, "openspec/changes/archive/")))
	}
	if s := rep.Stopped; s != nil {
		if s.Change == "" {
			h = append(h, "stopped at "+s.Step)
		} else {
			h = append(h, fmt.Sprintf("stopped at %s for %s", s.Step, s.Change))
		}
	}
	for _, d := range rep.Drifts {
		h = append(h, fmt.Sprintf("%s / %s at x-base:", d.Capability, d.Requirement))
		h = append(h, indented(d.Base)...)
		h = append(h, "now:")
		h = append(h, indented(d.Now)...)
	}
	if rd := rep.Readings; rd != nil {
		h = append(h, "docket:")
		h = append(h, indented(strings.Join(rd.Docket, "\n"))...)
		h = append(h, "openspec:")
		h = append(h, indented(strings.Join(rd.OpenSpec, "\n"))...)
	}
	return h
}

// indented returns the lines of s indented by two spaces, or "  (absent)"
// when s is empty.
func indented(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{"  (absent)"}
	}
	var out []string
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		out = append(out, "  "+l)
	}
	return out
}
