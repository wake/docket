package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/wake/docket"
	"github.com/wake/docket/internal/archive"
	"github.com/wake/docket/internal/lint"
	"github.com/wake/docket/internal/openspec"
	"github.com/wake/docket/internal/scan"
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
			r, err := openspec.New(c.Repo.Root, c.Env)
			if err != nil {
				return Outcome{}, envOrErr(err)
			}
			if _, err := r.CheckVersion(); err != nil {
				return Outcome{}, envOrErr(err)
			}
			db, err := scan.ResolveDefaultBranch(c.Git, c.DefaultBranch)
			if err != nil {
				if errors.Is(err, scan.ErrNoDefaultBranch) {
					return Outcome{}, EnvError{Msg: err.Error()}
				}
				return Outcome{}, err
			}
			unlock, err := lockOpenSpec(c.Repo.Root)
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
			if err != nil {
				return Outcome{}, envOrErr(err)
			}
			out := Outcome{Result: ArchiveResult{
				Archived: append([]archive.Archived{}, rep.Archived...),
				Stopped:  rep.Stopped,
				Drifts:   append([]archive.Drift{}, rep.Drifts...),
				Readings: rep.Readings,
			}}
			addArchiveReport(&out, rep.Findings, rep.Unreadable)
			for _, a := range rep.Archived {
				out.Human = append(out.Human, fmt.Sprintf("archived %s as %s", a.Change, strings.TrimPrefix(a.Folder, "openspec/changes/archive/")))
			}
			if s := rep.Stopped; s != nil {
				if s.Change == "" {
					out.Human = append(out.Human, "stopped at "+s.Step)
				} else {
					out.Human = append(out.Human, fmt.Sprintf("stopped at %s for %s", s.Step, s.Change))
				}
			}
			for _, d := range rep.Drifts {
				out.Human = append(out.Human, fmt.Sprintf("%s / %s at x-base:", d.Capability, d.Requirement))
				out.Human = append(out.Human, indented(d.Base)...)
				out.Human = append(out.Human, "now:")
				out.Human = append(out.Human, indented(d.Now)...)
			}
			if rd := rep.Readings; rd != nil {
				out.Human = append(out.Human, "docket:")
				out.Human = append(out.Human, indented(strings.Join(rd.Docket, "\n"))...)
				out.Human = append(out.Human, "openspec:")
				out.Human = append(out.Human, indented(strings.Join(rd.OpenSpec, "\n"))...)
			}
			return out, nil
		},
	}
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
