package gitx

import (
	"fmt"
	"strings"
	"time"
)

// Commit is a commit with its committer time and the values of its Change
// and Task trailers.
type Commit struct {
	SHA, Subject string
	Time         time.Time
	Change, Task []string
}

// Commits lists the commits in base..head, oldest first, with their committer
// time and the values of their Change and Task trailers (keys are
// case-insensitive, as in git). With paths, only commits that change a file
// under them are listed; --full-history keeps a merge that brings such files
// in, which git would otherwise drop when it matches its second parent only.
func (g Git) Commits(base, head string, paths ...string) ([]Commit, error) {
	args := []string{"log", "-z", "--reverse", "--full-history",
		"--format=%H%x00%s%x00%cI%x00%(trailers:key=Change,valueonly,unfold)%x00%(trailers:key=Task,valueonly,unfold)",
		base + ".." + head, "--"}
	out, err := g.output(append(args, paths...)...)
	if err != nil {
		return nil, err
	}
	// NUL cannot occur in commit data, so the stream is consecutive groups
	// of five NUL-separated fields.
	f := strings.Split(string(out), "\x00")
	if n := len(f); n > 0 && strings.TrimSpace(f[n-1]) == "" {
		f = f[:n-1]
	}
	if len(f)%5 != 0 {
		return nil, fmt.Errorf("git log: unexpected output of %d fields", len(f))
	}
	var commits []Commit
	for i := 0; i < len(f); i += 5 {
		when, err := time.Parse(time.RFC3339, strings.TrimSpace(f[i+2]))
		if err != nil {
			return nil, fmt.Errorf("git log: commit time of %s: %w", strings.TrimSpace(f[i]), err)
		}
		commits = append(commits, Commit{
			SHA:     strings.TrimSpace(f[i]),
			Subject: f[i+1],
			Time:    when,
			Change:  splitValues(f[i+3]),
			Task:    splitValues(f[i+4]),
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
