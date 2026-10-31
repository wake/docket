package gitx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Commit is a commit with its committer time and the values of its Change,
// Task and Record trailers.
type Commit struct {
	SHA, Subject         string
	Time                 time.Time
	Change, Task, Record []string
}

// Commits lists the commits in base..head, oldest first, with their committer
// time (the instant comes from epoch seconds when the committer's zone is not
// valid RFC 3339, as git allows) and the values of their Change and Task
// trailers (keys are case-insensitive, as in git). With paths, only commits
// that change a file under them are listed; --full-history keeps a merge that
// brings such files in, which git would otherwise drop when it matches its
// second parent only.
func (g Git) Commits(base, head string, paths ...string) ([]Commit, error) {
	args := []string{"log", "-z", "--reverse", "--full-history",
		"--format=%H%x00%s%x00%cI%x00%ct%x00%(trailers:key=Change,valueonly,unfold)%x00%(trailers:key=Task,valueonly,unfold)%x00%(trailers:key=Record,valueonly,unfold)",
		base + ".." + head, "--"}
	out, err := g.output(append(args, paths...)...)
	if err != nil {
		return nil, err
	}
	// NUL cannot occur in commit data, so the stream is consecutive groups
	// of seven NUL-separated fields.
	f := strings.Split(string(out), "\x00")
	if n := len(f); n > 0 && strings.TrimSpace(f[n-1]) == "" {
		f = f[:n-1]
	}
	if len(f)%7 != 0 {
		return nil, fmt.Errorf("git log: unexpected output of %d fields", len(f))
	}
	var commits []Commit
	for i := 0; i < len(f); i += 7 {
		secs, err := strconv.ParseInt(strings.TrimSpace(f[i+3]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("git log: commit time of %s: %w", strings.TrimSpace(f[i]), err)
		}
		// %cI keeps the committer's offset, but git accepts a zone Go
		// rejects (+2500); the epoch seconds name the same instant.
		when, err := time.Parse(time.RFC3339, strings.TrimSpace(f[i+2]))
		if err != nil {
			when = time.Unix(secs, 0).UTC()
		}
		commits = append(commits, Commit{
			SHA:     strings.TrimSpace(f[i]),
			Subject: f[i+1],
			Time:    when,
			Change:  splitValues(f[i+4]),
			Task:    splitValues(f[i+5]),
			Record:  splitValues(f[i+6]),
		})
	}
	return commits, nil
}

// splitValues splits unfolded trailer values on newlines, trimming them and
// dropping empty ones.
func splitValues(s string) []string {
	var vals []string
	for _, v := range strings.Split(s, "\n") {
		if v = strings.TrimSpace(v); v != "" {
			vals = append(vals, v)
		}
	}
	return vals
}
