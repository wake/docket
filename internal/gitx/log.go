package gitx

import (
	"fmt"
	"strings"
)

// Commit is a commit with the values of its Change and Task trailers.
type Commit struct {
	SHA, Subject string
	Change, Task []string
}

// Commits lists the commits in base..head, oldest first, with the values of
// their Change and Task trailers (keys are case-insensitive, as in git).
func (g Git) Commits(base, head string) ([]Commit, error) {
	out, err := g.output("log", "-z", "--reverse",
		"--format=%H%x1f%s%x1f%(trailers:key=Change,valueonly,separator=%x1e)%x1f%(trailers:key=Task,valueonly,separator=%x1e)",
		base+".."+head, "--")
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, rec := range strings.Split(string(out), "\x00") {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		f := strings.Split(rec, "\x1f")
		if len(f) != 4 {
			return nil, fmt.Errorf("git log: unexpected record %q", rec)
		}
		commits = append(commits, Commit{
			SHA:     strings.TrimSpace(f[0]),
			Subject: f[1],
			Change:  splitValues(f[2]),
			Task:    splitValues(f[3]),
		})
	}
	return commits, nil
}

// splitValues splits trailer values on RS, trimming them and dropping empty ones.
func splitValues(s string) []string {
	var vals []string
	for _, v := range strings.Split(s, "\x1e") {
		if v = strings.TrimSpace(v); v != "" {
			vals = append(vals, v)
		}
	}
	return vals
}
