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
		"--format=%H%x00%s%x00%(trailers:key=Change,valueonly,unfold)%x00%(trailers:key=Task,valueonly,unfold)",
		base+".."+head, "--")
	if err != nil {
		return nil, err
	}
	// NUL cannot occur in commit data, so the stream is consecutive groups
	// of four NUL-separated fields.
	f := strings.Split(string(out), "\x00")
	if n := len(f); n > 0 && strings.TrimSpace(f[n-1]) == "" {
		f = f[:n-1]
	}
	if len(f)%4 != 0 {
		return nil, fmt.Errorf("git log: unexpected output of %d fields", len(f))
	}
	var commits []Commit
	for i := 0; i < len(f); i += 4 {
		commits = append(commits, Commit{
			SHA:     strings.TrimSpace(f[i]),
			Subject: f[i+1],
			Change:  splitValues(f[i+2]),
			Task:    splitValues(f[i+3]),
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
